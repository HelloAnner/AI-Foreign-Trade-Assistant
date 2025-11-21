package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Router wires all HTTP routes.
func Router(h *Handlers) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json"))

	r.Route("/api", func(api chi.Router) {
		api.Post("/auth/login", h.Login)
		api.Get("/auth/public-key", h.PublicKey)

		api.Group(func(priv chi.Router) {
			if h != nil && h.Auth != nil {
				priv.Use(h.Auth.Middleware())
			}

			priv.Get("/health", h.Health)
			priv.Get("/settings", h.GetSettings)
			priv.Put("/settings", h.SaveSettings)
			priv.Post("/settings/test-llm", h.TestLLM)
			priv.Post("/settings/test-smtp", h.TestSMTP)
			priv.Post("/settings/test-search", h.TestSearch)

			priv.Post("/todos", h.EnqueueTodo)

			priv.Get("/customers", h.ListCustomers)
			priv.Get("/customers/{id}", h.GetCustomerDetail)
			priv.Put("/customers/{id}/followup-flag", h.UpdateFollowupStatus)
			priv.Delete("/customers/{id}", h.DeleteCustomer)

			priv.Post("/companies/resolve", h.ResolveCompany)
			priv.Post("/companies", h.CreateCompany)
			priv.Post("/companies/{id}/automation", h.EnqueueAutomation)
			priv.Put("/companies/{id}", h.UpdateCompany)
			priv.Post("/companies/{id}/contacts", h.ReplaceContacts)
			priv.Post("/companies/{id}/grade/suggest", h.SuggestGrade)
			priv.Post("/companies/{id}/grade/confirm", h.ConfirmGrade)
			priv.Post("/companies/{id}/analysis", h.GenerateAnalysis)
			priv.Put("/companies/{id}/analysis", h.UpdateAnalysis)
			priv.Post("/companies/{id}/email-draft", h.GenerateEmailDraft)
			priv.Put("/emails/{id}", h.UpdateEmailDraft)
			priv.Post("/companies/{id}/followup/first-save", h.SaveFirstFollowup)
			priv.Post("/followups/schedule", h.ScheduleFollowup)
			priv.Get("/scheduled-tasks", h.ListScheduledTasks)
			priv.Post("/scheduled-tasks/{id}/run-now", h.RunTaskNow)
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, Response{OK: false, Error: "route not found"})
	})

	return r
}

// writeJSON centralizes response encoding with consistent envelope.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

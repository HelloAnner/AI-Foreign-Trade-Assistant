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
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json"))

	r.Route("/api", func(api chi.Router) {
		api.Get("/health", h.Health)
	api.Get("/settings", h.GetSettings)
	api.Put("/settings", h.SaveSettings)
	api.Post("/settings/test-llm", h.TestLLM)
	api.Post("/settings/test-smtp", h.TestSMTP)
	api.Post("/settings/test-search", h.TestSearch)

	api.Post("/companies/resolve", h.ResolveCompany)
	api.Post("/companies", h.CreateCompany)
	api.Post("/companies/{id}/contacts", h.ReplaceContacts)
	api.Post("/companies/{id}/grade/suggest", h.SuggestGrade)
	api.Post("/companies/{id}/grade/confirm", h.ConfirmGrade)
	api.Post("/companies/{id}/analysis", h.GenerateAnalysis)
	api.Put("/companies/{id}/analysis", h.UpdateAnalysis)
	api.Post("/companies/{id}/email-draft", h.GenerateEmailDraft)
	api.Put("/emails/{id}", h.UpdateEmailDraft)
	api.Post("/companies/{id}/followup/first-save", h.SaveFirstFollowup)
	api.Post("/followups/schedule", h.ScheduleFollowup)
	api.Get("/scheduled-tasks", h.ListScheduledTasks)
	api.Post("/scheduled-tasks/{id}/run-now", h.RunTaskNow)
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

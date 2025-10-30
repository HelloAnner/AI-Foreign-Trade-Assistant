package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
	"github.com/anner/ai-foreign-trade-assistant/backend/services"
	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

// Response defines API envelope.
type Response struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// Handlers groups dependencies for HTTP handlers.
type Handlers struct {
	Store         *store.Store
	ServiceBundle *services.Bundle
}

// Health exposes a simple liveness probe.
func (h *Handlers) Health(w http.ResponseWriter, _ *http.Request) {
	if h == nil || h.Store == nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: "store not initialized"})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]string{"status": "ok"}})
}

// GetSettings returns persisted settings.
func (h *Handlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.Store.GetSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: settings})
}

// SaveSettings persists the settings payload.
func (h *Handlers) SaveSettings(w http.ResponseWriter, r *http.Request) {
	if err := h.Store.SaveSettings(r.Context(), r.Body); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true})
}

// TestLLM validates the configured LLM credentials.
func (h *Handlers) TestLLM(w http.ResponseWriter, r *http.Request) {
	result, err := h.ServiceBundle.LLM.TestConnection(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: result})
}

// TestSMTP sends a test email using the configured SMTP credentials.
func (h *Handlers) TestSMTP(w http.ResponseWriter, r *http.Request) {
	if err := h.ServiceBundle.Mailer.SendTest(r.Context()); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true})
}

// TestSearch verifies search provider availability.
func (h *Handlers) TestSearch(w http.ResponseWriter, r *http.Request) {
	if err := h.ServiceBundle.Search.TestSearch(r.Context()); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true})
}

// ResolveCompany triggers the multi-source enrichment pipeline.
func (h *Handlers) ResolveCompany(w http.ResponseWriter, r *http.Request) {
	var req domain.ResolveCompanyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	result, err := h.ServiceBundle.Enricher.ResolveCompany(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: result})
}

// CreateCompany saves Step 1 curated information.
func (h *Handlers) CreateCompany(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateCompanyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	id, err := h.Store.CreateCustomer(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]int64{"customer_id": id}})
}

// ReplaceContacts stores user-edited contacts for a customer.
func (h *Handlers) ReplaceContacts(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	var payload struct {
		Contacts []domain.Contact `json:"contacts"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if err := h.Store.ReplaceContacts(r.Context(), customerID, payload.Contacts); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true})
}

// SuggestGrade performs AI-driven grading.
func (h *Handlers) SuggestGrade(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	resp, err := h.ServiceBundle.Grader.Suggest(r.Context(), customerID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: resp})
}

// ConfirmGrade persists the user's grade decision.
func (h *Handlers) ConfirmGrade(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	var req domain.GradeConfirmRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if err := h.ServiceBundle.Grader.Confirm(r.Context(), customerID, req.Grade, req.Reason); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true})
}

// GenerateAnalysis produces entry-point suggestions.
func (h *Handlers) GenerateAnalysis(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	resp, err := h.ServiceBundle.Analyst.Generate(r.Context(), customerID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: resp})
}

// UpdateAnalysis allows saving manual edits to the analysis report.
func (h *Handlers) UpdateAnalysis(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	var content domain.AnalysisContent
	if err := decodeJSON(r, &content); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	analysisID, err := h.Store.SaveAnalysis(r.Context(), customerID, content)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]int64{"analysis_id": analysisID}})
}

// GenerateEmailDraft drafts the initial outreach mail.
func (h *Handlers) GenerateEmailDraft(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	resp, err := h.ServiceBundle.EmailComposer.DraftInitial(r.Context(), customerID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: resp})
}

// UpdateEmailDraft updates subject and body for an existing draft email.
func (h *Handlers) UpdateEmailDraft(w http.ResponseWriter, r *http.Request) {
	emailID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	var payload domain.EmailDraft
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if err := h.Store.UpdateEmailDraft(r.Context(), emailID, payload); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true})
}

// SaveFirstFollowup persists the first follow-up record referencing the draft email.
func (h *Handlers) SaveFirstFollowup(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	var req domain.FirstFollowupRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	followupID, err := h.Store.SaveInitialFollowup(r.Context(), customerID, req.EmailID, req.Notes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]int64{"followup_id": followupID}})
}

// ScheduleFollowup enqueues an automated follow-up task.
func (h *Handlers) ScheduleFollowup(w http.ResponseWriter, r *http.Request) {
	var req domain.ScheduleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	resp, err := h.ServiceBundle.Scheduler.Schedule(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: resp})
}

// ListScheduledTasks returns tasks filtered by status.
func (h *Handlers) ListScheduledTasks(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	tasks, err := h.Store.ListScheduledTasks(r.Context(), status)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: tasks})
}

// RunTaskNow executes a scheduled task immediately via API.
func (h *Handlers) RunTaskNow(w http.ResponseWriter, r *http.Request) {
	taskID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if err := h.ServiceBundle.Scheduler.RunNow(r.Context(), taskID); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true})
}

func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	return nil
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("无效的 ID")
	}
	return id, nil
}

package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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

// ListCustomers returns the customer summaries for management UI.
func (h *Handlers) ListCustomers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := store.CustomerListFilter{
		Grade:   strings.TrimSpace(query.Get("grade")),
		Country: strings.TrimSpace(query.Get("country")),
		Search:  strings.TrimSpace(query.Get("q")),
		Sort:    strings.TrimSpace(query.Get("sort")),
		Status:  strings.TrimSpace(query.Get("status")),
	}

	if limitStr := strings.TrimSpace(query.Get("limit")); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if offsetStr := strings.TrimSpace(query.Get("offset")); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	customers, err := h.Store.ListCustomers(r.Context(), filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: customers})
}

// GetCustomerDetail exposes all persisted information for a customer.
func (h *Handlers) GetCustomerDetail(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	detail, err := h.Store.GetCustomerDetail(r.Context(), customerID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: detail})
}

// SaveSettings persists the settings payload.
func (h *Handlers) SaveSettings(w http.ResponseWriter, r *http.Request) {
	if err := h.Store.SaveSettings(r.Context(), r.Body); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	settings, err := h.Store.GetSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, Response{OK: true})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: settings})
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
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]string{"message": "搜索 API 测试成功"}})
}

// ResolveCompany triggers the multi-source enrichment pipeline.
func (h *Handlers) ResolveCompany(w http.ResponseWriter, r *http.Request) {
	var req domain.ResolveCompanyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}

	existing, contacts, err := h.Store.FindCustomerByQuery(r.Context(), req.Query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	if existing != nil {
		response := domain.ResolveCompanyResponse{
			CustomerID:  existing.ID,
			Name:        existing.Name,
			Website:     existing.Website,
			Country:     existing.Country,
			Contacts:    contacts,
			Summary:     existing.Summary,
			Grade:       strings.ToUpper(strings.TrimSpace(existing.Grade)),
			GradeReason: existing.GradeReason,
		}

		lastStep := 1
		if response.Grade != "" && response.Grade != "UNKNOWN" {
			lastStep = maxInt(lastStep, 2)
			if response.Grade != "A" {
				lastStep = maxInt(lastStep, 5)
			}
		}

		if analysis, err := h.Store.GetLatestAnalysis(r.Context(), existing.ID); err == nil {
			response.Analysis = analysis
			lastStep = maxInt(lastStep, 4)
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
			return
		}

		if emailDraft, err := h.Store.GetLatestEmailDraft(r.Context(), existing.ID, "initial"); err == nil {
			response.EmailDraft = emailDraft
			lastStep = maxInt(lastStep, 5)
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
			return
		}

		if followupID, err := h.Store.GetLatestFollowupID(r.Context(), existing.ID); err == nil && followupID > 0 {
			response.FollowupID = followupID
			lastStep = maxInt(lastStep, 5)
		} else if err != nil {
			writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
			return
		}

		if task, err := h.Store.GetLatestScheduledTask(r.Context(), existing.ID); err == nil && task != nil {
			response.ScheduledTask = task
			lastStep = maxInt(lastStep, 5)
		} else if err != nil {
			writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
			return
		}

		if lastStep <= 0 {
			lastStep = 1
		}
		response.LastStep = lastStep
		writeJSON(w, http.StatusOK, Response{OK: true, Data: response})
		return
	}

	result, err := h.ServiceBundle.Enricher.ResolveCompany(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	trimmed := strings.TrimSpace(req.Query)
	if result != nil && result.Name == "" && trimmed != "" {
		result.Name = trimmed
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

// UpdateCompany updates Step 1 information for an existing customer.
func (h *Handlers) UpdateCompany(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	var req domain.CreateCompanyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if err := h.Store.UpdateCustomer(r.Context(), customerID, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]int64{"customer_id": customerID}})
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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

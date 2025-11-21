package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

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
	Auth          *AuthManager
}

// Health exposes a simple liveness probe.
func (h *Handlers) Health(w http.ResponseWriter, _ *http.Request) {
	if h == nil || h.Store == nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: "store not initialized"})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]string{"status": "ok"}})
}

// Login 处理前端登录请求。
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Auth == nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: "鉴权模块未就绪"})
		return
	}
	var payload struct {
		Cipher string `json:"cipher"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "请求体格式错误"})
		return
	}
	token, expiresAt, err := h.Auth.IssueToken(payload.Cipher)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, Response{OK: false, Error: err.Error()})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     tokenCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		SameSite: http.SameSiteLaxMode,
	})
	writeJSON(w, http.StatusOK, Response{
		OK: true,
		Data: map[string]interface{}{
			"token":      token,
			"expires_at": expiresAt.UTC().Format(time.RFC3339),
		},
	})
}

// PublicKey 返回 RSA 公钥。
func (h *Handlers) PublicKey(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Auth == nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: "鉴权模块未就绪"})
		return
	}
	info := h.Auth.PublicKey()
	if info == nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: "公钥不可用"})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: info})
}

// GetSettings returns persisted settings.
func (h *Handlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.Store.GetSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: sanitizeSettings(h.Auth, settings)})
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

	result, err := h.Store.ListCustomers(r.Context(), filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: result})
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

// DeleteCustomer removes a customer and cascaded data.
func (h *Handlers) DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if err := h.Store.DeleteCustomer(r.Context(), customerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, Response{OK: false, Error: "客户不存在"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true})
}

// UpdateFollowupStatus toggles whether automated followups can continue sending emails.
func (h *Handlers) UpdateFollowupStatus(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	var payload struct {
		FollowupSent bool `json:"followup_sent"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if err := h.Store.UpdateFollowupSent(r.Context(), customerID, payload.FollowupSent); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: map[string]bool{"followup_sent": payload.FollowupSent}})
}

// SaveSettings persists the settings payload.
func (h *Handlers) SaveSettings(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: fmt.Sprintf("读取请求失败: %v", err)})
		return
	}
	payload := bytes.TrimSpace(raw)
	if len(payload) == 0 {
		payload = raw
	}
	if h.Auth != nil && len(payload) > 0 {
		if payload, err = h.Auth.DecryptJSONFields(payload, nil); err != nil {
			writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
			return
		}
	}
	var incoming store.Settings
	if err := json.Unmarshal(payload, &incoming); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: fmt.Sprintf("解析配置失败: %v", err)})
		return
	}
	if err := h.handleLoginPasswordChange(r.Context(), &incoming); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	incoming.LoginPassword = ""
	if updated, err := json.Marshal(&incoming); err == nil {
		payload = updated
	}
	if err := h.Store.SaveSettings(r.Context(), bytes.NewReader(payload)); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	settings, err := h.Store.GetSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, Response{OK: true})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: sanitizeSettings(h.Auth, settings)})
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
	var overrides *store.Settings
	if r.Body != nil {
		var payload store.Settings
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			if !errors.Is(err, io.EOF) {
				writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
				return
			}
		} else {
			overrides = &payload
			if err := h.decryptSettingsStruct(overrides); err != nil {
				writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
				return
			}
		}
	}
	if err := h.ServiceBundle.Mailer.SendTest(r.Context(), overrides); err != nil {
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

// EnqueueTodo accepts a raw query and persists it for background processing.
func (h *Handlers) EnqueueTodo(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Query string `json:"query"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	q := strings.TrimSpace(payload.Query)
	if q == "" {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "请输入客户公司名称或官网地址"})
		return
	}
	if h.ServiceBundle == nil || h.ServiceBundle.Todo == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "待办服务未启用"})
		return
	}
	task, err := h.ServiceBundle.Todo.Enqueue(r.Context(), q)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: task})
}

// ResolveCompany triggers the multi-source enrichment pipeline.
func (h *Handlers) ResolveCompany(w http.ResponseWriter, r *http.Request) {
	var req domain.ResolveCompanyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}

	query := strings.TrimSpace(req.Query)
	if query == "" {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "请输入客户公司名称或官网地址"})
		return
	}
	log.Printf("[flow] resolve request query=%s", query)

	existing, contacts, err := h.Store.FindCustomerByQuery(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	if existing != nil {
		log.Printf("[flow] resolve hit existing customer_id=%d name=%s", existing.ID, strings.TrimSpace(existing.Name))
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

		if job, err := h.Store.GetLatestAutomationJob(r.Context(), existing.ID); err == nil && job != nil {
			response.AutomationJob = job
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

	req.Query = query
	log.Printf("[flow] resolve new query=%s entering enrichment", query)
	result, err := h.ServiceBundle.Enricher.ResolveCompany(r.Context(), &req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	trimmed := strings.TrimSpace(req.Query)
	if result != nil && result.Name == "" && trimmed != "" {
		result.Name = trimmed
	}
	// 在自动化开启时，直接在服务端保存解析结果，确保连续输入也能可靠入库。
	if h.ServiceBundle != nil && h.Store != nil {
		// 使用独立的后台上下文，避免 HTTP 请求上下文被取消导致保存失败。
		ctxSave, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		settings, sErr := h.Store.GetSettings(ctxSave)
		if sErr != nil {
			log.Printf("fetch settings for resolve auto-save failed: %v", sErr)
		} else if settings.AutomationEnabled {
			existsCustomer, _, ferr := h.Store.FindCustomerByQuery(ctxSave, trimmed)
			if ferr != nil {
				log.Printf("find customer before auto-save failed: %v", ferr)
			}
			if existsCustomer == nil {
				var src json.RawMessage
				if b, mErr := json.Marshal(result); mErr == nil {
					src = b
				}
				saveReq := &domain.CreateCompanyRequest{
					Name:       strings.TrimSpace(result.Name),
					Website:    strings.TrimSpace(result.Website),
					Country:    strings.TrimSpace(result.Country),
					Summary:    strings.TrimSpace(result.Summary),
					Contacts:   result.Contacts,
					SourceJSON: src,
				}
				if strings.TrimSpace(saveReq.Name) == "" {
					saveReq.Name = trimmed
				}
				if id, cErr := h.Store.CreateCustomer(ctxSave, saveReq); cErr != nil {
					log.Printf("[customers] auto-create failed name=%s website=%s: %v", strings.TrimSpace(saveReq.Name), strings.TrimSpace(saveReq.Website), cErr)
				} else if id > 0 {
					log.Printf("[customers] auto-created (resolve) id=%d name=%s", id, strings.TrimSpace(saveReq.Name))
					result.CustomerID = id
					if h.ServiceBundle.Automation != nil && settings.AutomationEnabled {
						if job, qErr := h.ServiceBundle.Automation.Enqueue(ctxSave, id); qErr != nil {
							if !errors.Is(qErr, services.ErrAutomationJobExists) {
								log.Printf("enqueue automation job (resolve) failed (customer %d): %v", id, qErr)
							} else if job != nil {
								result.AutomationJob = job
							}
						} else if job != nil {
							result.AutomationJob = job
						}
					}
				}
			} else {
				result.CustomerID = existsCustomer.ID
			}
		}
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
		log.Printf("[customers] create failed name=%s website=%s: %v", strings.TrimSpace(req.Name), strings.TrimSpace(req.Website), err)
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}

	payload := map[string]interface{}{"customer_id": id}

	if h.ServiceBundle != nil && h.ServiceBundle.Automation != nil {
		settings, err := h.Store.GetSettings(r.Context())
		if err != nil {
			log.Printf("fetch settings for automation failed: %v", err)
		} else if settings.AutomationEnabled {
			job, err := h.ServiceBundle.Automation.Enqueue(r.Context(), id)
			if err != nil {
				if errors.Is(err, services.ErrAutomationJobExists) {
					if job != nil {
						payload["automation_job"] = job
					}
				} else {
					log.Printf("enqueue automation job failed (customer %d): %v", id, err)
				}
			} else if job != nil {
				payload["automation_job"] = job
			}
		}
	}

	log.Printf("[customers] created customer id=%d name=%s", id, strings.TrimSpace(req.Name))
	writeJSON(w, http.StatusOK, Response{OK: true, Data: payload})
}

// EnqueueAutomation manually enqueues a background automation job for a customer.
func (h *Handlers) EnqueueAutomation(w http.ResponseWriter, r *http.Request) {
	if h.ServiceBundle == nil || h.ServiceBundle.Automation == nil {
		writeJSON(w, http.StatusServiceUnavailable, Response{OK: false, Error: "自动化服务未启用"})
		return
	}
	customerID, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	if _, err := h.Store.GetCustomer(r.Context(), customerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, Response{OK: false, Error: "客户不存在"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	settings, err := h.Store.GetSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	if !settings.AutomationEnabled {
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: "自动化模式未开启"})
		return
	}
	job, err := h.ServiceBundle.Automation.Enqueue(r.Context(), customerID)
	if err != nil {
		if errors.Is(err, services.ErrAutomationJobExists) {
			writeJSON(w, http.StatusConflict, Response{OK: false, Error: "自动分析任务已在排队或执行中"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, Response{OK: false, Error: err.Error()})
		return
	}
	if job != nil {
		log.Printf("[automation] queued job id=%d customer=%d", job.ID, customerID)
	}
	writeJSON(w, http.StatusOK, Response{OK: true, Data: job})
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
		log.Printf("[customers] update failed id=%d name=%s website=%s: %v", customerID, strings.TrimSpace(req.Name), strings.TrimSpace(req.Website), err)
		writeJSON(w, http.StatusBadRequest, Response{OK: false, Error: err.Error()})
		return
	}
	log.Printf("[customers] updated customer id=%d name=%s", customerID, strings.TrimSpace(req.Name))
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

func (h *Handlers) handleLoginPasswordChange(ctx context.Context, incoming *store.Settings) error {
	if h == nil || h.Store == nil || h.Auth == nil || incoming == nil {
		return nil
	}
	plain := strings.TrimSpace(incoming.LoginPassword)
	if plain == "" {
		return nil
	}
	if len([]rune(plain)) < 8 {
		return fmt.Errorf("登录口令长度至少 8 位")
	}
	currentHash, currentVersion, err := h.Store.GetLoginPassword(ctx)
	if err != nil {
		return fmt.Errorf("读取当前登录口令失败: %w", err)
	}
	newHash, err := HashLoginPassword(plain)
	if err != nil {
		return err
	}
	if strings.TrimSpace(currentHash) == newHash {
		incoming.LoginPassword = ""
		return nil
	}
	if currentVersion <= 0 {
		currentVersion = 1
	}
	newVersion := currentVersion + 1
	if err := h.Store.UpdateLoginPassword(ctx, newHash, newVersion); err != nil {
		return err
	}
	if err := h.Auth.UpdatePassword(newHash, newVersion); err != nil {
		return fmt.Errorf("刷新内存登录口令失败: %w", err)
	}
	log.Printf("登录口令已更新，全部已登录会话需要重新认证。")
	incoming.LoginPassword = ""
	return nil
}

func (h *Handlers) decryptSettingsStruct(settings *store.Settings) error {
	if h == nil || h.Auth == nil || settings == nil {
		return nil
	}
	val := reflect.ValueOf(settings).Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if !field.CanSet() || field.Kind() != reflect.String {
			continue
		}
		raw := field.String()
		if strings.TrimSpace(raw) == "" {
			continue
		}
		plain, err := h.Auth.DecryptField(raw)
		if err != nil {
			return fmt.Errorf("敏感字段解密失败: %w", err)
		}
		if plain != raw {
			field.SetString(plain)
		}
	}
	return nil
}

func sanitizeSettings(auth *AuthManager, settings *store.Settings) *store.Settings {
	if settings == nil {
		return nil
	}
	sanitized := *settings
	val := reflect.ValueOf(&sanitized).Elem()
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Kind() != reflect.String {
			continue
		}
		name := strings.ToLower(typ.Field(i).Name)
		if !strings.Contains(name, "password") && !strings.Contains(name, "key") && !strings.Contains(name, "secret") && !strings.Contains(name, "token") {
			continue
		}
		raw := field.String()
		if strings.TrimSpace(raw) == "" || auth == nil {
			field.SetString("")
			continue
		}
		enc, err := auth.EncryptField(raw)
		if err != nil {
			log.Printf("sanitize settings: encrypt %s failed: %v", name, err)
			field.SetString("")
			continue
		}
		field.SetString(enc)
	}
	return &sanitized
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

package domain

import "encoding/json"

// Contact captures a potential customer contact.
type Contact struct {
	Name   string `json:"name"`
	Title  string `json:"title"`
	Email  string `json:"email"`
	Phone  string `json:"phone,omitempty"`
	Source string `json:"source,omitempty"`
	IsKey  bool   `json:"is_key"`
}

// CandidateWebsite represents a possible official website candidate.
type CandidateWebsite struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Rank   int    `json:"rank"`
	Reason string `json:"reason"`
}

// ResolveCompanyRequest contains the user query for Step 1.
type ResolveCompanyRequest struct {
	Query string `json:"query"`
}

// ResolveCompanyResponse holds aggregated company data suggested by AI.
type ResolveCompanyResponse struct {
	CustomerID    int64               `json:"customer_id,omitempty"`
	Name          string              `json:"name,omitempty"`
	Website       string              `json:"website"`
	Country       string              `json:"country"`
	Contacts      []Contact           `json:"contacts"`
	Candidates    []CandidateWebsite  `json:"candidates"`
	Summary       string              `json:"summary"`
	Grade         string              `json:"grade,omitempty"`
	GradeReason   string              `json:"grade_reason,omitempty"`
	LastStep      int                 `json:"last_step,omitempty"`
	Analysis      *AnalysisResponse   `json:"analysis,omitempty"`
	EmailDraft    *EmailDraftResponse `json:"email_draft,omitempty"`
	FollowupID    int64               `json:"followup_id,omitempty"`
	ScheduledTask *ScheduledTask      `json:"scheduled_task,omitempty"`
}

// CreateCompanyRequest persists the curated Step 1 output.
type CreateCompanyRequest struct {
	Name       string          `json:"name"`
	Website    string          `json:"website"`
	Country    string          `json:"country"`
	Summary    string          `json:"summary"`
	Contacts   []Contact       `json:"contacts"`
	SourceJSON json.RawMessage `json:"source_json"`
}

// Customer represents a stored customer record.
type Customer struct {
	ID          int64
	Name        string
	Website     string
	Country     string
	Grade       string
	GradeReason string
	Summary     string
	SourceJSON  json.RawMessage
	CreatedAt   string
	UpdatedAt   string
}

// EmailRecord reflects a stored email row.
type EmailRecord struct {
	ID         int64  `json:"id"`
	CustomerID int64  `json:"customer_id"`
	Type       string `json:"type"`
	Subject    string `json:"subject"`
	Body       string `json:"body"`
	Status     string `json:"status"`
	SentAt     string `json:"sent_at"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// GradeSuggestionResponse holds AI grade recommendation.
type GradeSuggestionResponse struct {
	SuggestedGrade string `json:"suggested_grade"`
	Reason         string `json:"reason"`
}

// GradeConfirmRequest confirms the final grade decided by the user.
type GradeConfirmRequest struct {
	Grade  string `json:"grade"`
	Reason string `json:"reason,omitempty"`
}

// AnalysisResponse stores Step 3 analysis report.
type AnalysisContent struct {
	CoreBusiness  string `json:"core_business"`
	PainPoints    string `json:"pain_points"`
	MyEntryPoints string `json:"my_entry_points"`
	FullReport    string `json:"full_report"`
}

// AnalysisResponse adds database id to the analysis content.
type AnalysisResponse struct {
	AnalysisID int64 `json:"analysis_id"`
	AnalysisContent
}

// EmailDraft represents a generated email.
type EmailDraft struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// EmailDraftResponse represents generated email draft info.
type EmailDraftResponse struct {
	EmailID int64 `json:"email_id"`
	EmailDraft
}

// FirstFollowupRequest links the initial email as the first followup record.
type FirstFollowupRequest struct {
	EmailID int64  `json:"email_id"`
	Notes   string `json:"notes,omitempty"`
}

// ScheduleRequest schedules an automated follow-up.
type ScheduleRequest struct {
	CustomerID     int64 `json:"customer_id"`
	ContextEmailID int64 `json:"context_email_id"`
	DelayDays      int   `json:"delay_days"`
}

// ScheduleResponse provides task id and due time.
type ScheduleResponse struct {
	TaskID int64  `json:"task_id"`
	DueAt  string `json:"due_at"`
}

// ScheduledTask is used when listing scheduled followups.
type ScheduledTask struct {
	ID               int64  `json:"id"`
	CustomerID       int64  `json:"customer_id"`
	DueAt            string `json:"due_at"`
	Status           string `json:"status"`
	ContextEmailID   int64  `json:"context_email_id"`
	GeneratedEmailID int64  `json:"generated_email_id"`
	LastError        string `json:"last_error,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

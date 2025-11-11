package domain

import "encoding/json"

const (
	AutomationStatusQueued    = "queued"
	AutomationStatusRunning   = "running"
	AutomationStatusCompleted = "completed"
	AutomationStatusFailed    = "failed"
)

const (
	AutomationStagePending   = "pending"
	AutomationStageGrading   = "grading"
	AutomationStageAnalysis  = "analysis"
	AutomationStageEmail     = "email"
	AutomationStageFollowup  = "followup"
	AutomationStageCompleted = "completed"
	AutomationStageStopped   = "stopped"
)

// Contact captures a potential customer contact.
type Contact struct {
	Name   string `json:"name"`
	Title  string `json:"title"`
	Email  string `json:"email"`
	Phone  string `json:"phone,omitempty"`
	Source string `json:"source,omitempty"`
	IsKey  bool   `json:"is_key"`
	// IsKeyDecisionMaker mirrors the JSON field returned by the enrichment model.
	IsKeyDecisionMaker bool `json:"is_key_decision_maker,omitempty"`
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
	CustomerID        int64               `json:"customer_id,omitempty"`
	Name              string              `json:"name,omitempty"`
	Website           string              `json:"website"`
	WebsiteConfidence float64             `json:"website_confidence,omitempty"`
	Country           string              `json:"country"`
	Contacts          []Contact           `json:"contacts"`
	Candidates        []CandidateWebsite  `json:"candidates"`
	Summary           string              `json:"summary"`
	Grade             string              `json:"grade,omitempty"`
	GradeReason       string              `json:"grade_reason,omitempty"`
	LastStep          int                 `json:"last_step,omitempty"`
	Analysis          *AnalysisResponse   `json:"analysis,omitempty"`
	EmailDraft        *EmailDraftResponse `json:"email_draft,omitempty"`
	FollowupID        int64               `json:"followup_id,omitempty"`
	ScheduledTask     *ScheduledTask      `json:"scheduled_task,omitempty"`
	AutomationJob     *AutomationJob      `json:"automation_job,omitempty"`
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
	ID           int64
	Name         string
	Website      string
	Country      string
	Grade        string
	GradeReason  string
	Summary      string
	FollowupSent bool
	SourceJSON   json.RawMessage
	CreatedAt    string
	UpdatedAt    string
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
	SuggestedGrade  string   `json:"suggested_grade"`
	Reason          string   `json:"reason"`
	Confidence      float64  `json:"confidence_score,omitempty"`
	PositiveSignals []string `json:"positive_signals,omitempty"`
	NegativeSignals []string `json:"negative_signals,omitempty"`
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
	CustomerID     int64  `json:"customer_id"`
	ContextEmailID int64  `json:"context_email_id"`
	Mode           string `json:"mode,omitempty"`
	DelayValue     int    `json:"delay_value,omitempty"`
	DelayUnit      string `json:"delay_unit,omitempty"`
	CronExpression string `json:"cron_expression,omitempty"`
}

// ScheduleResponse provides task id and due time.
type ScheduleResponse struct {
	TaskID         int64  `json:"task_id"`
	DueAt          string `json:"due_at"`
	Mode           string `json:"mode"`
	DelayValue     int    `json:"delay_value,omitempty"`
	DelayUnit      string `json:"delay_unit,omitempty"`
	CronExpression string `json:"cron_expression,omitempty"`
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
	Mode             string `json:"mode,omitempty"`
	DelayValue       int    `json:"delay_value,omitempty"`
	DelayUnit        string `json:"delay_unit,omitempty"`
	CronExpression   string `json:"cron_expression,omitempty"`
	Attempts         int    `json:"attempts,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// AutomationJob describes a background automation workflow execution.
type AutomationJob struct {
	ID         int64  `json:"id"`
	CustomerID int64  `json:"customer_id"`
	Status     string `json:"status"`
	Stage      string `json:"stage"`
	LastError  string `json:"last_error,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// TodoTask persists raw user queries for background processing.
type TodoTask struct {
	ID         int64  `json:"id"`
	Query      string `json:"query"`
	Status     string `json:"status"`
	LastError  string `json:"last_error,omitempty"`
	CustomerID int64  `json:"customer_id,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	FinishedAt string `json:"finished_at,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// CustomerSummary represents the lightweight information shown in the customer list.
type CustomerSummary struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Website        string `json:"website,omitempty"`
	Country        string `json:"country"`
	Grade          string `json:"grade"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	LastFollowupAt string `json:"last_followup_at,omitempty"`
	NextFollowupAt string `json:"next_followup_at,omitempty"`
	Status         string `json:"status"`
	FollowupSent   bool   `json:"followup_sent"`
}

// CustomerDetail aggregates all five workflow steps for editing.
type CustomerDetail struct {
	ID            int64               `json:"id"`
	Name          string              `json:"name"`
	Website       string              `json:"website"`
	Country       string              `json:"country"`
	Summary       string              `json:"summary"`
	Grade         string              `json:"grade"`
	GradeReason   string              `json:"grade_reason"`
	FollowupSent  bool                `json:"followup_sent"`
	Contacts      []Contact           `json:"contacts"`
	Analysis      *AnalysisResponse   `json:"analysis,omitempty"`
	EmailDraft    *EmailDraftResponse `json:"email_draft,omitempty"`
	FollowupID    int64               `json:"followup_id,omitempty"`
	ScheduledTask *ScheduledTask      `json:"scheduled_task,omitempty"`
	AutomationJob *AutomationJob      `json:"automation_job,omitempty"`
	SourceJSON    json.RawMessage     `json:"source_json,omitempty"`
	CreatedAt     string              `json:"created_at"`
	UpdatedAt     string              `json:"updated_at"`
}

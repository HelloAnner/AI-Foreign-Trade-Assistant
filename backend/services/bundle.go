package services

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

// ErrNotImplemented indicates the service is not ready yet.
var ErrNotImplemented = errors.New("service not implemented")

// Bundle aggregates all runtime services the handlers depend on.
type Bundle struct {
	LLM           LLMService
	Mailer        MailService
	Search        SearchService
	Enricher      EnrichmentService
	Grader        GradingService
	Analyst       AnalysisService
	EmailComposer EmailComposerService
	Scheduler     SchedulerService
	Automation    AutomationService
}

// Options describes dependencies shared across services.
type Options struct {
	Store      *store.Store
	HTTPClient *http.Client
}

// LLMService validates credentials and proxies prompt calls.
type LLMService interface {
	TestConnection(ctx context.Context) (map[string]string, error)
}

// MailService sends transactional emails.
type MailService interface {
	SendTest(ctx context.Context) error
	Send(ctx context.Context, to []string, subject, body string) (string, error)
}

// SearchService performs external queries to discover company info.
type SearchService interface {
	Search(ctx context.Context, query string, limit int) ([]SearchItem, error)
	TestSearch(ctx context.Context) error
}

// EnrichmentService resolves company data.
type EnrichmentService interface {
	ResolveCompany(ctx context.Context, req *domain.ResolveCompanyRequest) (*domain.ResolveCompanyResponse, error)
}

// GradingService evaluates customers.
type GradingService interface {
	Suggest(ctx context.Context, customerID int64) (*domain.GradeSuggestionResponse, error)
	Confirm(ctx context.Context, customerID int64, grade, reason string) error
}

// AnalysisService builds entry point reports.
type AnalysisService interface {
	Generate(ctx context.Context, customerID int64) (*domain.AnalysisResponse, error)
}

// EmailComposerService drafts outbound emails.
type EmailComposerService interface {
	DraftInitial(ctx context.Context, customerID int64) (*domain.EmailDraftResponse, error)
	DraftFollowup(ctx context.Context, customerID int64, contextEmailID int64) (*domain.EmailDraft, error)
}

// SchedulerService schedules follow-up jobs.
type SchedulerService interface {
	Schedule(ctx context.Context, req *domain.ScheduleRequest) (*domain.ScheduleResponse, error)
	RunNow(ctx context.Context, taskID int64) error
}

// AutomationService coordinates end-to-end background workflows.
type AutomationService interface {
	Enqueue(ctx context.Context, customerID int64) (*domain.AutomationJob, error)
	ProcessNext(ctx context.Context) (bool, error)
}

// NewStubBundle provides placeholder implementations for early scaffolding.
func NewStubBundle() *Bundle {
	return &Bundle{
		LLM:           stubLLM{},
		Mailer:        stubMailer{},
		Search:        stubSearch{},
		Enricher:      stubEnricher{},
		Grader:        stubGrader{},
		Analyst:       stubAnalyst{},
		EmailComposer: stubEmailComposer{},
		Scheduler:     stubScheduler{},
		Automation:    stubAutomation{},
	}
}

type stubLLM struct{}

func (stubLLM) TestConnection(ctx context.Context) (map[string]string, error) {
	return map[string]string{"status": "pending", "message": ErrNotImplemented.Error()}, ErrNotImplemented
}

type stubMailer struct{}

func (stubMailer) SendTest(ctx context.Context) error {
	return ErrNotImplemented
}

func (stubMailer) Send(ctx context.Context, to []string, subject, body string) (string, error) {
	return "", ErrNotImplemented
}

type stubSearch struct{}

func (stubSearch) Search(ctx context.Context, query string, limit int) ([]SearchItem, error) {
	return nil, ErrNotImplemented
}

func (stubSearch) TestSearch(ctx context.Context) error {
	return ErrNotImplemented
}

type stubEnricher struct{}

func (stubEnricher) ResolveCompany(ctx context.Context, req *domain.ResolveCompanyRequest) (*domain.ResolveCompanyResponse, error) {
	return nil, ErrNotImplemented
}

type stubGrader struct{}

func (stubGrader) Suggest(ctx context.Context, customerID int64) (*domain.GradeSuggestionResponse, error) {
	return nil, ErrNotImplemented
}

func (stubGrader) Confirm(ctx context.Context, customerID int64, grade, reason string) error {
	return ErrNotImplemented
}

type stubAnalyst struct{}

func (stubAnalyst) Generate(ctx context.Context, customerID int64) (*domain.AnalysisResponse, error) {
	return nil, ErrNotImplemented
}

type stubEmailComposer struct{}

func (stubEmailComposer) DraftInitial(ctx context.Context, customerID int64) (*domain.EmailDraftResponse, error) {
	return nil, ErrNotImplemented
}

func (stubEmailComposer) DraftFollowup(ctx context.Context, customerID int64, contextEmailID int64) (*domain.EmailDraft, error) {
	return nil, ErrNotImplemented
}

type stubScheduler struct{}

func (stubScheduler) Schedule(ctx context.Context, req *domain.ScheduleRequest) (*domain.ScheduleResponse, error) {
	return nil, ErrNotImplemented
}

func (stubScheduler) RunNow(ctx context.Context, taskID int64) error {
	return ErrNotImplemented
}

type stubAutomation struct{}

func (stubAutomation) Enqueue(ctx context.Context, customerID int64) (*domain.AutomationJob, error) {
	return nil, ErrNotImplemented
}

func (stubAutomation) ProcessNext(ctx context.Context) (bool, error) {
	return false, ErrNotImplemented
}

// NewBundle wires production implementations backed by the provided store and HTTP client.
func NewBundle(opts Options) *Bundle {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	llmClient := NewLLMClient(opts.Store, httpClient)
	mailer := NewSMTPMailer(opts.Store)
	search := NewSearchClient(opts.Store, httpClient)
	fetcher := NewWebFetcher(httpClient)

	enricher := NewEnrichmentService(llmClient, search, fetcher)
	grader := NewGradingService(opts.Store, llmClient)
	analyst := NewAnalysisService(opts.Store, llmClient)
	emailComposer := NewEmailComposerService(opts.Store, llmClient)
	scheduler := NewSchedulerService(opts.Store, emailComposer, mailer)
	automation := NewAutomationService(opts.Store, grader, analyst, emailComposer, scheduler)

	return &Bundle{
		LLM:           llmClient,
		Mailer:        mailer,
		Search:        search,
		Enricher:      enricher,
		Grader:        grader,
		Analyst:       analyst,
		EmailComposer: emailComposer,
		Scheduler:     scheduler,
		Automation:    automation,
	}
}

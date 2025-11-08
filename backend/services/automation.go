package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

// AutomationServiceImpl orchestrates end-to-end customer automation workflows.
type AutomationServiceImpl struct {
	store     *store.Store
	grader    GradingService
	analyst   AnalysisService
	email     EmailComposerService
	scheduler SchedulerService
}

// ErrAutomationJobExists indicates a queued or running automation job already exists.
var ErrAutomationJobExists = errors.New("automation job already in progress")

// NewAutomationService constructs an automation service instance.
func NewAutomationService(st *store.Store, grader GradingService, analyst AnalysisService, email EmailComposerService, scheduler SchedulerService) *AutomationServiceImpl {
	return &AutomationServiceImpl{store: st, grader: grader, analyst: analyst, email: email, scheduler: scheduler}
}

// Enqueue registers a new automation workflow for the given customer.
func (s *AutomationServiceImpl) Enqueue(ctx context.Context, customerID int64) (*domain.AutomationJob, error) {
	existing, err := s.store.GetActiveAutomationJob(ctx, customerID)
	if err != nil {
		return existing, err
	}
	if existing != nil {
		return existing, ErrAutomationJobExists
	}
	job, err := s.store.CreateAutomationJob(ctx, customerID)
	if err == nil && job != nil {
		// 轻量“唤醒”，确保队列不会因为 runner 间隔而显得中断
		go func() { _, _ = s.ProcessNext(context.Background()) }()
	}
	return job, err
}

// ProcessNext claims and executes the next pending automation job.
func (s *AutomationServiceImpl) ProcessNext(ctx context.Context) (bool, error) {
	job, err := s.store.ClaimNextAutomationJob(ctx)
	if err != nil {
		return false, err
	}
	if job == nil {
		return false, nil
	}
	if err := s.runJob(ctx, job); err != nil {
		return true, err
	}
	return true, nil
}

func (s *AutomationServiceImpl) runJob(ctx context.Context, job *domain.AutomationJob) error {
	log.Printf("[automation] job=%d customer=%d started", job.ID, job.CustomerID)
	settings, err := s.store.GetSettings(ctx)
	if err != nil {
		_ = s.store.MarkAutomationJobFailed(ctx, job.ID, domain.AutomationStagePending, err.Error())
		return err
	}

	requiredGrade := strings.ToUpper(strings.TrimSpace(settings.AutomationRequiredGrade))
	if requiredGrade == "" {
		requiredGrade = "A"
	}
	if requiredGrade == "S" {
		requiredGrade = "A"
	}

	// Stage: grading
	if err := s.store.UpdateAutomationJobStage(ctx, job.ID, domain.AutomationStageGrading); err != nil {
		return err
	}
	log.Printf("[automation] job=%d stage=%s", job.ID, domain.AutomationStageGrading)

	suggestion, err := s.grader.Suggest(ctx, job.CustomerID)
	if err != nil {
		_ = s.store.MarkAutomationJobFailed(ctx, job.ID, domain.AutomationStageGrading, err.Error())
		return err
	}

	grade := strings.ToUpper(strings.TrimSpace(suggestion.SuggestedGrade))
	if grade == "" {
		grade = "C"
	}
	reason := strings.TrimSpace(suggestion.Reason)

	if err := s.grader.Confirm(ctx, job.CustomerID, grade, reason); err != nil {
		_ = s.store.MarkAutomationJobFailed(ctx, job.ID, domain.AutomationStageGrading, err.Error())
		return err
	}
	log.Printf("[automation] job=%d grading grade=%s", job.ID, grade)

	if grade != requiredGrade {
		msg := fmt.Sprintf("评级为 %s，未达到自动化阈值 %s，自动化结束", grade, requiredGrade)
		log.Printf("[automation] job=%d stop reason=%s", job.ID, msg)
		if err := s.store.MarkAutomationJobStopped(ctx, job.ID, msg); err != nil {
			return err
		}
		return s.store.DeleteAutomationJob(ctx, job.ID)
	}

	// Stage: analysis
	if err := s.store.UpdateAutomationJobStage(ctx, job.ID, domain.AutomationStageAnalysis); err != nil {
		return err
	}
	log.Printf("[automation] job=%d stage=%s", job.ID, domain.AutomationStageAnalysis)
	if _, err := s.analyst.Generate(ctx, job.CustomerID); err != nil {
		_ = s.store.MarkAutomationJobFailed(ctx, job.ID, domain.AutomationStageAnalysis, err.Error())
		return err
	}
	log.Printf("[automation] job=%d analysis generated", job.ID)

	// Stage: email drafting
	if err := s.store.UpdateAutomationJobStage(ctx, job.ID, domain.AutomationStageEmail); err != nil {
		return err
	}
	log.Printf("[automation] job=%d stage=%s", job.ID, domain.AutomationStageEmail)
	emailDraft, err := s.email.DraftInitial(ctx, job.CustomerID)
	if err != nil {
		_ = s.store.MarkAutomationJobFailed(ctx, job.ID, domain.AutomationStageEmail, err.Error())
		return err
	}
	emailID := emailDraft.EmailID
	if emailID == 0 {
		err := fmt.Errorf("自动化生成的邮件缺少有效 ID")
		_ = s.store.MarkAutomationJobFailed(ctx, job.ID, domain.AutomationStageEmail, err.Error())
		return err
	}
	log.Printf("[automation] job=%d email_draft id=%d", job.ID, emailID)

	// Stage: follow-up bookkeeping & scheduling
	if err := s.store.UpdateAutomationJobStage(ctx, job.ID, domain.AutomationStageFollowup); err != nil {
		return err
	}
	log.Printf("[automation] job=%d stage=%s", job.ID, domain.AutomationStageFollowup)

	followupID, ferr := s.store.GetLatestFollowupID(ctx, job.CustomerID)
	if ferr != nil {
		log.Printf("automation: 获取跟进记录失败 customer=%d: %v", job.CustomerID, ferr)
	}
	if followupID == 0 {
		if _, err := s.store.SaveInitialFollowup(ctx, job.CustomerID, emailID, "自动化流程创建"); err != nil {
			_ = s.store.MarkAutomationJobFailed(ctx, job.ID, domain.AutomationStageFollowup, err.Error())
			return err
		}
		log.Printf("[automation] job=%d followup created", job.ID)
	}

	scheduled, serr := s.store.GetLatestScheduledTask(ctx, job.CustomerID)
	if serr != nil {
		log.Printf("automation: 获取自动跟进任务失败 customer=%d: %v", job.CustomerID, serr)
	}
	if scheduled == nil {
		delay := settings.AutomationFollowupDays
		if delay <= 0 {
			delay = 3
		}
		_, err := s.scheduler.Schedule(ctx, &domain.ScheduleRequest{
			CustomerID:     job.CustomerID,
			ContextEmailID: emailID,
			Mode:           "simple",
			DelayValue:     delay,
			DelayUnit:      "days",
		})
		if err != nil {
			// If there is no admin email configured, gracefully stop automation.
			if errors.Is(err, ErrNoAdminEmail) {
				msg := "未配置管理员邮箱，已跳过自动跟进；请在设置中填写管理员邮箱后再试"
				if e := s.store.MarkAutomationJobStopped(ctx, job.ID, msg); e != nil {
					return e
				}
				log.Printf("[automation] job=%d stopped: %s", job.ID, msg)
				// Clean up the job record as in other stop/completed paths.
				return s.store.DeleteAutomationJob(ctx, job.ID)
			}
			_ = s.store.MarkAutomationJobFailed(ctx, job.ID, domain.AutomationStageFollowup, err.Error())
			return err
		}
		log.Printf("[automation] job=%d followup scheduled delay_days=%d", job.ID, delay)
	}

	if err := s.store.MarkAutomationJobCompleted(ctx, job.ID, domain.AutomationStageCompleted); err != nil {
		return err
	}
	log.Printf("[automation] job=%d completed", job.ID)
	return s.store.DeleteAutomationJob(ctx, job.ID)
}

var _ AutomationService = (*AutomationServiceImpl)(nil)

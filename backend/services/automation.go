package services

import (
	"context"
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

// NewAutomationService constructs an automation service instance.
func NewAutomationService(st *store.Store, grader GradingService, analyst AnalysisService, email EmailComposerService, scheduler SchedulerService) *AutomationServiceImpl {
	return &AutomationServiceImpl{store: st, grader: grader, analyst: analyst, email: email, scheduler: scheduler}
}

// Enqueue registers a new automation workflow for the given customer.
func (s *AutomationServiceImpl) Enqueue(ctx context.Context, customerID int64) (*domain.AutomationJob, error) {
	return s.store.CreateAutomationJob(ctx, customerID)
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
	settings, err := s.store.GetSettings(ctx)
	if err != nil {
		_ = s.store.MarkAutomationJobFailed(ctx, job.ID, domain.AutomationStagePending, err.Error())
		return err
	}

	requiredGrade := strings.ToUpper(strings.TrimSpace(settings.AutomationRequiredGrade))
	if requiredGrade == "" {
		requiredGrade = "A"
	}

	// Stage: grading
	if err := s.store.UpdateAutomationJobStage(ctx, job.ID, domain.AutomationStageGrading); err != nil {
		return err
	}

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

	if grade != requiredGrade {
		msg := fmt.Sprintf("评级为 %s，未达到自动化阈值 %s，自动化结束", grade, requiredGrade)
		if err := s.store.MarkAutomationJobStopped(ctx, job.ID, msg); err != nil {
			return err
		}
		return s.store.DeleteAutomationJob(ctx, job.ID)
	}

	// Stage: analysis
	if err := s.store.UpdateAutomationJobStage(ctx, job.ID, domain.AutomationStageAnalysis); err != nil {
		return err
	}
	if _, err := s.analyst.Generate(ctx, job.CustomerID); err != nil {
		_ = s.store.MarkAutomationJobFailed(ctx, job.ID, domain.AutomationStageAnalysis, err.Error())
		return err
	}

	// Stage: email drafting
	if err := s.store.UpdateAutomationJobStage(ctx, job.ID, domain.AutomationStageEmail); err != nil {
		return err
	}
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

	// Stage: follow-up bookkeeping & scheduling
	if err := s.store.UpdateAutomationJobStage(ctx, job.ID, domain.AutomationStageFollowup); err != nil {
		return err
	}

	followupID, ferr := s.store.GetLatestFollowupID(ctx, job.CustomerID)
	if ferr != nil {
		log.Printf("automation: 获取跟进记录失败 customer=%d: %v", job.CustomerID, ferr)
	}
	if followupID == 0 {
		if _, err := s.store.SaveInitialFollowup(ctx, job.CustomerID, emailID, "自动化流程创建"); err != nil {
			_ = s.store.MarkAutomationJobFailed(ctx, job.ID, domain.AutomationStageFollowup, err.Error())
			return err
		}
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
			_ = s.store.MarkAutomationJobFailed(ctx, job.ID, domain.AutomationStageFollowup, err.Error())
			return err
		}
	}

	if err := s.store.MarkAutomationJobCompleted(ctx, job.ID, domain.AutomationStageCompleted); err != nil {
		return err
	}
	return s.store.DeleteAutomationJob(ctx, job.ID)
}

var _ AutomationService = (*AutomationServiceImpl)(nil)

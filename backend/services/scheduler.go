package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

// SchedulerServiceImpl manages follow-up scheduling and dispatch.
type SchedulerServiceImpl struct {
	store    *store.Store
	composer EmailComposerService
	mailer   MailService
}

// NewSchedulerService constructs a scheduler instance.
func NewSchedulerService(st *store.Store, composer EmailComposerService, mailer MailService) *SchedulerServiceImpl {
	return &SchedulerServiceImpl{store: st, composer: composer, mailer: mailer}
}

// Schedule queues a follow-up job after the given delay.
func (s *SchedulerServiceImpl) Schedule(ctx context.Context, req *domain.ScheduleRequest) (*domain.ScheduleResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数为空")
	}
	if req.CustomerID <= 0 || req.ContextEmailID <= 0 {
		return nil, fmt.Errorf("缺少客户或邮件信息")
	}
	if req.DelayDays <= 0 {
		return nil, fmt.Errorf("延迟天数须大于 0")
	}
	dueAt := time.Now().Add(time.Hour * 24 * time.Duration(req.DelayDays))
	taskID, err := s.store.CreateScheduledTask(ctx, req.CustomerID, req.ContextEmailID, dueAt)
	if err != nil {
		return nil, err
	}
	return &domain.ScheduleResponse{TaskID: taskID, DueAt: dueAt.UTC().Format(time.RFC3339)}, nil
}

// RunNow executes a scheduled task immediately.
func (s *SchedulerServiceImpl) RunNow(ctx context.Context, taskID int64) error {
	task, err := s.store.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	if err := s.store.MarkTaskRunning(ctx, taskID); err != nil {
		return err
	}

	finalize := func(status string, emailID sql.NullInt64, errMsg string) error {
		var lastErr sql.NullString
		if errMsg != "" {
			lastErr = sql.NullString{String: errMsg, Valid: true}
		}
		return s.store.UpdateTaskStatus(ctx, taskID, status, emailID, lastErr)
	}

	contacts, err := s.store.ListContacts(ctx, task.CustomerID)
	if err != nil {
		finalize("failed", sql.NullInt64{}, err.Error())
		return err
	}
	recipients := selectRecipientEmails(contacts)
	if len(recipients) == 0 {
		errMsg := "未找到包含邮箱的联系人，请在 Step 1 中补充"
		finalize("failed", sql.NullInt64{}, errMsg)
		return fmt.Errorf(errMsg)
	}

	draft, err := s.composer.DraftFollowup(ctx, task.CustomerID, task.ContextEmailID)
	if err != nil {
		finalize("failed", sql.NullInt64{}, err.Error())
		return err
	}

	emailID, err := s.store.InsertEmailDraft(ctx, task.CustomerID, "followup", *draft, "draft")
	if err != nil {
		finalize("failed", sql.NullInt64{}, err.Error())
		return err
	}

	messageID, err := s.mailer.Send(ctx, recipients, draft.Subject, draft.Body)
	if err != nil {
		finalize("failed", sql.NullInt64{Int64: emailID, Valid: true}, err.Error())
		return err
	}

	now := time.Now()
	if err := s.store.UpdateEmailStatus(ctx, emailID, "sent", &now, messageID); err != nil {
		finalize("failed", sql.NullInt64{Int64: emailID, Valid: true}, err.Error())
		return err
	}

	if err := finalize("sent", sql.NullInt64{Int64: emailID, Valid: true}, ""); err != nil {
		return err
	}
	return nil
}

func selectRecipientEmails(contacts []domain.Contact) []string {
	emails := make([]string, 0)
	for _, c := range contacts {
		if strings.TrimSpace(c.Email) == "" {
			continue
		}
		if c.IsKey {
			emails = append([]string{strings.TrimSpace(c.Email)}, emails...)
		} else {
			emails = append(emails, strings.TrimSpace(c.Email))
		}
	}
	if len(emails) == 0 {
		return emails
	}
	// 去重
	unique := make([]string, 0, len(emails))
	seen := map[string]bool{}
	for _, e := range emails {
		lower := strings.ToLower(e)
		if !seen[lower] {
			seen[lower] = true
			unique = append(unique, e)
		}
	}
	return unique
}

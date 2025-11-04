package services

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "log"
    "strings"
    "time"

	"github.com/robfig/cron/v3"

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

// ErrNoRecipientEmails indicates there is no valid recipient email in contacts.
var ErrNoRecipientEmails = errors.New("no recipient email found")

// Schedule queues a follow-up job after the given delay.
func (s *SchedulerServiceImpl) Schedule(ctx context.Context, req *domain.ScheduleRequest) (*domain.ScheduleResponse, error) {
    if req == nil {
        return nil, fmt.Errorf("请求参数为空")
    }
    if req.CustomerID <= 0 || req.ContextEmailID <= 0 {
        return nil, fmt.Errorf("缺少客户或邮件信息")
    }
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "simple"
	}

	now := time.Now()
	var (
		dueAt      time.Time
		delayValue int
		delayUnit  string
		cronExpr   string
		parsedMode string
	)

	switch mode {
	case "simple":
		unit := normalizeUnit(req.DelayUnit)
		if unit == "" {
			unit = "days"
		}
		value := req.DelayValue
		if value <= 0 {
			value = 3
		}
		duration, canonicalUnit, err := deriveDuration(value, unit)
		if err != nil {
			return nil, err
		}
		delayValue = value
		delayUnit = canonicalUnit
		dueAt = now.Add(duration)
		parsedMode = "simple"
	case "cron":
		cronExpr = strings.TrimSpace(req.CronExpression)
		if cronExpr == "" {
			return nil, fmt.Errorf("cron 表达式不能为空")
		}
		schedule, err := parseCronExpression(cronExpr)
		if err != nil {
			return nil, fmt.Errorf("解析 cron 表达式失败: %w", err)
		}
		next := schedule.Next(now)
		if next.IsZero() || !next.After(now) {
			return nil, fmt.Errorf("无法计算下次执行时间")
		}
		dueAt = next
		parsedMode = "cron"
	default:
		return nil, fmt.Errorf("未知的调度模式: %s", mode)
	}

    // Validate recipient emails before creating schedule to avoid later runtime failures.
    contacts, err := s.store.ListContacts(ctx, req.CustomerID)
    if err != nil {
        return nil, err
    }
    if len(selectRecipientEmails(contacts)) == 0 {
        // allow fallback to admin email if configured
        settings, sErr := s.store.GetSettings(ctx)
        if sErr != nil || strings.TrimSpace(settings.AdminEmail) == "" {
            return nil, fmt.Errorf("%w: 未找到联系人邮箱且未配置管理员邮箱，请在 Step 1 中补充或在设置中填写管理员邮箱", ErrNoRecipientEmails)
        }
    }

    taskID, err := s.store.CreateScheduledTask(ctx, &store.ScheduledTaskInput{
        CustomerID:     req.CustomerID,
        ContextEmailID: req.ContextEmailID,
        DueAt:          dueAt,
        Mode:           parsedMode,
        DelayValue:     delayValue,
		DelayUnit:      delayUnit,
		CronExpression: cronExpr,
	})
	if err != nil {
		return nil, err
	}
	return &domain.ScheduleResponse{
		TaskID:         taskID,
		DueAt:          dueAt.UTC().Format(time.RFC3339),
		Mode:           parsedMode,
		DelayValue:     delayValue,
		DelayUnit:      delayUnit,
		CronExpression: cronExpr,
	}, nil
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

    // retry policy
    const maxAttempts = 3
    backoff := func(nextAttempt int) time.Duration {
        switch nextAttempt {
        case 1:
            return 10 * time.Minute
        case 2:
            return time.Hour
        default:
            return 6 * time.Hour
        }
    }
    reschedule := func(curAttempts int, errMsg string) error {
        nextAttempts := curAttempts + 1
        if nextAttempts > maxAttempts {
            return finalize("failed", sql.NullInt64{}, errMsg)
        }
        nextDue := time.Now().Add(backoff(nextAttempts))
        return s.store.RescheduleTaskAfterFailure(ctx, taskID, nextDue, nextAttempts, errMsg)
    }

    contacts, err := s.store.ListContacts(ctx, task.CustomerID)
    if err != nil {
        _ = reschedule(task.Attempts, err.Error())
        return err
    }
    recipients := selectRecipientEmails(contacts)
    if len(recipients) == 0 {
        // fallback to admin email if configured
        settings, sErr := s.store.GetSettings(ctx)
        if sErr == nil {
            admin := strings.TrimSpace(settings.AdminEmail)
            if admin != "" {
                recipients = []string{admin}
            }
        }
        if len(recipients) == 0 {
            errMsg := "未找到联系人邮箱且未配置管理员邮箱，请在 Step 1 中补充或在设置中填写管理员邮箱"
            _ = reschedule(task.Attempts, errMsg)
            return fmt.Errorf(errMsg)
        }
    }

    draft, err := s.composer.DraftFollowup(ctx, task.CustomerID, task.ContextEmailID)
    if err != nil {
        _ = reschedule(task.Attempts, err.Error())
        return err
    }

    emailID, err := s.store.InsertEmailDraft(ctx, task.CustomerID, "followup", *draft, "draft")
    if err != nil {
        _ = reschedule(task.Attempts, err.Error())
        return err
    }

    messageID, err := s.mailer.Send(ctx, recipients, draft.Subject, draft.Body)
    if err != nil {
        _ = reschedule(task.Attempts, err.Error())
        return err
    }

    now := time.Now()
    if err := s.store.UpdateEmailStatus(ctx, emailID, "sent", &now, messageID); err != nil {
        _ = reschedule(task.Attempts, err.Error())
        return err
    }

    if err := finalize("sent", sql.NullInt64{Int64: emailID, Valid: true}, ""); err != nil {
        return err
    }

	if task.Mode == "cron" && strings.TrimSpace(task.CronExpression) != "" {
		if schedule, err := parseCronExpression(task.CronExpression); err != nil {
			log.Printf("reschedule cron task %d parse error: %v", task.ID, err)
		} else {
			next := schedule.Next(time.Now())
			if next.IsZero() || !next.After(time.Now()) {
				log.Printf("reschedule cron task %d: next occurrence not found", task.ID)
			} else {
				if _, err := s.store.CreateScheduledTask(ctx, &store.ScheduledTaskInput{
					CustomerID:     task.CustomerID,
					ContextEmailID: task.ContextEmailID,
					DueAt:          next,
					Mode:           "cron",
					CronExpression: task.CronExpression,
				}); err != nil {
					log.Printf("reschedule cron task %d insert error: %v", task.ID, err)
				}
			}
		}
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

func normalizeUnit(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "m", "min", "mins", "minute", "minutes", "分钟":
		return "minutes"
	case "h", "hr", "hrs", "hour", "hours", "小时":
		return "hours"
	case "d", "day", "days", "天":
		return "days"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func deriveDuration(value int, unit string) (time.Duration, string, error) {
	if value <= 0 {
		return 0, "", fmt.Errorf("延迟时间必须大于 0")
	}
	switch unit {
	case "minutes":
		return time.Duration(value) * time.Minute, "minutes", nil
	case "hours":
		return time.Duration(value) * time.Hour, "hours", nil
	case "days":
		return time.Duration(value) * 24 * time.Hour, "days", nil
	default:
		return 0, "", fmt.Errorf("不支持的时间单位: %s", unit)
	}
}

var cronParser = cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

func parseCronExpression(expr string) (cron.Schedule, error) {
	return cronParser.Parse(expr)
}

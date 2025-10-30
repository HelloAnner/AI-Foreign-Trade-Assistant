package store

import (
	"context"
	"fmt"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
)

// UpdateEmailStatus updates email status, sent time and message id.
func (s *Store) UpdateEmailStatus(ctx context.Context, emailID int64, status string, sentAt *time.Time, messageID string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	var sentValue interface{}
	if sentAt != nil {
		sentValue = sentAt.UTC().Format(time.RFC3339)
	} else {
		sentValue = nil
	}
	_, err := s.DB.ExecContext(ctx,
		`UPDATE emails SET status = ?, sent_at = ?, smtp_message_id = ?, updated_at = ? WHERE id = ?`,
		status,
		sentValue,
		messageID,
		Now(),
		emailID,
	)
	if err != nil {
		return fmt.Errorf("更新邮件状态失败: %w", err)
	}
	return nil
}

// UpdateEmailDraft overwrites subject and body for a draft email.
func (s *Store) UpdateEmailDraft(ctx context.Context, emailID int64, draft domain.EmailDraft) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("store not initialized")
	}
	_, err := s.DB.ExecContext(ctx,
		`UPDATE emails SET subject = ?, body = ?, updated_at = ? WHERE id = ?`,
		draft.Subject,
		draft.Body,
		Now(),
		emailID,
	)
	if err != nil {
		return fmt.Errorf("更新邮件草稿失败: %w", err)
	}
	return nil
}

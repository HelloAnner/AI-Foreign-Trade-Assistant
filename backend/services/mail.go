package services

import (
	"context"
	"fmt"
	"time"

	gomail "gopkg.in/gomail.v2"

	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

const mailTimeout = 20 * time.Second

// SMTPMailer delivers emails using SMTP settings stored in SQLite.
type SMTPMailer struct {
	store *store.Store
}

// NewSMTPMailer creates a new SMTPMailer.
func NewSMTPMailer(st *store.Store) *SMTPMailer {
	return &SMTPMailer{store: st}
}

// SendTest dispatches a short email to the configured admin inbox.
func (m *SMTPMailer) SendTest(ctx context.Context) error {
	if m == nil || m.store == nil {
		return fmt.Errorf("smtp mailer not initialized")
	}

	ctx, cancel := context.WithTimeout(ctx, mailTimeout)
	defer cancel()

	settings, err := m.store.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	if settings.SMTPHost == "" || settings.SMTPPort == 0 ||
		settings.SMTPUsername == "" || settings.SMTPPassword == "" || settings.AdminEmail == "" {
		return fmt.Errorf("请先在设置中填写完整 SMTP 主机、端口、账号、授权码及管理员邮箱")
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", settings.SMTPUsername)
	msg.SetHeader("To", settings.AdminEmail)
	msg.SetHeader("Subject", "AI 外贸客户开发助手 - SMTP 测试邮件")
	msg.SetBody("text/plain", "这是一封来自 AI 外贸客户开发助手的测试邮件，用于验证 SMTP 配置是否正确。")

	dialer := gomail.NewDialer(settings.SMTPHost, settings.SMTPPort, settings.SMTPUsername, settings.SMTPPassword)

	done := make(chan error, 1)
	go func() {
		done <- dialer.DialAndSend(msg)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("发送测试邮件失败: %w", err)
		}
	}

	return nil
}

// Send delivers an email to the provided recipients and returns a message id.
func (m *SMTPMailer) Send(ctx context.Context, to []string, subject, body string) (string, error) {
	if m == nil || m.store == nil {
		return "", fmt.Errorf("smtp mailer not initialized")
	}
	if len(to) == 0 {
		return "", fmt.Errorf("缺少收件人")
	}

	ctx, cancel := context.WithTimeout(ctx, mailTimeout)
	defer cancel()

	settings, err := m.store.GetSettings(ctx)
	if err != nil {
		return "", fmt.Errorf("读取配置失败: %w", err)
	}

	if settings.SMTPHost == "" || settings.SMTPPort == 0 || settings.SMTPUsername == "" || settings.SMTPPassword == "" {
		return "", fmt.Errorf("请先在设置页填写完整 SMTP 信息")
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", settings.SMTPUsername)
	msg.SetHeader("To", to...)
	if subject == "" {
		subject = "Follow-up"
	}
	msg.SetHeader("Subject", subject)
	messageID := fmt.Sprintf("<%d@%s>", time.Now().UnixNano(), settings.SMTPHost)
	msg.SetHeader("Message-ID", messageID)
	msg.SetBody("text/plain", body)

	dialer := gomail.NewDialer(settings.SMTPHost, settings.SMTPPort, settings.SMTPUsername, settings.SMTPPassword)

	done := make(chan error, 1)
	go func() {
		done <- dialer.DialAndSend(msg)
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("发送邮件失败: %w", err)
		}
	}

	return messageID, nil
}

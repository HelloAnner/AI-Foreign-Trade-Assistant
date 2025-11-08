package services

import (
	"context"
	"fmt"
	"strings"
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
func (m *SMTPMailer) SendTest(ctx context.Context, overrides *store.Settings) error {
	ctx, cancel := context.WithTimeout(ctx, mailTimeout)
	defer cancel()

	settings, err := m.resolveSettings(ctx, overrides)
	if err != nil {
		return err
	}

	inputs, err := buildSMTPInputs(settings, true)
	if err != nil {
		return err
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", inputs.username)
	msg.SetHeader("To", inputs.admin)
	msg.SetHeader("Subject", "AI 外贸客户开发助手 - SMTP 测试邮件")
	msg.SetBody("text/plain", "这是一封来自 AI 外贸客户开发助手的测试邮件，用于验证 SMTP 配置是否正确。")

	dialer := gomail.NewDialer(inputs.host, inputs.port, inputs.username, inputs.password)

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

	settings, err := m.resolveSettings(ctx, nil)
	if err != nil {
		return "", err
	}

	inputs, err := buildSMTPInputs(settings, false)
	if err != nil {
		return "", err
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", inputs.username)
	msg.SetHeader("To", to...)
	if subject == "" {
		subject = "Follow-up"
	}
	msg.SetHeader("Subject", subject)
	messageID := fmt.Sprintf("<%d@%s>", time.Now().UnixNano(), inputs.host)
	msg.SetHeader("Message-ID", messageID)
	msg.SetBody("text/plain", body)

	dialer := gomail.NewDialer(inputs.host, inputs.port, inputs.username, inputs.password)

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

func (m *SMTPMailer) resolveSettings(ctx context.Context, overrides *store.Settings) (*store.Settings, error) {
	if m == nil || m.store == nil {
		if overrides == nil {
			return nil, fmt.Errorf("smtp mailer not initialized")
		}
		return mergeSMTPOverrides(&store.Settings{}, overrides), nil
	}

	settings, err := m.store.GetSettings(ctx)
	if err != nil {
		if overrides == nil {
			return nil, fmt.Errorf("load settings: %w", err)
		}
		settings = &store.Settings{}
	}

	return mergeSMTPOverrides(settings, overrides), nil
}

func mergeSMTPOverrides(base *store.Settings, overrides *store.Settings) *store.Settings {
	if overrides == nil {
		return base
	}
	if base == nil {
		base = &store.Settings{}
	}
	if host := strings.TrimSpace(overrides.SMTPHost); host != "" {
		base.SMTPHost = host
	}
	if overrides.SMTPPort > 0 {
		base.SMTPPort = overrides.SMTPPort
	}
	if username := strings.TrimSpace(overrides.SMTPUsername); username != "" {
		base.SMTPUsername = username
	}
	if password := strings.TrimSpace(overrides.SMTPPassword); password != "" {
		base.SMTPPassword = password
	}
	if admin := strings.TrimSpace(overrides.AdminEmail); admin != "" {
		base.AdminEmail = admin
	}
	return base
}

type smtpInputs struct {
	host     string
	port     int
	username string
	password string
	admin    string
}

func buildSMTPInputs(settings *store.Settings, requireAdmin bool) (*smtpInputs, error) {
	if settings == nil {
		return nil, fmt.Errorf("缺少 SMTP 配置")
	}
	host := strings.TrimSpace(settings.SMTPHost)
	username := strings.TrimSpace(settings.SMTPUsername)
	password := strings.TrimSpace(settings.SMTPPassword)
	admin := strings.TrimSpace(settings.AdminEmail)
	port := settings.SMTPPort

	var missing []string
	if host == "" {
		missing = append(missing, "SMTP 主机")
	}
	if port <= 0 {
		missing = append(missing, "SMTP 端口")
	}
	if username == "" {
		missing = append(missing, "SMTP 账号")
	}
	if password == "" {
		missing = append(missing, "SMTP 授权码")
	}
	if requireAdmin && admin == "" {
		missing = append(missing, "管理员邮箱")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("SMTP 配置缺失：%s", strings.Join(missing, "、"))
	}

	return &smtpInputs{
		host:     host,
		port:     port,
		username: username,
		password: password,
		admin:    admin,
	}, nil
}

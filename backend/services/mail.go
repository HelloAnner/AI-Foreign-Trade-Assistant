package services

import (
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net/url"
	"runtime/debug"
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

	testSubject := "AI 外贸客户开发助手 · SMTP 通道测试"
	testBody := `Dear Administrator,

This is a formal verification email from AI Foreign Trade Assistant. It confirms that the SMTP channel you configured is reachable and ready for production use.

You can now safely trigger automated follow-ups and manual notifications. This message is for diagnostics only—no reply is required.

Thank you for your continued trust and cooperation.`

	plainBody, htmlBody := composeEmailBodies(testSubject, testBody, settings)

	msg := gomail.NewMessage()
	msg.SetHeader("From", inputs.username)
	msg.SetHeader("To", inputs.admin)
	msg.SetHeader("Subject", testSubject)
	if htmlBody != "" {
		msg.SetBody("text/html", htmlBody)
		msg.AddAlternative("text/plain", plainBody)
	} else {
		msg.SetBody("text/plain", plainBody)
	}

	dialer := gomail.NewDialer(inputs.host, inputs.port, inputs.username, inputs.password)
	applyDialerSecurity(dialer, inputs)

	done := make(chan error, 1)
	go func() {
		done <- dialer.DialAndSend(msg)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			logSMTPError("send_test", inputs, err)
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
	plainBody, htmlBody := composeEmailBodies(subject, body, settings)
	if htmlBody != "" {
		msg.SetBody("text/html", htmlBody)
		msg.AddAlternative("text/plain", plainBody)
	} else {
		msg.SetBody("text/plain", plainBody)
	}

	dialer := gomail.NewDialer(inputs.host, inputs.port, inputs.username, inputs.password)
	applyDialerSecurity(dialer, inputs)

	done := make(chan error, 1)
	go func() {
		done <- dialer.DialAndSend(msg)
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case err := <-done:
		if err != nil {
			logSMTPError("send", inputs, err)
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
	if sec := strings.TrimSpace(overrides.SMTPSecurity); sec != "" {
		base.SMTPSecurity = sec
	}
	return base
}

type smtpInputs struct {
	host     string
	port     int
	username string
	password string
	admin    string
	security string
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
	security := strings.ToLower(strings.TrimSpace(settings.SMTPSecurity))

	var missing []string
	if host == "" {
		missing = append(missing, "SMTP 主机")
	}
	if port <= 0 {
		port = defaultSMTPPort(security)
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

	switch security {
	case "ssl", "tls":
	default:
		if port == 465 {
			security = "ssl"
		} else {
			security = "tls"
		}
	}

	return &smtpInputs{
		host:     host,
		port:     port,
		username: username,
		password: password,
		admin:    admin,
		security: security,
	}, nil
}

func applyDialerSecurity(dialer *gomail.Dialer, inputs *smtpInputs) {
	if dialer == nil || inputs == nil {
		return
	}
	switch inputs.security {
	case "ssl":
		dialer.SSL = true
		dialer.TLSConfig = tlsConfigForHost(inputs.host)
	case "tls":
		dialer.SSL = false
		dialer.TLSConfig = tlsConfigForHost(inputs.host)
	default:
		dialer.SSL = inputs.port == 465
		if !dialer.SSL {
			dialer.TLSConfig = tlsConfigForHost(inputs.host)
		}
	}
}

func tlsConfigForHost(host string) *tls.Config {
	if host == "" {
		return &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}
}

func defaultSMTPPort(security string) int {
	switch security {
	case "ssl":
		return 465
	case "tls":
		return 587
	default:
		return 587
	}
}

func logSMTPError(action string, inputs *smtpInputs, err error) {
	if err == nil {
		return
	}
	host := ""
	port := 0
	username := ""
	security := ""
	if inputs != nil {
		host = inputs.host
		port = inputs.port
		username = inputs.username
		security = inputs.security
	}
	stack := debug.Stack()
	log.Printf("[smtp] action=%s host=%s port=%d username=%s security=%s error=%v\n%s", action, host, port, username, security, err, stack)
}

func composeEmailBodies(subject, body string, settings *store.Settings) (string, string) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		trimmed = "Hello,\n\nThis is an automated notification from AI Foreign Trade Assistant."
	}
	signature := buildSignatureLines(settings)
	replyTo := selectReplyTo(settings)

	plainBuilder := strings.Builder{}
	plainBuilder.WriteString(trimmed)
	plainBuilder.WriteString("\n\nPlease feel free to reply directly to this email if you would like to continue the discussion or arrange a call.")
	if len(signature) > 0 {
		plainBuilder.WriteString("\n\nWarm regards,\n")
		plainBuilder.WriteString(strings.Join(signature, "\n"))
	} else {
		plainBuilder.WriteString("\n\nAI Foreign Trade Assistant")
	}
	plain := plainBuilder.String()

	html := renderHTMLBody(subject, trimmed, signature, replyTo)
	return plain, html
}

func buildSignatureLines(settings *store.Settings) []string {
	if settings == nil {
		return nil
	}
	lines := make([]string, 0, 4)
	if name := strings.TrimSpace(settings.MyCompanyName); name != "" {
		lines = append(lines, name)
	}
	if product := strings.TrimSpace(settings.MyProduct); product != "" {
		lines = append(lines, product)
	}
	if sender := strings.TrimSpace(settings.SMTPUsername); sender != "" {
		lines = append(lines, fmt.Sprintf("Email: %s", sender))
	}
	if admin := strings.TrimSpace(settings.AdminEmail); admin != "" && admin != strings.TrimSpace(settings.SMTPUsername) {
		lines = append(lines, fmt.Sprintf("Admin: %s", admin))
	}
	return lines
}

func selectReplyTo(settings *store.Settings) string {
	if settings == nil {
		return ""
	}
	if sender := strings.TrimSpace(settings.SMTPUsername); sender != "" {
		return sender
	}
	return strings.TrimSpace(settings.AdminEmail)
}

func renderHTMLBody(subject, body string, signature []string, replyTo string) string {
	paragraphs := splitParagraphs(body)
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>`)
	if subject != "" {
		sb.WriteString(template.HTMLEscapeString(subject))
	} else {
		sb.WriteString("Message from AI Foreign Trade Assistant")
	}
	sb.WriteString(`</title><style>
*{box-sizing:border-box;}
body{margin:0;font-family:'Segoe UI','Helvetica Neue','PingFang SC',Arial,sans-serif;background:#eef2f9;padding:32px;color:#111827;}
.card{max-width:680px;margin:0 auto;background:#ffffff;border-radius:18px;box-shadow:0 25px 60px rgba(15,23,42,0.08);border:1px solid #dfe3ec;overflow:hidden;}
.card-header{padding:28px 32px;border-bottom:1px solid #edf1f7;background:linear-gradient(135deg,#f9fbff 0%,#edf2ff 60%,#fefefe 100%);}
.card-header p{margin:0;font-size:12px;letter-spacing:0.08em;color:#64748b;text-transform:uppercase;}
.card-header h1{margin:10px 0 0;font-size:22px;color:#0f172a;}
.card-body{padding:28px 32px;}
.card-body p{margin:0 0 18px;line-height:1.75;font-size:15px;color:#374151;}
.cta{display:inline-block;margin:8px 0 24px;padding:12px 26px;border-radius:999px;background:#111f5c;color:#ffffff;text-decoration:none;font-weight:600;letter-spacing:0.02em;}
.signature{padding:22px 32px;background:#f9fafc;border-top:1px solid #e2e8f0;font-size:13px;color:#475569;}
.signature strong{display:block;margin-bottom:8px;color:#0f172a;font-size:14px;}
@media (max-width:600px){body{padding:16px;} .card-header,.card-body,.signature{padding:20px;}}
</style></head><body><div class="card">`)
	sb.WriteString(`<div class="card-header"><p>AI Foreign Trade Assistant</p><h1>`)
	if subject != "" {
		sb.WriteString(template.HTMLEscapeString(subject))
	} else {
		sb.WriteString("Professional Outreach")
	}
	sb.WriteString(`</h1></div><div class="card-body">`)
	for _, p := range paragraphs {
		if strings.TrimSpace(p) == "" {
			continue
		}
		sb.WriteString(`<p>`)
		sb.WriteString(template.HTMLEscapeString(p))
		sb.WriteString(`</p>`)
	}
	if replyTo != "" {
		mailto := fmt.Sprintf("mailto:%s", template.URLQueryEscaper(replyTo))
		if subject != "" {
			mailto += "?subject=" + url.QueryEscape("Re: "+subject)
		}
		sb.WriteString(`<a class="cta" href="`)
		sb.WriteString(mailto)
		sb.WriteString(`">安排进一步沟通</a>`)
	}
	sb.WriteString(`</div>`)
	if len(signature) > 0 {
		sb.WriteString(`<div class="signature"><strong>Warm regards,</strong>`)
		for _, line := range signature {
			sb.WriteString(template.HTMLEscapeString(line))
			sb.WriteString(`<br/>`)
		}
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</div></body></html>`)
	return sb.String()
}

func splitParagraphs(body string) []string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	lines := strings.Split(body, "\n")
	var paragraphs []string
	var current []string
	flush := func() {
		if len(current) == 0 {
			return
		}
		paragraphs = append(paragraphs, strings.Join(current, " "))
		current = nil
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			flush()
			continue
		}
		current = append(current, line)
	}
	flush()
	if len(paragraphs) == 0 && strings.TrimSpace(body) != "" {
		return []string{strings.TrimSpace(body)}
	}
	if len(paragraphs) == 0 {
		return []string{}
	}
	return paragraphs
}

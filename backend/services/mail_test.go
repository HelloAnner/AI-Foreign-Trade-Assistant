package services

import (
	"strings"
	"testing"

	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

func TestMergeSMTPOverrides(t *testing.T) {
	base := &store.Settings{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     465,
		SMTPUsername: "user@example.com",
		SMTPPassword: "secret",
		AdminEmail:   "admin@example.com",
	}

	override := &store.Settings{
		SMTPHost:     "  smtp.override.com ",
		SMTPPort:     587,
		SMTPUsername: " override@example.com ",
		SMTPPassword: "override-secret",
		AdminEmail:   " admin@override.com ",
	}

	result := mergeSMTPOverrides(base, override)
	if result.SMTPHost != "smtp.override.com" ||
		result.SMTPPort != 587 ||
		result.SMTPUsername != "override@example.com" ||
		result.SMTPPassword != "override-secret" ||
		result.AdminEmail != "admin@override.com" {
		t.Fatalf("unexpected merge result: %#v", result)
	}

	// Blank overrides should keep existing values.
	empty := &store.Settings{
		SMTPHost:     " ",
		SMTPUsername: "",
		SMTPPassword: "",
		AdminEmail:   "",
	}
	result = mergeSMTPOverrides(base, empty)
	if result.SMTPHost != "smtp.override.com" {
		t.Fatalf("blank overrides should not replace host: %#v", result)
	}
}

func TestBuildSMTPInputsValidation(t *testing.T) {
	settings := &store.Settings{
		SMTPHost:     "smtp.example.com",
		SMTPPort:     465,
		SMTPUsername: "user@example.com",
		SMTPPassword: "secret",
		AdminEmail:   "admin@example.com",
	}
	inputs, err := buildSMTPInputs(settings, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inputs.host != settings.SMTPHost || inputs.admin != settings.AdminEmail {
		t.Fatalf("unexpected inputs: %#v", inputs)
	}

	settings.AdminEmail = ""
	_, err = buildSMTPInputs(settings, true)
	if err == nil || !strings.Contains(err.Error(), "管理员邮箱") {
		t.Fatalf("expected admin email error, got %v", err)
	}

	settings.AdminEmail = "admin@example.com"
	settings.SMTPPassword = ""
	_, err = buildSMTPInputs(settings, false)
	if err == nil || !strings.Contains(err.Error(), "SMTP 授权码") {
		t.Fatalf("expected password error, got %v", err)
	}
}

func TestComposeEmailBodies(t *testing.T) {
	settings := &store.Settings{
		MyCompanyName: "Acme Inc.",
		MyProduct:     "High-efficiency widgets",
		SMTPUsername:  "sales@acme.com",
		AdminEmail:    "ops@acme.com",
	}
	subject := "Partnership Opportunity"
	body := "Hello John,\n\nThanks for your time last week.\nWe would love to introduce our new line."
	plain, html := composeEmailBodies(subject, body, settings)
	if !strings.Contains(plain, "Warm regards") || !strings.Contains(plain, "Please feel free to reply directly") {
		t.Fatalf("plain body missing closing: %s", plain)
	}
	if !strings.Contains(html, "<a class=\"cta\"") || !strings.Contains(html, "mailto:sales%40acme.com") {
		t.Fatalf("html body not formatted: %s", html)
	}
}

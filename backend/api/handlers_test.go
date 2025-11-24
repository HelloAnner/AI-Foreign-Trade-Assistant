package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

func TestSanitizeSettingsMasksSensitiveFields(t *testing.T) {
	settings := &store.Settings{
		LLMAPIKey:     "sk-test",
		SMTPPassword:  "secret",
		LoginPassword: "should-not-leak",
	}
	masked := sanitizeSettings(settings)
	if masked == nil {
		t.Fatalf("sanitizeSettings returned nil")
	}
	if masked.LLMAPIKey != maskedSecretPlaceholder {
		t.Fatalf("expected LLMAPIKey to be masked, got %q", masked.LLMAPIKey)
	}
	if masked.SMTPPassword != maskedSecretPlaceholder {
		t.Fatalf("expected SMTPPassword to be masked, got %q", masked.SMTPPassword)
	}
	if masked.LoginPassword != "" {
		t.Fatalf("expected LoginPassword to be empty, got %q", masked.LoginPassword)
	}
}

func TestSaveSettingsPreservesMaskedSecrets(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "app.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	if err := st.InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	initial := store.Settings{
		LLMAPIKey:    "sk-initial",
		SMTPPassword: "hunter2",
		SMTPHost:     "smtp.example.com",
		SMTPUsername: "user@example.com",
		AdminEmail:   "admin@example.com",
	}
	blob, _ := json.Marshal(initial)
	if err := st.SaveSettings(context.Background(), bytes.NewReader(blob)); err != nil {
		t.Fatalf("seed settings: %v", err)
	}
	h := &Handlers{Store: st}
	update := initial
	update.SMTPHost = "smtp.new.com"
	update.SMTPUsername = "user2@example.com"
	update.SMTPPassword = maskedSecretPlaceholder
	update.LLMAPIKey = maskedSecretPlaceholder
	update.AdminEmail = "ops@example.com"
	update.AutomationEnabled = true
	body, _ := json.Marshal(update)
	req := httptest.NewRequest(http.MethodPut, "/settings", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.SaveSettings(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d body=%s", rec.Code, rec.Body.String())
	}
	saved, err := st.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if saved.SMTPPassword != initial.SMTPPassword {
		t.Fatalf("smtp_password overwritten: got %q", saved.SMTPPassword)
	}
	if saved.LLMAPIKey != initial.LLMAPIKey {
		t.Fatalf("llm_api_key overwritten: got %q", saved.LLMAPIKey)
	}
	if saved.SMTPHost != "smtp.new.com" {
		t.Fatalf("expected smtp_host updated, got %q", saved.SMTPHost)
	}
	if saved.SMTPUsername != "user2@example.com" {
		t.Fatalf("expected smtp_username updated, got %q", saved.SMTPUsername)
	}
	if saved.AdminEmail != "ops@example.com" {
		t.Fatalf("expected admin_email updated, got %q", saved.AdminEmail)
	}
}

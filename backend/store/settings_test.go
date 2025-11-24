package store

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsPersistence(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "app.db")

	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if err := st.InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	payload := Settings{
		LLMBaseURL:      "https://example.com/v1",
		LLMAPIKey:       "sk-test",
		LLMModel:        "gpt-4o-mini",
		MyCompanyName:   "Acme Inc",
		MyProduct:       "Widgets",
		SMTPHost:        "smtp.example.com",
		SMTPPort:        465,
		SMTPUsername:    "user@example.com",
		SMTPPassword:    "password",
		AdminEmail:      "admin@example.com",
		RatingGuideline: "A/B/C",
	}
	data, _ := json.Marshal(payload)

	if err := st.SaveSettings(context.Background(), bytes.NewReader(data)); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	st2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer st2.Close()

	settings, err := st2.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}

	if settings.LLMBaseURL != payload.LLMBaseURL ||
		settings.LLMAPIKey != payload.LLMAPIKey ||
		settings.LLMModel != payload.LLMModel ||
		settings.MyCompanyName != payload.MyCompanyName ||
		settings.MyProduct != payload.MyProduct ||
		settings.SMTPHost != payload.SMTPHost ||
		settings.SMTPPort != payload.SMTPPort ||
		settings.SMTPUsername != payload.SMTPUsername ||
		settings.SMTPPassword != payload.SMTPPassword ||
		settings.AdminEmail != payload.AdminEmail ||
		settings.RatingGuideline != payload.RatingGuideline {
		t.Fatalf("settings mismatch after reload: %#v", settings)
	}
}

func TestSettingsSecretsStoredEncrypted(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "app.db")

	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if err := st.InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	payload := Settings{
		LLMAPIKey:    "sk-secret",
		SMTPPassword: "smtp-secret",
	}
	data, _ := json.Marshal(payload)
	if err := st.SaveSettings(context.Background(), bytes.NewReader(data)); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	var rawSMTP string
	if err := st.DB.QueryRowContext(context.Background(), `SELECT smtp_password FROM settings WHERE id = 1`).Scan(&rawSMTP); err != nil {
		t.Fatalf("query raw password: %v", err)
	}
	if !strings.HasPrefix(rawSMTP, dbSecretPrefix) {
		t.Fatalf("expected encrypted smtp password, got %q", rawSMTP)
	}

	settings, err := st.GetSettings(context.Background())
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if settings.SMTPPassword != payload.SMTPPassword {
		t.Fatalf("expected decrypted smtp password, got %q", settings.SMTPPassword)
	}
	if settings.LLMAPIKey != payload.LLMAPIKey {
		t.Fatalf("expected decrypted api key, got %q", settings.LLMAPIKey)
	}
}

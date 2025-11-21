package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Settings represents the persisted global configuration.
type Settings struct {
	LLMBaseURL              string `json:"llm_base_url"`
	LLMAPIKey               string `json:"llm_api_key"`
	LLMModel                string `json:"llm_model"`
	MyCompanyName           string `json:"my_company_name"`
	MyProduct               string `json:"my_product_profile"`
	SMTPHost                string `json:"smtp_host"`
	SMTPPort                int    `json:"smtp_port"`
	SMTPUsername            string `json:"smtp_username"`
	SMTPPassword            string `json:"smtp_password"`
	SMTPSecurity            string `json:"smtp_security"`
	AdminEmail              string `json:"admin_email"`
	RatingGuideline         string `json:"rating_guideline"`
	AutomationEnabled       bool   `json:"automation_enabled"`
	AutomationFollowupDays  int    `json:"automation_followup_days"`
	AutomationRequiredGrade string `json:"automation_required_grade"`
}

// GetSettings fetches the single settings row.
func (s *Store) GetSettings(ctx context.Context) (*Settings, error) {
	row := s.DB.QueryRowContext(ctx, `
	SELECT
	  COALESCE(llm_base_url, ''),
	  COALESCE(llm_api_key, ''),
	  COALESCE(llm_model, ''),
	  COALESCE(my_company_name, ''),
	  COALESCE(my_product_profile, ''),
	  COALESCE(smtp_host, ''),
	  COALESCE(smtp_port, 0),
	  COALESCE(smtp_username, ''),
	  COALESCE(smtp_password, ''),
	  COALESCE(smtp_security, 'auto'),
	  COALESCE(admin_email, ''),
	  COALESCE(rating_guideline, ''),
	  COALESCE(automation_enabled, 0),
	  COALESCE(automation_followup_days, 0),
	  COALESCE(automation_required_grade, '')
	FROM settings WHERE id = 1;
`)
	var settings Settings
	var automationEnabledInt int
	if err := row.Scan(
		&settings.LLMBaseURL,
		&settings.LLMAPIKey,
		&settings.LLMModel,
		&settings.MyCompanyName,
		&settings.MyProduct,
		&settings.SMTPHost,
		&settings.SMTPPort,
		&settings.SMTPUsername,
		&settings.SMTPPassword,
		&settings.SMTPSecurity,
		&settings.AdminEmail,
		&settings.RatingGuideline,
		&automationEnabledInt,
		&settings.AutomationFollowupDays,
		&settings.AutomationRequiredGrade,
	); err != nil {
		return nil, fmt.Errorf("scan settings: %w", err)
	}
	if automationEnabledInt == 1 {
		settings.AutomationEnabled = true
	}
	if settings.AutomationFollowupDays <= 0 {
		settings.AutomationFollowupDays = 3
	}
	if strings.TrimSpace(settings.SMTPSecurity) == "" {
		settings.SMTPSecurity = "auto"
	}
	return &settings, nil
}

// SaveSettings decodes JSON payload and writes to the database.
func (s *Store) SaveSettings(ctx context.Context, body io.Reader) error {
	var payload Settings
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	if payload.AutomationFollowupDays <= 0 {
		payload.AutomationFollowupDays = 3
	}
	payload.SMTPSecurity = strings.TrimSpace(payload.SMTPSecurity)
	if payload.SMTPSecurity == "" {
		payload.SMTPSecurity = "auto"
	}

	_, err := s.DB.ExecContext(ctx, `
		UPDATE settings
		SET llm_base_url = ?, llm_api_key = ?, llm_model = ?,
		    my_company_name = ?, my_product_profile = ?,
		    smtp_host = ?, smtp_port = ?, smtp_username = ?, smtp_password = ?,
		    smtp_security = ?,
		    admin_email = ?, rating_guideline = ?,
		    automation_enabled = ?, automation_followup_days = ?, automation_required_grade = ?,
		    updated_at = datetime('now')
		WHERE id = 1;
	`,
		payload.LLMBaseURL,
		payload.LLMAPIKey,
		payload.LLMModel,
		payload.MyCompanyName,
		payload.MyProduct,
		payload.SMTPHost,
		payload.SMTPPort,
		payload.SMTPUsername,
		payload.SMTPPassword,
		payload.SMTPSecurity,
		payload.AdminEmail,
		payload.RatingGuideline,
		boolToInt(payload.AutomationEnabled),
		payload.AutomationFollowupDays,
		payload.AutomationRequiredGrade,
	)
	if err != nil {
		return fmt.Errorf("update settings: %w", err)
	}
	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

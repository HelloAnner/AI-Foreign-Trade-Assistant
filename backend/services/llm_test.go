package services

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

func TestEnsureConfigured(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/app.db")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	if err := st.InitSchema(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
	payload := store.Settings{
		LLMBaseURL: "https://example.com/v1",
		LLMAPIKey:  "test",
		LLMModel:   "gpt",
	}
	data, _ := json.Marshal(payload)
	if err := st.SaveSettings(context.Background(), bytes.NewReader(data)); err != nil {
		t.Fatalf("save: %v", err)
	}
	client := NewLLMClient(st, nil)
	if err := client.EnsureConfigured(context.Background()); err != nil {
		t.Fatalf("ensure configured unexpectedly failed: %v", err)
	}
}

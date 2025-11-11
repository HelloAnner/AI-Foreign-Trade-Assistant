package services

import (
    "bytes"
    "context"
    "encoding/json"
    "os"
    "path/filepath"
    "testing"

    "github.com/anner/ai-foreign-trade-assistant/backend/store"
)

func setupTestStore(t *testing.T) *store.Store {
    t.Helper()
    dir := t.TempDir()
    dbPath := filepath.Join(dir, "app.db")
    st, err := store.Open(dbPath)
    if err != nil {
        t.Fatalf("open store: %v", err)
    }
    if err := st.InitSchema(context.Background()); err != nil {
        t.Fatalf("init schema: %v", err)
    }
    return st
}

// TestLLMConnection tests LLM connectivity using environment variables
// Required env vars: LLM_API_KEY, LLM_BASE_URL, LLM_MODEL_NAME
func TestLLMConnection(t *testing.T) {
    baseURL := os.Getenv("LLM_BASE_URL")
    apiKey := os.Getenv("LLM_API_KEY")
    model := os.Getenv("LLM_MODEL_NAME")

    if baseURL == "" || apiKey == "" || model == "" {
        t.Skip("LLM 配置不完整（需要 LLM_API_KEY, LLM_BASE_URL, LLM_MODEL_NAME），跳过测试")
    }

    st := setupTestStore(t)
    defer st.Close()

    payload := store.Settings{
        LLMBaseURL: baseURL,
        LLMAPIKey:  apiKey,
        LLMModel:   model,
    }
    data, _ := json.Marshal(payload)
    if err := st.SaveSettings(context.Background(), bytes.NewReader(data)); err != nil {
        t.Fatalf("save settings: %v", err)
    }

    llm := NewLLMClient(st, nil)
    res, err := llm.TestConnection(context.Background())
    if err != nil {
        t.Fatalf("LLM 测试失败: %v", err)
    }
    if res["message"] == "" {
        t.Fatalf("预期返回信息，但得到空响应: %#v", res)
    }
}

// TestSearchClient tests search functionality using Playwright (no API key required)
func TestSearchClient(t *testing.T) {
    st := setupTestStore(t)
    defer st.Close()

    // Playwright doesn't require API configuration
    search := NewSearchClient(st, nil)
    if err := search.TestSearch(context.Background()); err != nil {
        t.Fatalf("Playwright search test failed: %v", err)
    }
}

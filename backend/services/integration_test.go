package services

import (
    "bytes"
    "context"
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/anner/ai-foreign-trade-assistant/backend/store"
)

func loadKeyValueFile(path string) (map[string]string, bool) {
    content, err := os.ReadFile(path)
    if err != nil {
        return nil, false
    }
    result := make(map[string]string)
    lines := strings.Split(string(content), "\n")
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue
        }
        key := strings.TrimSpace(parts[0])
        value := strings.TrimSpace(parts[1])
        result[key] = value
    }
    return result, true
}

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

func TestLLMConnection(t *testing.T) {
    cfg, ok := loadKeyValueFile("config/llm.config")
    if !ok {
        t.Skip("未找到 config/llm.config，跳过 LLM 连通性测试")
    }
    baseURL := cfg["DEEPSEEK_BASE_URL"]
    apiKey := cfg["DEEPSEEK_API_KEY"]
    model := cfg["DEEPSEEK_MODEL_NAME"]
    if baseURL == "" || apiKey == "" || model == "" {
        t.Skip("LLM 配置不完整，跳过测试")
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

func TestSearchClient(t *testing.T) {
    cfg, ok := loadKeyValueFile("config/search.config")
    if !ok {
        cfg = make(map[string]string)
    }
    apiKey := cfg["SERPAPI_API_KEY"]
    if apiKey == "" {
        if env := os.Getenv("SERPAPI_API_KEY"); env != "" {
            apiKey = env
        }
    }
    if apiKey == "" {
        t.Skip("未配置 SERPAPI_API_KEY，跳过搜索 API 测试")
    }

    st := setupTestStore(t)
    defer st.Close()

    payload := store.Settings{
        SearchProvider: "serpapi",
        SearchAPIKey:   apiKey,
    }
    data, _ := json.Marshal(payload)
    if err := st.SaveSettings(context.Background(), bytes.NewReader(data)); err != nil {
        t.Fatalf("save settings: %v", err)
    }

    search := NewSearchClient(st, nil)
    if err := search.TestSearch(context.Background()); err != nil {
        t.Fatalf("搜索 API 测试失败: %v", err)
    }
}

package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

const (
	defaultLLMTimeout = 30 * time.Second
)

// ChatMessage models a single message in a chat completion.
type ChatMessage struct {
	Role    string
	Content string
}

// ChatOptions controls the chat completion parameters.
type ChatOptions struct {
	MaxTokens      int
	Temperature    float32
	ResponseFormat string // e.g. "json_object"
}

// LLMClient calls OpenAI-compatible chat completion endpoints.
type LLMClient struct {
	store      *store.Store
	httpClient *http.Client
}

// NewLLMClient constructs a new LLMClient.
func NewLLMClient(st *store.Store, client *http.Client) *LLMClient {
	if client == nil {
		client = &http.Client{Timeout: defaultLLMTimeout}
	}
	return &LLMClient{
		store:      st,
		httpClient: client,
	}
}

// TestConnection sends a lightweight chat completion request to validate credentials.
func (c *LLMClient) TestConnection(ctx context.Context) (map[string]string, error) {
	if _, err := c.ensureConfigured(ctx); err != nil {
		return nil, err
	}

	content, usage, err := c.Chat(ctx, []ChatMessage{{Role: "user", Content: "你好"}}, ChatOptions{MaxTokens: 32, Temperature: 0.1})
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"message":           "LLM 测试成功",
		"echo":              content,
		"prompt_tokens":     fmt.Sprintf("%d", usage.PromptTokens),
		"completion_tokens": fmt.Sprintf("%d", usage.CompletionTokens),
		"total_tokens":      fmt.Sprintf("%d", usage.TotalTokens),
	}, nil
}

func stringifyError(payload map[string]any) string {
	if payload == nil {
		return "unknown error"
	}
	if errObj, ok := payload["error"].(map[string]any); ok {
		if msg, ok := errObj["message"].(string); ok {
			return msg
		}
	}
	if msg, ok := payload["message"].(string); ok {
		return msg
	}
	return "unknown error"
}

// ensureConfigured returns settings if LLM credentials are present.
func (c *LLMClient) ensureConfigured(ctx context.Context) (*store.Settings, error) {
	if c == nil || c.store == nil {
		return nil, fmt.Errorf("llm client not initialized")
	}
	settings, err := c.store.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	var missing []string
	if strings.TrimSpace(settings.LLMBaseURL) == "" {
		missing = append(missing, "Base URL")
	}
	if strings.TrimSpace(settings.LLMAPIKey) == "" {
		missing = append(missing, "API Key")
	}
	if strings.TrimSpace(settings.LLMModel) == "" {
		missing = append(missing, "模型名称")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("LLM 配置不完整，缺少：%s", strings.Join(missing, " / "))
	}
	return settings, nil
}

// EnsureConfigured validates LLM credentials without issuing a request.
func (c *LLMClient) EnsureConfigured(ctx context.Context) error {
	_, err := c.ensureConfigured(ctx)
	return err
}

// Chat executes a chat completion request and returns the assistant message content.
func (c *LLMClient) Chat(ctx context.Context, messages []ChatMessage, opts ChatOptions) (string, *Usage, error) {
	settings, err := c.ensureConfigured(ctx)
	if err != nil {
		return "", nil, err
	}

	if len(messages) == 0 {
		return "", nil, fmt.Errorf("至少需要一条对话消息")
	}

	reqMessages := make([]map[string]string, 0, len(messages))
	for _, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			role = "user"
		}
		reqMessages = append(reqMessages, map[string]string{
			"role":    role,
			"content": msg.Content,
		})
	}

	maxTokens := opts.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	temperature := opts.Temperature
	if temperature < 0 {
		temperature = 0.6
	}

	payload := map[string]any{
		"model":       settings.LLMModel,
		"messages":    reqMessages,
		"max_tokens":  maxTokens,
		"temperature": temperature,
		"stream":      false,
	}
	if opts.ResponseFormat != "" {
		payload["response_format"] = map[string]string{"type": opts.ResponseFormat}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", nil, fmt.Errorf("编码 LLM 请求失败: %w", err)
	}

	endpoint, err := url.JoinPath(strings.TrimRight(settings.LLMBaseURL, "/"), "chat/completions")
	if err != nil {
		return "", nil, fmt.Errorf("拼接 LLM 地址失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", nil, fmt.Errorf("创建 LLM 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", settings.LLMAPIKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("无法连接到 LLM 服务: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var apiErr map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return "", nil, fmt.Errorf("LLM 接口返回错误(%d): %s", resp.StatusCode, stringifyError(apiErr))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", nil, fmt.Errorf("解析 LLM 响应失败: %w", err)
	}

	if len(parsed.Choices) == 0 {
		return "", &parsed.Usage, fmt.Errorf("LLM 未返回任何内容")
	}

	return strings.TrimSpace(parsed.Choices[0].Message.Content), &parsed.Usage, nil
}

// Usage represents token usage stats from LLM responses.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

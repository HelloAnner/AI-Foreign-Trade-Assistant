package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

// GradingServiceImpl handles Step 2 AI grading.
type GradingServiceImpl struct {
	store *store.Store
	llm   *LLMClient
}

// NewGradingService constructs the grading service.
func NewGradingService(st *store.Store, llm *LLMClient) *GradingServiceImpl {
	return &GradingServiceImpl{store: st, llm: llm}
}

// Suggest provides AI grade recommendation.
func (g *GradingServiceImpl) Suggest(ctx context.Context, customerID int64) (*domain.GradeSuggestionResponse, error) {
	customer, err := g.store.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	settings, err := g.store.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	guideline := strings.TrimSpace(settings.RatingGuideline)
	if guideline == "" {
		guideline = "A级：核心目标客户；B级：潜在合作伙伴；C级：暂不跟进。"
	}

	prompt := buildGradingPrompt(customer, guideline)
	content, _, err := g.llm.Chat(ctx, []ChatMessage{
		{Role: "system", Content: gradingSystemPrompt},
		{Role: "user", Content: prompt},
	}, ChatOptions{MaxTokens: 200, Temperature: 0.2, ResponseFormat: "json_object"})
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Grade  string `json:"grade"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("解析评分结果失败: %w", err)
	}
	parsed.Grade = strings.ToUpper(strings.TrimSpace(parsed.Grade))
	if parsed.Grade != "A" && parsed.Grade != "B" && parsed.Grade != "C" {
		parsed.Grade = "C"
	}
	return &domain.GradeSuggestionResponse{
		SuggestedGrade: parsed.Grade,
		Reason:         strings.TrimSpace(parsed.Reason),
	}, nil
}

// Confirm persists the user's final grade decision.
func (g *GradingServiceImpl) Confirm(ctx context.Context, customerID int64, grade, reason string) error {
	grade = strings.ToUpper(strings.TrimSpace(grade))
	if grade != "A" && grade != "B" && grade != "C" {
		return fmt.Errorf("无效的等级，请选择 A/B/C")
	}
	if err := g.store.UpdateCustomerGrade(ctx, customerID, grade, strings.TrimSpace(reason)); err != nil {
		return err
	}
	return nil
}

func buildGradingPrompt(customer *domain.Customer, guideline string) string {
	var sb strings.Builder
	sb.WriteString("客户名称: ")
	sb.WriteString(customer.Name)
	sb.WriteString("\n官网: ")
	sb.WriteString(customer.Website)
	sb.WriteString("\n国家: ")
	sb.WriteString(customer.Country)
	sb.WriteString("\n已有摘要: ")
	sb.WriteString(customer.Summary)
	sb.WriteString("\n\n评级标准:\n")
	sb.WriteString(guideline)
	sb.WriteString(`
请根据上述标准，输出 JSON：{"grade":"A|B|C","reason":"简要说明"}。
`)
	return sb.String()
}

const gradingSystemPrompt = "你是熟悉外贸客户分级策略的分析师，请严格按照用户提供的标准给出评级建议。"

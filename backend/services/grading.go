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
	}, ChatOptions{MaxTokens: 320, Temperature: 0.2, ResponseFormat: "json_object"})
	if err != nil {
		return nil, err
	}

	var parsed struct {
		SuggestedGrade string  `json:"suggested_grade"`
		Confidence     float64 `json:"confidence_score"`
		Reasoning      struct {
			Positive []string `json:"positive_signals"`
			Negative []string `json:"negative_signals"`
		} `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("解析评分结果失败: %w", err)
	}
	grade := strings.ToUpper(strings.TrimSpace(parsed.SuggestedGrade))
	if grade != "A" && grade != "B" && grade != "C" {
		grade = "C"
	}
	positive := sanitizeSignals(parsed.Reasoning.Positive)
	negative := sanitizeSignals(parsed.Reasoning.Negative)
	reason := buildReasonSummary(positive, negative)
	return &domain.GradeSuggestionResponse{
		SuggestedGrade:  grade,
		Reason:          reason,
		Confidence:      clampConfidence(parsed.Confidence),
		PositiveSignals: positive,
		NegativeSignals: negative,
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

var sb strings.Builder

func buildGradingPrompt(customer *domain.Customer, guideline string) string {
	var sb strings.Builder
	sb.WriteString("### Customer Profile:\n")
	sb.WriteString("- Name: ")
	sb.WriteString(strings.TrimSpace(customer.Name))
	sb.WriteString("\n- Website: ")
	sb.WriteString(strings.TrimSpace(customer.Website))
	sb.WriteString("\n- Country: ")
	sb.WriteString(strings.TrimSpace(customer.Country))
	sb.WriteString("\n- Summary: ")
	sb.WriteString(strings.TrimSpace(customer.Summary))

	sb.WriteString("\n\n### Rating Guideline:\n")
	sb.WriteString(strings.TrimSpace(guideline))

	sb.WriteString(`

### Rating Examples (for calibration):
- **Example of an 'A' Grade Customer**: A large manufacturer in a target industry with over 200 employees and clear demand for our type of product.
- **Example of a 'C' Grade Customer**: A small trading company with a generic website, unclear business focus, and located in a high-risk region.

### Instructions:
Analyze the customer profile against the guideline and examples. Output a JSON object with your suggested grade and a structured reasoning.

### JSON Output Format:
{
  "suggested_grade": "A|B|C",
  "confidence_score": 0.9,
  "reasoning": {
    "positive_signals": [
      "e.g., Company operates in a key target industry.",
      "e.g., Website showcases high-value products relevant to our offerings."
    ],
    "negative_signals": [
      "e.g., Company size appears to be small.",
      "e.g., No key contact information was found."
    ]
  }
}
`)
	return sb.String()
}

const defaultRatingGuideline = `请按照以下标准对潜在客户进行评级：
A级：核心目标客户，具有明确采购需求，规模较大（员工数>200 或营收显著），所在行业与我方产品高度契合，决策链清晰且风险低。
B级：潜在合作伙伴，业务方向相关但短期内需求不明朗，体量或采购能力有限，或信息尚不充分，需要持续跟进验证。
C级：暂不跟进的客户，业务需求与我方产品匹配度低，规模较小或风险较高，无法短期产生合作机会。`

const gradingSystemPrompt = "You are a B2B sales strategist specializing in customer segmentation. Your task is to provide a precise customer rating (A, B, or C) based on the user's guideline. You must justify your rating by listing clear positive and negative signals.\n使用中文输出"

func sanitizeSignals(items []string) []string {
	clean := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	return clean
}

func buildReasonSummary(positive, negative []string) string {
	var parts []string
	if len(positive) > 0 {
		parts = append(parts, "正向信号："+strings.Join(positive, "；"))
	}
	if len(negative) > 0 {
		parts = append(parts, "负向信号："+strings.Join(negative, "；"))
	}
	if len(parts) == 0 {
		return "模型未提供详细理由"
	}
	return strings.Join(parts, "；")
}

func clampConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

// AnalysisServiceImpl implements Step 3 content generation.
type AnalysisServiceImpl struct {
	store *store.Store
	llm   *LLMClient
}

// NewAnalysisService builds a new analysis service.
func NewAnalysisService(st *store.Store, llm *LLMClient) *AnalysisServiceImpl {
	return &AnalysisServiceImpl{store: st, llm: llm}
}

// Generate produces the product entry-point analysis and persists it.
func (a *AnalysisServiceImpl) Generate(ctx context.Context, customerID int64) (*domain.AnalysisResponse, error) {
	customer, err := a.store.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	settings, err := a.store.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	if strings.TrimSpace(settings.MyProduct) == "" {
		return nil, fmt.Errorf("请先在设置页填写“我的产品/服务简介”")
	}

	prompt := buildAnalysisPrompt(customer, settings.MyProduct)
	content, _, err := a.llm.Chat(ctx, []ChatMessage{
		{Role: "system", Content: analysisSystemPrompt},
		{Role: "user", Content: prompt},
	}, ChatOptions{MaxTokens: 600, Temperature: 0.3, ResponseFormat: "json_object"})
	if err != nil {
		return nil, err
	}

	var parsed domain.AnalysisContent
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("解析分析结果失败: %w", err)
	}

	analysisID, err := a.store.SaveAnalysis(ctx, customerID, parsed)
	if err != nil {
		return nil, err
	}

	return &domain.AnalysisResponse{AnalysisID: analysisID, AnalysisContent: parsed}, nil
}

func buildAnalysisPrompt(customer *domain.Customer, productProfile string) string {
	var sb strings.Builder
	sb.WriteString("客户名称: ")
	sb.WriteString(customer.Name)
	sb.WriteString("\n官网: ")
	sb.WriteString(customer.Website)
	sb.WriteString("\n国家: ")
	sb.WriteString(customer.Country)
	sb.WriteString("\n客户摘要: ")
	sb.WriteString(customer.Summary)
	sb.WriteString("\n\n我的产品/服务简介:\n")
	sb.WriteString(productProfile)
	sb.WriteString(`
请输出 JSON：{
  "core_business": "客户主营业务与定位",
  "pain_points": "客户可能的痛点或待解决问题",
  "my_entry_points": "结合我方产品的切入建议",
  "full_report": "对上述内容进行200字左右的综述"
}
使用专业中文表达。
`)
	return sb.String()
}

const analysisSystemPrompt = "你是资深产品策略顾问，请结合客户需求与我方产品优势，输出精准的切入分析。"

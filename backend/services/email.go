package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

// EmailComposerServiceImpl handles email drafting logic.
type EmailComposerServiceImpl struct {
	store *store.Store
	llm   *LLMClient
}

// NewEmailComposerService constructs the email composer.
func NewEmailComposerService(st *store.Store, llm *LLMClient) *EmailComposerServiceImpl {
	return &EmailComposerServiceImpl{store: st, llm: llm}
}

// DraftInitial generates and persists the first outreach email.
func (e *EmailComposerServiceImpl) DraftInitial(ctx context.Context, customerID int64) (*domain.EmailDraftResponse, error) {
	customer, err := e.store.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	analysis, err := e.store.GetLatestAnalysis(ctx, customerID)
	if err != nil {
		return nil, err
	}
	contacts, err := e.store.ListContacts(ctx, customerID)
	if err != nil {
		return nil, err
	}
	settings, err := e.store.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}

	prompt := buildInitialEmailPrompt(customer, analysis, contacts, settings)
	content, _, err := e.llm.Chat(ctx, []ChatMessage{
		{Role: "system", Content: emailSystemPrompt},
		{Role: "user", Content: prompt},
	}, ChatOptions{MaxTokens: 550, Temperature: 0.55, ResponseFormat: "json_object"})
	if err != nil {
		return nil, err
	}

	var parsed domain.EmailDraft
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("解析邮件草稿失败: %w", err)
	}

	subject := strings.TrimSpace(parsed.Subject)
	if subject == "" {
		subject = fmt.Sprintf("合作机会：关于 %s 的解决方案", customer.Name)
	}
	parsed.Subject = subject
	parsed.Body = strings.TrimSpace(parsed.Body)

	emailID, err := e.store.InsertEmailDraft(ctx, customerID, "initial", parsed, "draft")
	if err != nil {
		return nil, err
	}

	return &domain.EmailDraftResponse{EmailID: emailID, EmailDraft: parsed}, nil
}

// DraftFollowup produces a follow-up email body based on previous content.
func (e *EmailComposerServiceImpl) DraftFollowup(ctx context.Context, customerID int64, contextEmailID int64) (*domain.EmailDraft, error) {
	contextEmail, err := e.store.GetEmail(ctx, contextEmailID)
	if err != nil {
		return nil, err
	}
	if contextEmail.CustomerID != customerID {
		return nil, fmt.Errorf("上下文邮件不属于当前客户")
	}
	settings, err := e.store.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}

	prompt := buildFollowupPrompt(contextEmail, settings)
	content, _, err := e.llm.Chat(ctx, []ChatMessage{
		{Role: "system", Content: followupSystemPrompt},
		{Role: "user", Content: prompt},
	}, ChatOptions{MaxTokens: 320, Temperature: 0.6, ResponseFormat: "json_object"})
	if err != nil {
		return nil, err
	}

	var parsed domain.EmailDraft
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("解析跟进邮件失败: %w", err)
	}
	parsed.Subject = strings.TrimSpace(parsed.Subject)
	parsed.Body = strings.TrimSpace(parsed.Body)
	return &parsed, nil
}

func buildInitialEmailPrompt(customer *domain.Customer, analysis *domain.AnalysisResponse, contacts []domain.Contact, settings *store.Settings) string {
	var contactLine string
	if len(contacts) > 0 {
		key := contacts[0]
		if key.Name != "" {
			contactLine = key.Name
		} else if key.Email != "" {
			contactLine = key.Email
		} else {
			contactLine = "there"
		}
	} else {
		contactLine = "there"
	}

	var sb strings.Builder
	sb.WriteString("客户名称: ")
	sb.WriteString(customer.Name)
	sb.WriteString("\n官网: ")
	sb.WriteString(customer.Website)
	sb.WriteString("\n客户摘要: ")
	sb.WriteString(customer.Summary)
	sb.WriteString("\n\n切入点分析:\n")
	sb.WriteString("核心业务: " + analysis.CoreBusiness + "\n")
	sb.WriteString("痛点: " + analysis.PainPoints + "\n")
	sb.WriteString("我方切入点: " + analysis.MyEntryPoints + "\n")
	sb.WriteString("\n我的公司名: " + strings.TrimSpace(settings.MyCompanyName) + "\n")
	sb.WriteString("产品简介: " + strings.TrimSpace(settings.MyProduct) + "\n")
	sb.WriteString("目标联系人: " + contactLine + "\n")
	sb.WriteString("请参考以下三段式英文邮件的结构与语气（仅示例，不可直接复制原文）：\n")
	sb.WriteString("Dear Shirley,\nThank you for your feedback.\nFor your information, we have received quotation for gold plated > 3microns with this target price in average.\nLet me know if your price can be more competitive.\nI would be happy to consider a partnership then.\nThank you very much for your continuous support,\nKind regards,\n")
	sb.WriteString(`
请使用简洁商务英文，输出 JSON：{
  "subject": "邮件标题",
  "body": "150-220词正文，需包含客户痛点与我方解决方案，并以柔性 CTA 收尾"
}
如果缺少联系人姓名，请以 "Hi there" 开头。
`)
	return sb.String()
}

func buildFollowupPrompt(contextEmail *domain.EmailRecord, settings *store.Settings) string {
	var sb strings.Builder
	sb.WriteString("此前已发送的首封邮件正文如下：\n")
	sb.WriteString(contextEmail.Body)
	sb.WriteString(`
请基于上封邮件，撰写一封不超过120词的英文跟进邮件，保持友好语气，并提供一个新的价值点（例如更高性能指标、成功案例或资源下载链接占位）。输出 JSON：{
  "subject": "标题",
  "body": "正文"
}
请继续参考以下三段式示例的语气与节奏（不可原封不动复制内容）：
Dear Shirley,
Thank you for your feedback.
For your information, we have received quotation for gold plated > 3microns with this target price in average.
Let me know if your price can be more competitive.
I would be happy to consider a partnership then.
Thank you very much for your continuous support,
Kind regards,
`)
	if strings.TrimSpace(settings.MyCompanyName) != "" {
		sb.WriteString("发件公司: " + strings.TrimSpace(settings.MyCompanyName) + "\n")
	}
	return sb.String()
}

const emailSystemPrompt = "你是专业的 B2B 外贸业务开发邮件写手，请输出高质量英文邮件。"
const followupSystemPrompt = "你是专业的客户成功经理，请基于已有上下文撰写精炼的英文跟进。"

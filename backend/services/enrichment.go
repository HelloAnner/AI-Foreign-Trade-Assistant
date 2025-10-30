package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
)

// EnrichmentServiceImpl implements Step 1 resolution.
type EnrichmentServiceImpl struct {
	llm     *LLMClient
	search  *SearchClient
	fetcher *WebFetcher
}

// NewEnrichmentService creates a new enrichment service instance.
func NewEnrichmentService(llm *LLMClient, search *SearchClient, fetcher *WebFetcher) *EnrichmentServiceImpl {
	return &EnrichmentServiceImpl{llm: llm, search: search, fetcher: fetcher}
}

// ResolveCompany aggregates search + website info and asks LLM for structured insights.
func (s *EnrichmentServiceImpl) ResolveCompany(ctx context.Context, req *domain.ResolveCompanyRequest) (*domain.ResolveCompanyResponse, error) {
	if req == nil || strings.TrimSpace(req.Query) == "" {
		return nil, fmt.Errorf("请输入客户公司名称或官网地址")
	}

	if err := s.llm.EnsureConfigured(ctx); err != nil {
		return nil, err
	}

	query := strings.TrimSpace(req.Query)
	var (
		searchItems []SearchItem
		err         error
	)

	looksURL := looksLikeURL(query)
	if looksURL {
		searchItems = []SearchItem{{
			Title:   "用户输入",
			URL:     normalizeMaybe(query),
			Snippet: "用户直接提供的候选官网",
		}}
	} else {
		searchItems, err = s.search.Search(ctx, query, 5)
		if err != nil {
			return nil, fmt.Errorf("搜索失败: %w", err)
		}
		if len(searchItems) == 0 {
			return nil, fmt.Errorf("未从搜索中获取有效结果，请尝试手动输入官网")
		}
	}

	candidates := make([]domain.CandidateWebsite, 0, len(searchItems))
	for i, item := range searchItems {
		candidates = append(candidates, domain.CandidateWebsite{
			URL:    item.URL,
			Title:  item.Title,
			Rank:   i + 1,
			Reason: item.Snippet,
		})
	}

	primaryURL := ""
	for _, item := range searchItems {
		if item.URL != "" {
			primaryURL = item.URL
			break
		}
	}

	var pageSummary *WebPageSummary
	if primaryURL != "" {
		ctxFetch, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		pageSummary, err = s.fetcher.Fetch(ctxFetch, primaryURL)
		if err != nil {
			// 忽略抓取错误，继续依赖搜索结果
			pageSummary = &WebPageSummary{URL: normalizeMaybe(primaryURL)}
		}
	} else {
		pageSummary = &WebPageSummary{}
	}

	prompt := buildEnrichmentPrompt(query, searchItems, pageSummary)
	content, _, err := s.llm.Chat(ctx, []ChatMessage{
		{Role: "system", Content: enrichmentSystemPrompt},
		{Role: "user", Content: prompt},
	}, ChatOptions{MaxTokens: 600, Temperature: 0.2, ResponseFormat: "json_object"})
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Website  string           `json:"website"`
		Country  string           `json:"country"`
		Contacts []domain.Contact `json:"contacts"`
		Summary  string           `json:"summary"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("解析 LLM 返回结果失败: %w", err)
	}

	if parsed.Website == "" {
		parsed.Website = primaryURL
	}
	if parsed.Website == "" {
		parsed.Website = query
	}

	contacts := sanitizeContacts(parsed.Contacts)
	if len(contacts) == 0 && pageSummary != nil && len(pageSummary.Emails) > 0 {
		contacts = append(contacts, domain.Contact{
			Name:   "General",
			Title:  "Generic",
			Email:  pageSummary.Emails[0],
			Source: pageSummary.URL,
			IsKey:  true,
		})
	}

	name := ""
	if !looksURL {
		name = query
	}

	return &domain.ResolveCompanyResponse{
		Name:       name,
		Website:    parsed.Website,
		Country:    parsed.Country,
		Contacts:   contacts,
		Candidates: candidates,
		Summary:    parsed.Summary,
	}, nil
}

func buildEnrichmentPrompt(query string, items []SearchItem, page *WebPageSummary) string {
	var b strings.Builder
	b.WriteString("用户输入的公司信息：\n")
	b.WriteString(query)
	b.WriteString("\n\n搜索结果：\n")
	for i, item := range items {
		b.WriteString(fmt.Sprintf("%d. %s\nURL: %s\n摘要: %s\n", i+1, item.Title, item.URL, item.Snippet))
	}
	if page != nil {
		b.WriteString("\n官网页面摘录：\n")
		if page.URL != "" {
			b.WriteString("URL: " + page.URL + "\n")
		}
		if page.Text != "" {
			b.WriteString(page.Text)
		} else {
			b.WriteString("(未获取到正文，仍请根据搜索结果判断)\n")
		}
		if len(page.Emails) > 0 {
			b.WriteString("\n已发现的邮箱：" + strings.Join(page.Emails, ", ") + "\n")
		}
	}
	b.WriteString(`
请基于以上材料，判断最可信的官网、所属国家与关键联系人信息。输出 JSON，字段要求：
{
  "website": string (网址，使用 https 开头),
  "country": string (国家或地区中文或英文写法皆可),
  "contacts": [
    {"name": string, "title": string, "email": string, "is_key": bool, "source": string}
  ],
  "summary": string (80-120 字概览)
}
若没有可靠联系人，可返回空数组，但请尽量基于信息推断。`)
	return b.String()
}

func sanitizeContacts(in []domain.Contact) []domain.Contact {
	out := make([]domain.Contact, 0, len(in))
	seen := map[string]bool{}
	for _, c := range in {
		email := strings.TrimSpace(strings.ToLower(c.Email))
		name := strings.TrimSpace(c.Name)
		title := strings.TrimSpace(c.Title)
		if email == "" && name == "" {
			continue
		}
		key := email
		if key == "" {
			key = name + "|" + title
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		c.Email = email
		c.Name = name
		c.Title = title
		c.Source = strings.TrimSpace(c.Source)
		out = append(out, c)
	}
	return out
}

func looksLikeURL(input string) bool {
	lower := strings.ToLower(strings.TrimSpace(input))
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.Contains(lower, ".")
}

func normalizeMaybe(raw string) string {
	if raw == "" {
		return ""
	}
	normalized, err := normalizeURL(raw)
	if err != nil {
		return raw
	}
	return normalized
}

const enrichmentSystemPrompt = "你是资深的 B2B 市场研究助理，请根据搜索结果与官网文本，输出结构化信息帮助外贸业务员建立客户画像。"

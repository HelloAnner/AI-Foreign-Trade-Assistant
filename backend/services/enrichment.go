package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/mozillazg/go-pinyin"

	"github.com/anner/ai-foreign-trade-assistant/backend/domain"
)

// EnrichmentServiceImpl implements Step 1 resolution.
type EnrichmentServiceImpl struct {
	llm     *LLMClient
	search  *SearchClient
	fetcher *WebFetcher
}

var aggregatorDomainSuffixes = []string{
	"linkedin.com",
	"linkedin.cn",
	"facebook.com",
	"twitter.com",
	"instagram.com",
	"youtube.com",
	"crunchbase.com",
	"bloomberg.com",
	"reuters.com",
	"glassdoor.com",
	"indeed.com",
	"fang.com",
	"58.com",
	"ganji.com",
	"zhihu.com",
	"baidu.com",
	"baike.com",
	"wiki",
	"wikipedia.org",
	"google.com",
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

	query := strings.TrimSpace(req.Query)
	looksURL := looksLikeURL(query)

	llmReady := s.llm != nil
	if llmReady {
		if err := s.llm.EnsureConfigured(ctx); err != nil {
			log.Printf("[enrichment] LLM 未就绪，启用降级模式: %v", err)
			llmReady = false
		}
	}

	var (
		searchItems []SearchItem
		primaryURL  string
		pageSummary *WebPageSummary
		searchErr   error

		knowledge    string
		knowledgeErr error
	)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		searchItems, primaryURL, pageSummary, searchErr = s.collectSearchArtifacts(ctx, query, looksURL)
	}()

	if llmReady {
		wg.Add(1)
		go func() {
			defer wg.Done()
			knowledge, knowledgeErr = s.gatherLLMContext(ctx, query)
		}()
	}

	wg.Wait()

	if searchErr != nil {
		return nil, searchErr
	}
	if len(searchItems) == 0 {
		return nil, fmt.Errorf("未从搜索中获取有效结果，请尝试手动输入官网")
	}
	if knowledgeErr != nil {
		log.Printf("[enrichment] LLM 背景查询失败: %v", knowledgeErr)
	}

	baseWebsite := normalizeMaybe(primaryURL)
	name := ""
	if !looksURL {
		name = query
	}

	summary := ""
	country := ""
	rawContacts := contactsFromPage(pageSummary, baseWebsite)
	llmUsed := false
	parsedConfidence := 0.0
	llmCandidates := make([]struct {
		URL    string `json:"url"`
		Title  string `json:"title"`
		Rank   int    `json:"rank"`
		Reason string `json:"reason"`
	}, 0)
	websiteCandidate := ""

	if llmReady {
		prompt := buildEnrichmentPrompt(query, searchItems, pageSummary, knowledge)
		content, _, err := s.llm.Chat(ctx, []ChatMessage{
			{Role: "system", Content: enrichmentSystemPrompt},
			{Role: "user", Content: prompt},
		}, ChatOptions{MaxTokens: 900, Temperature: 0.2, ResponseFormat: "json_object"})
		if err != nil {
			log.Printf("[enrichment] 解析 LLM 工作流失败，启用降级模式: %v", err)
			llmReady = false
		} else {
			var parsed struct {
				Website           string  `json:"website"`
				WebsiteConfidence float64 `json:"website_confidence"`
				Country           string  `json:"country"`
				Contacts          []struct {
					Name   string `json:"name"`
					Title  string `json:"title"`
					Email  string `json:"email"`
					Phone  string `json:"phone"`
					Source string `json:"source"`
					IsKey  bool   `json:"is_key_decision_maker"`
				} `json:"contacts"`
				Summary    string `json:"summary"`
				Candidates []struct {
					URL    string `json:"url"`
					Title  string `json:"title"`
					Rank   int    `json:"rank"`
					Reason string `json:"reason"`
				} `json:"candidates"`
			}
			if err := json.Unmarshal([]byte(content), &parsed); err != nil {
				log.Printf("[enrichment] 解析结构化结果失败，启用降级模式: %v", err)
				llmReady = false
			} else {
				llmUsed = true
				websiteCandidate = strings.TrimSpace(parsed.Website)
				summary = strings.TrimSpace(parsed.Summary)
				country = strings.TrimSpace(parsed.Country)
				for _, c := range parsed.Contacts {
					rawContacts = append(rawContacts, domain.Contact{
						Name:               strings.TrimSpace(c.Name),
						Title:              strings.TrimSpace(c.Title),
						Email:              strings.TrimSpace(c.Email),
						Phone:              strings.TrimSpace(c.Phone),
						Source:             strings.TrimSpace(c.Source),
						IsKey:              c.IsKey,
						IsKeyDecisionMaker: c.IsKey,
					})
				}
				parsedConfidence = parsed.WebsiteConfidence
				llmCandidates = parsed.Candidates
			}
		}
	}

	website := finalizeWebsite(websiteCandidate, primaryURL, query, looksURL)
	if website == "" {
		website = baseWebsite
	}

	contacts := sanitizeContacts(rawContacts)
	websiteConfidence := computeWebsiteConfidence(website, primaryURL, searchItems, pageSummary)
	if parsedConfidence > 0 {
		websiteConfidence = math.Max(websiteConfidence, parsedConfidence)
	}
	websiteConfidence = math.Min(1, math.Max(0, websiteConfidence))

	candidates := buildCandidateList(searchItems, website)
	if len(llmCandidates) > 0 {
		candidates = mergeCandidateDetails(candidates, llmCandidates)
	}

	if strings.TrimSpace(summary) == "" {
		summary = buildFallbackSummary(pageSummary, knowledge)
	}

	log.Printf("[enrichment] result query=%s name=%s website=%s confidence=%.2f contacts=%d llm_used=%v", query, name, website, websiteConfidence, len(contacts), llmUsed)

	return &domain.ResolveCompanyResponse{
		Name:              name,
		Website:           website,
		WebsiteConfidence: websiteConfidence,
		Country:           country,
		Contacts:          contacts,
		Candidates:        candidates,
		Summary:           summary,
	}, nil
}

func (s *EnrichmentServiceImpl) collectSearchArtifacts(ctx context.Context, query string, looksURL bool) ([]SearchItem, string, *WebPageSummary, error) {
	if looksURL {
		normalized := normalizeMaybe(query)
		items := []SearchItem{{
			Title:   "用户输入",
			URL:     normalized,
			Snippet: "用户直接提供的候选官网",
		}}
		primary := choosePrimaryWebsite(items, query)
		if primary == "" {
			primary = normalized
		}
		summary := s.fetchWebSummary(ctx, primary)
		log.Printf("[enrichment] query=%s results=%d", query, len(items))
		return items, primary, summary, nil
	}

	items, err := s.search.Search(ctx, query, 5)
	if err != nil {
		return nil, "", nil, fmt.Errorf("搜索失败: %w", err)
	}
	if len(items) == 0 {
		return nil, "", nil, fmt.Errorf("未从搜索中获取有效结果，请尝试手动输入官网")
	}
	primary := choosePrimaryWebsite(items, query)
	summary := s.fetchWebSummary(ctx, primary)
	log.Printf("[enrichment] query=%s results=%d", query, len(items))
	return items, primary, summary, nil
}

func (s *EnrichmentServiceImpl) fetchWebSummary(ctx context.Context, url string) *WebPageSummary {
	if strings.TrimSpace(url) == "" {
		return &WebPageSummary{}
	}
	ctxFetch, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	summary, err := s.fetcher.Fetch(ctxFetch, url)
	if err != nil {
		return &WebPageSummary{URL: normalizeMaybe(url)}
	}
	return summary
}

func (s *EnrichmentServiceImpl) gatherLLMContext(ctx context.Context, query string) (string, error) {
	if s.llm == nil {
		return "", fmt.Errorf("llm 未配置")
	}
	messages := []ChatMessage{
		{Role: "system", Content: researchSystemPrompt},
		{Role: "user", Content: fmt.Sprintf("请提供关于公司「%s」的公开信息，包括主要业务、所在国家、官网（如知道）以及其他有参考价值的事实。", strings.TrimSpace(query))},
	}
	content, _, err := s.llm.Chat(ctx, messages, ChatOptions{MaxTokens: 600, Temperature: 0.3})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(content), nil
}

func buildEnrichmentPrompt(query string, items []SearchItem, page *WebPageSummary, knowledge string) string {
	var b strings.Builder
	b.WriteString("My Goal: To build a profile for a potential B2B customer.\n")
	b.WriteString("Initial Query: ")
	b.WriteString(strings.TrimSpace(query))
	b.WriteString("\n\n### Search Engine Results:\n")
	if len(items) == 0 {
		b.WriteString("- 无可用搜索结果\n")
	} else {
		for _, item := range items {
			title := strings.TrimSpace(item.Title)
			if title == "" {
				title = "(无标题)"
			}
			url := strings.TrimSpace(item.URL)
			if url == "" {
				url = "(无 URL)"
			}
			snippet := strings.TrimSpace(stripSearchTag(item.Snippet))
			if snippet == "" {
				snippet = "(无摘要)"
			}
			b.WriteString(fmt.Sprintf("- **%s**\n  URL: %s\n  Snippet: %s\n", title, url, snippet))
		}
	}

	b.WriteString("\n### Official Website Content Analysis:\n")
	if page != nil {
		b.WriteString("Crawled URL: ")
		if strings.TrimSpace(page.URL) != "" {
			b.WriteString(page.URL)
		} else {
			b.WriteString("(抓取失败)")
		}
		b.WriteString("\nKey Text Sections:\n")
		if strings.TrimSpace(page.Text) != "" {
			b.WriteString(page.Text)
		} else {
			b.WriteString("(未抓取到正文内容)\n")
		}
		b.WriteString("\nDiscovered Emails: ")
		b.WriteString(formatEmailsList(page.Emails))
		b.WriteString("\n")
	} else {
		b.WriteString("Crawled URL: (未抓取)\nKey Text Sections:\n(无)\nDiscovered Emails: 无\n")
	}

	if strings.TrimSpace(knowledge) != "" {
		b.WriteString("\n### LLM Background Insights:\n")
		b.WriteString(strings.TrimSpace(knowledge))
		b.WriteString("\n")
	}

	b.WriteString(`
### Instructions:
Based on all the material provided, please perform the following steps:
1.  **Identify the most credible official website.**
2.  **Determine the company's country of operation.**
3.  **Extract key contact persons, especially those in leadership or procurement roles.**
4.  **Write a concise summary focusing on their business model and target market.**
5.  **Output a clean JSON object.**

### JSON Output Format:
{
  "website": "The normalized official URL (https://...)",
  "website_confidence": 0.95,
  "country": "...",
  "contacts": [
    {
      "name": "...",
      "title": "...",
      "email": "...",
      "is_key_decision_maker": true,
      "source": "e.g., Website 'About Us' page, Search Snippet 1"
    }
  ],
  "summary": "A 100-150 word summary in Chinese, focusing on their core business, scale, and primary customer base.",
  "candidates": [
    {
      "url": "...",
      "title": "...",
      "rank": 2,
      "reason": "e.g., Appears to be a regional distributor site, not corporate HQ."
    }
  ]
}
`)

	return b.String()
}

func buildFallbackSummary(page *WebPageSummary, knowledge string) string {
	if strings.TrimSpace(knowledge) != "" {
		return truncateRunes(strings.TrimSpace(knowledge), 500)
	}
	if page != nil && strings.TrimSpace(page.Text) != "" {
		return truncateRunes(page.Text, 400)
	}
	return "暂无概述，可在后续步骤中手动补充。"
}

func finalizeWebsite(candidate, primary, query string, looksURL bool) string {
	if isPlausibleWebsite(candidate) {
		return normalizeMaybe(candidate)
	}
	if isPlausibleWebsite(primary) {
		return normalizeMaybe(primary)
	}
	if looksURL && isPlausibleWebsite(query) {
		return normalizeMaybe(query)
	}
	return ""
}

func isPlausibleWebsite(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	normalized, err := normalizeURL(raw)
	if err != nil {
		return false
	}
	u, err := url.Parse(normalized)
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(u.Host))
	if host == "" || !strings.Contains(host, ".") {
		return false
	}
	if strings.Contains(host, "%") || len(host) < 4 {
		return false
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
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
		isKey := c.IsKey || c.IsKeyDecisionMaker
		c.IsKey = isKey
		c.IsKeyDecisionMaker = isKey
		out = append(out, c)
	}
	return out
}

func computeWebsiteConfidence(website, primaryURL string, items []SearchItem, page *WebPageSummary) float64 {
	websiteDomain := normalizeDomain(website)
	if websiteDomain == "" {
		return 0.2
	}
	score := 0.4
	if primaryURL != "" && normalizeDomain(primaryURL) == websiteDomain {
		score += 0.3
	}
	matches := 0
	for _, item := range items {
		if normalizeDomain(item.URL) == websiteDomain {
			matches++
		}
	}
	if matches > 0 {
		score += math.Min(0.3, float64(matches)*0.1)
	}
	if page != nil {
		if strings.TrimSpace(page.Text) != "" {
			score += 0.05
		}
		if len(page.Emails) > 0 {
			score += 0.05
		}
	}
	if score > 1 {
		score = 1
	}
	if score < 0.1 {
		score = 0.1
	}
	return score
}

func buildCandidateList(items []SearchItem, chosenWebsite string) []domain.CandidateWebsite {
	chosenDomain := normalizeDomain(chosenWebsite)
	type candidate struct {
		domain string
		url    string
		title  string
		reason string
		score  float64
		index  int
	}

	seen := map[string]struct{}{}
	candidates := make([]candidate, 0, len(items))
	for idx, item := range items {
		normURL := strings.TrimSpace(normalizeMaybe(item.URL))
		if normURL == "" {
			continue
		}
		domain := normalizeDomain(normURL)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		label, snippet := splitSnippetLabel(item.Snippet)
		reason := strings.TrimSpace(snippet)
		if reason == "" {
			reason = "来源：搜索结果摘要"
		}
		if label != "" {
			reason = fmt.Sprintf("来源 %s：%s", label, reason)
		}
		score := 0.4
		if chosenDomain != "" && domain == chosenDomain {
			score = 1.0
		} else if chosenDomain != "" && strings.Contains(strings.ToLower(reason), chosenDomain) {
			score += 0.2
		}
		candidates = append(candidates, candidate{
			domain: domain,
			url:    normURL,
			title:  strings.TrimSpace(item.Title),
			reason: reason,
			score:  score,
			index:  idx,
		})
	}

	if chosenDomain != "" {
		found := false
		for _, c := range candidates {
			if c.domain == chosenDomain {
				found = true
				break
			}
		}
		if !found {
			candidates = append(candidates, candidate{
				domain: chosenDomain,
				url:    strings.TrimSpace(normalizeMaybe(chosenWebsite)),
				title:  "",
				reason: "LLM 推荐的官网",
				score:  1.1,
				index:  -1,
			})
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].index < candidates[j].index
		}
		return candidates[i].score > candidates[j].score
	})

	result := make([]domain.CandidateWebsite, 0, len(candidates))
	for _, cand := range candidates {
		if strings.TrimSpace(cand.url) == "" {
			continue
		}
		title := cand.title
		if strings.TrimSpace(title) == "" {
			title = cand.domain
		}
		reason := cand.reason
		if cand.domain != chosenDomain && chosenDomain != "" {
			reason = reasonWithSuffix(reason, "（与推荐官网不同）")
		}
		result = append(result, domain.CandidateWebsite{
			URL:    cand.url,
			Title:  title,
			Reason: reason,
		})
	}
	for i := range result {
		result[i].Rank = i + 1
	}
	return result
}

func mergeCandidateDetails(base []domain.CandidateWebsite, llmCandidates []struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Rank   int    `json:"rank"`
	Reason string `json:"reason"`
}) []domain.CandidateWebsite {
	if len(llmCandidates) == 0 {
		for i := range base {
			base[i].Rank = i + 1
		}
		return base
	}

	indexMap := make(map[string]int)
	for i := range base {
		key := strings.ToLower(normalizeMaybe(base[i].URL))
		if key != "" {
			indexMap[key] = i
		}
	}

	for _, cand := range llmCandidates {
		normURL := strings.TrimSpace(normalizeMaybe(cand.URL))
		if normURL == "" {
			continue
		}
		key := strings.ToLower(normURL)
		if idx, ok := indexMap[key]; ok {
			if strings.TrimSpace(cand.Reason) != "" {
				base[idx].Reason = cand.Reason
			}
			if strings.TrimSpace(cand.Title) != "" {
				base[idx].Title = cand.Title
			}
		} else {
			base = append(base, domain.CandidateWebsite{
				URL:    normURL,
				Title:  strings.TrimSpace(cand.Title),
				Reason: strings.TrimSpace(cand.Reason),
			})
			indexMap[key] = len(base) - 1
		}
	}

	for i := range base {
		if strings.TrimSpace(base[i].Title) == "" {
			base[i].Title = base[i].URL
		}
		base[i].Rank = i + 1
	}

	return base
}

func splitSnippetLabel(snippet string) (string, string) {
	trimmed := strings.TrimSpace(snippet)
	if strings.HasPrefix(trimmed, "[") {
		if idx := strings.Index(trimmed, "]"); idx >= 0 && idx < len(trimmed)-1 {
			label := strings.TrimSpace(trimmed[1:idx])
			rest := strings.TrimSpace(trimmed[idx+1:])
			return label, rest
		}
	}
	return "", trimmed
}

func stripSearchTag(snippet string) string {
	_, rest := splitSnippetLabel(snippet)
	return rest
}

func reasonWithSuffix(reason, suffix string) string {
	if strings.Contains(reason, suffix) {
		return reason
	}
	return reason + suffix
}

func normalizeDomain(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	normalized, err := normalizeURL(trimmed)
	if err == nil {
		trimmed = normalized
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(trimmed)), "www.")
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Host))
	host = strings.TrimPrefix(host, "www.")
	return host
}

func formatEmailsList(emails []string) string {
	if len(emails) == 0 {
		return "无"
	}
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(emails))
	for _, email := range emails {
		e := strings.TrimSpace(strings.ToLower(email))
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		unique = append(unique, e)
	}
	if len(unique) == 0 {
		return "无"
	}
	return strings.Join(unique, ", ")
}

func contactsFromPage(page *WebPageSummary, website string) []domain.Contact {
	if page == nil {
		return nil
	}
	source := firstNonEmpty(page.URL, website)
	contacts := make([]domain.Contact, 0, len(page.Emails)+len(page.Phones))
	seenEmail := map[string]struct{}{}
	for idx, email := range page.Emails {
		e := strings.TrimSpace(strings.ToLower(email))
		if e == "" {
			continue
		}
		if _, ok := seenEmail[e]; ok {
			continue
		}
		seenEmail[e] = struct{}{}
		contacts = append(contacts, domain.Contact{
			Email:              e,
			Source:             source,
			IsKey:              idx == 0,
			IsKeyDecisionMaker: idx == 0,
		})
	}
	seenPhone := map[string]struct{}{}
	for _, phone := range page.Phones {
		p := strings.TrimSpace(phone)
		if p == "" {
			continue
		}
		if _, ok := seenPhone[p]; ok {
			continue
		}
		seenPhone[p] = struct{}{}
		contacts = append(contacts, domain.Contact{
			Phone:  p,
			Source: source,
		})
	}
	return contacts
}

func choosePrimaryWebsite(items []SearchItem, query string) string {
	if len(items) == 0 {
		return ""
	}
	tokens := buildQueryTokens(query)
	bestURL := ""
	bestScore := -1.0
	for idx, item := range items {
		url := strings.TrimSpace(normalizeMaybe(item.URL))
		if url == "" {
			continue
		}
		domain := normalizeDomain(url)
		if domain == "" {
			continue
		}
		if isAggregatorDomain(domain) {
			continue
		}
		score := scoreDomainForTokens(domain, tokens, item, idx, query)
		if score > bestScore {
			bestScore = score
			bestURL = url
		}
	}
	if bestURL != "" {
		return bestURL
	}
	for _, item := range items {
		url := strings.TrimSpace(normalizeMaybe(item.URL))
		if url == "" {
			continue
		}
		domain := normalizeDomain(url)
		if domain == "" || isAggregatorDomain(domain) {
			continue
		}
		return url
	}
	return strings.TrimSpace(normalizeMaybe(items[0].URL))
}

func isAggregatorDomain(domain string) bool {
	if domain == "" {
		return false
	}
	lower := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(domain)), "www.")
	for _, suffix := range aggregatorDomainSuffixes {
		s := strings.ToLower(strings.TrimSpace(suffix))
		if s == "" {
			continue
		}
		if strings.HasSuffix(lower, s) || (len(s) > 4 && strings.Contains(lower, s)) {
			return true
		}
	}
	return false
}

func scoreDomainForTokens(domain string, tokens []string, item SearchItem, index int, query string) float64 {
	if domain == "" {
		return -1
	}
	score := 1.0
	if index >= 0 {
		score += 2.0 / float64(index+1)
	}
	lowerDomain := strings.ToLower(domain)
	lowerTitle := strings.ToLower(item.Title)
	lowerSnippet := strings.ToLower(item.Snippet)
	for _, token := range tokens {
		if token == "" {
			continue
		}
		if strings.Contains(lowerDomain, token) {
			score += 3
			continue
		}
		if strings.Contains(lowerTitle, token) {
			score += 1.5
		} else if strings.Contains(lowerSnippet, token) {
			score += 0.8
		}
	}
	parts := strings.Split(lowerDomain, ".")
	if len(parts) > 0 {
		score += math.Max(0.2, 1.6/float64(len(parts)))
	}
	if queryHasChinese(query) && strings.HasSuffix(lowerDomain, ".cn") {
		score += 0.5
	}
	return score
}

func buildQueryTokens(query string) []string {
	lower := strings.ToLower(strings.TrimSpace(query))
	if lower == "" {
		return nil
	}
	seen := map[string]struct{}{}
	var tokens []string
	addToken := func(token string) {
		norm := removeNonAlphaNum(token)
		if len(norm) < 2 {
			return
		}
		if _, ok := seen[norm]; ok {
			return
		}
		seen[norm] = struct{}{}
		tokens = append(tokens, norm)
	}

	split := strings.FieldsFunc(lower, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	for _, part := range split {
		addToken(part)
	}
	addToken(lower)

	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	args.Heteronym = false
	if syllables := pinyin.Pinyin(query, args); len(syllables) > 0 {
		var builder strings.Builder
		for _, sy := range syllables {
			if len(sy) == 0 {
				continue
			}
			norm := removeNonAlphaNum(strings.ToLower(sy[0]))
			addToken(norm)
			builder.WriteString(norm)
		}
		addToken(builder.String())
	}
	return tokens
}

func removeNonAlphaNum(input string) string {
	if input == "" {
		return ""
	}
	var sb strings.Builder
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			sb.WriteRune(unicode.ToLower(r))
		}
	}
	return sb.String()
}

func queryHasChinese(query string) bool {
	for _, r := range query {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
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

const researchSystemPrompt = "You are a seasoned market analyst. Provide factual, concise public information about companies. Prefer verifiable knowledge and mention when uncertain. 使用中文输出"

const enrichmentSystemPrompt = "You are a senior B2B market intelligence analyst. Your mission is to synthesize web search results, prior knowledge, and website content into a structured company profile for a sales team. Prioritize accuracy and identify key decision-makers. 使用中文输出"

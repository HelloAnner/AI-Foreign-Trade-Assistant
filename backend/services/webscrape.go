package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
var phoneRegex = regexp.MustCompile(`(?i)(?:\+?\d[\d\s\-().]{6,}\d)`)

// WebPageSummary captures lightweight website information for LLM prompting.
type WebPageSummary struct {
	URL    string
	Text   string
	Emails []string
	Phones []string
}

// WebFetcher retrieves and summarises webpages.
type WebFetcher struct {
	client *http.Client
}

// NewWebFetcher builds a new WebFetcher.
func NewWebFetcher(client *http.Client) *WebFetcher {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &WebFetcher{client: client}
}

// Fetch retrieves a page and returns condensed text plus discovered emails.
func (w *WebFetcher) Fetch(ctx context.Context, rawURL string) (*WebPageSummary, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("empty url")
	}
	parsed, err := normalizeURL(rawURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed, nil)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AI-Trade-Assistant/1.0)")
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求官网失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("官网返回状态码 %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, 1<<20) // 1MB
	doc, err := goquery.NewDocumentFromReader(limited)
	if err != nil {
		return nil, fmt.Errorf("解析网页失败: %w", err)
	}

	bodyPlain := strings.TrimSpace(compactWhitespace(doc.Find("body").Text()))
	sections := collectKeySections(doc, bodyPlain)
	bodyText := strings.Join(sections, "\n---\n")
	if strings.TrimSpace(bodyText) == "" {
		bodyText = bodyPlain
	}
	bodyText = truncateRunes(bodyText, 4000)

	var emails []string
	var phones []string
	doc.Find("a[href^='mailto:']").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		address := strings.TrimPrefix(href, "mailto:")
		address = strings.TrimSpace(address)
		if address == "" {
			return
		}
		if !containsString(emails, address) {
			emails = append(emails, address)
		}
	})

	// 电话链接
	doc.Find("a[href^='tel:']").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		number := strings.TrimPrefix(href, "tel:")
		number = sanitizePhone(number)
		if number == "" {
			return
		}
		if !containsString(phones, number) {
			phones = append(phones, number)
		}
	})

	matches := emailRegex.FindAllString(bodyPlain, -1)
	for _, m := range matches {
		if !containsString(emails, m) {
			emails = append(emails, m)
		}
	}

	for _, m := range phoneRegex.FindAllString(bodyPlain, -1) {
		number := sanitizePhone(m)
		if number == "" {
			continue
		}
		if !containsString(phones, number) {
			phones = append(phones, number)
		}
	}

	return &WebPageSummary{URL: parsed, Text: bodyText, Emails: emails, Phones: phones}, nil
}

func compactWhitespace(input string) string {
	fields := strings.Fields(input)
	return strings.Join(fields, " ")
}

func containsString(items []string, item string) bool {
	for _, v := range items {
		if strings.EqualFold(v, item) {
			return true
		}
	}
	return false
}

func normalizeURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("empty url")
	}
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		trimmed = "https://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("URL 解析失败: %w", err)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("URL 缺少域名")
	}
	parsed.Fragment = ""
	return parsed.String(), nil
}

func sanitizePhone(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.ReplaceAll(trimmed, "\u00a0", " ")
	trimmed = strings.ReplaceAll(trimmed, "(0)", "")
	allowPlus := false
	if strings.HasPrefix(trimmed, "+") {
		allowPlus = true
	}
	var sb strings.Builder
	for _, r := range trimmed {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
			continue
		}
		if allowPlus && sb.Len() == 0 && r == '+' {
			sb.WriteRune(r)
		}
	}
	result := sb.String()
	if allowPlus && strings.HasPrefix(result, "+") && len(result) < 8 {
		return ""
	}
	if !allowPlus && len(result) < 7 {
		return ""
	}
	return result
}

func collectKeySections(doc *goquery.Document, fallback string) []string {
	type sectionCandidate struct {
		score int
		text  string
	}
	const maxSections = 6
	const maxPerSectionChars = 800

	keywords := []string{"about", "team", "contact", "service", "product", "solution", "company", "history", "联系我们", "关于我们", "产品", "服务", "团队", "简介"}
	keywordWeight := func(text string) int {
		score := 0
		lower := strings.ToLower(text)
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				score += 3
			}
		}
		return score
	}

	candidates := make([]sectionCandidate, 0, 16)
	seen := map[string]struct{}{}

	doc.Find("h1, h2, h3, h4").Each(func(i int, sel *goquery.Selection) {
		heading := strings.TrimSpace(compactWhitespace(sel.Text()))
		if heading == "" {
			return
		}
		var parts []string
		parts = append(parts, heading)
		for _, para := range collectFollowingParagraphs(sel, 3) {
			if para != "" {
				parts = append(parts, para)
			}
		}
		if len(parts) == 0 {
			return
		}
		text := strings.Join(parts, "\n")
		if len(text) < 60 {
			return
		}
		score := 10 - i
		if score < 0 {
			score = 0
		}
		score += keywordWeight(text)
		key := textKey(text)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, sectionCandidate{score: score, text: truncateRunes(text, maxPerSectionChars)})
	})

	doc.Find("section, article").EachWithBreak(func(i int, sel *goquery.Selection) bool {
		if i >= 12 {
			return false
		}
		text := strings.TrimSpace(compactWhitespace(sel.Text()))
		if len(text) < 120 {
			return true
		}
		score := 5 + keywordWeight(text)
		key := textKey(text)
		if _, ok := seen[key]; ok {
			return true
		}
		seen[key] = struct{}{}
		candidates = append(candidates, sectionCandidate{score: score, text: truncateRunes(text, maxPerSectionChars)})
		return true
	})

	if len(candidates) < maxSections {
		doc.Find("p").EachWithBreak(func(i int, sel *goquery.Selection) bool {
			if i >= 40 {
				return false
			}
			text := strings.TrimSpace(compactWhitespace(sel.Text()))
			if len(text) < 80 {
				return true
			}
			key := textKey(text)
			if _, ok := seen[key]; ok {
				return true
			}
			seen[key] = struct{}{}
			score := 1 + keywordWeight(text)
			candidates = append(candidates, sectionCandidate{score: score, text: truncateRunes(text, maxPerSectionChars)})
			return len(candidates) < maxSections*2
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return len(candidates[i].text) > len(candidates[j].text)
		}
		return candidates[i].score > candidates[j].score
	})

	var sections []string
	for _, cand := range candidates {
		if len(sections) >= maxSections {
			break
		}
		if strings.TrimSpace(cand.text) == "" {
			continue
		}
		sections = append(sections, cand.text)
	}

	if len(sections) == 0 && fallback != "" {
		sections = append(sections, truncateRunes(fallback, maxPerSectionChars))
	}

	return sections
}

func collectFollowingParagraphs(sel *goquery.Selection, maxCount int) []string {
	var paragraphs []string
	sel.NextAll().EachWithBreak(func(_ int, sibling *goquery.Selection) bool {
		node := strings.ToLower(goquery.NodeName(sibling))
		if node == "" {
			return true
		}
		if strings.HasPrefix(node, "h1") || strings.HasPrefix(node, "h2") || strings.HasPrefix(node, "h3") || strings.HasPrefix(node, "h4") {
			return false
		}
		if node == "p" || node == "li" {
			text := strings.TrimSpace(compactWhitespace(sibling.Text()))
			if text != "" {
				paragraphs = append(paragraphs, text)
			}
		}
		if len(paragraphs) >= maxCount {
			return false
		}
		return true
	})
	return paragraphs
}

func truncateRunes(input string, limit int) string {
	if limit <= 0 {
		return input
	}
	runes := []rune(input)
	if len(runes) <= limit {
		return input
	}
	return string(runes[:limit])
}

func textKey(text string) string {
	return strings.ToLower(strings.TrimSpace(text))
}

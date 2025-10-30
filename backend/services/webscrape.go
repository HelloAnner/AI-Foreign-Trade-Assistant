package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

// WebPageSummary captures lightweight website information for LLM prompting.
type WebPageSummary struct {
	URL    string
	Text   string
	Emails []string
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

	bodyText := strings.TrimSpace(compactWhitespace(doc.Find("body").Text()))
	if len(bodyText) > 2000 {
		bodyText = bodyText[:2000]
	}

	var emails []string
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

	matches := emailRegex.FindAllString(bodyText, -1)
	for _, m := range matches {
		if !containsString(emails, m) {
			emails = append(emails, m)
		}
	}

	return &WebPageSummary{URL: parsed, Text: bodyText, Emails: emails}, nil
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

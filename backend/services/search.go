package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

const defaultSearchTimeout = 25 * time.Second

// SearchItem represents a single search result snippet.
type SearchItem struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// SearchClient invokes third-party search engines.
type SearchClient struct {
	store      *store.Store
	httpClient *http.Client
	fetcher    *WebFetcher
}

// NewSearchClient constructs a search client.
func NewSearchClient(st *store.Store, client *http.Client) *SearchClient {
	if client == nil {
		client = &http.Client{Timeout: defaultSearchTimeout}
	}
	return &SearchClient{
		store:      st,
		httpClient: client,
		fetcher:    NewWebFetcher(client),
	}
}

// Search executes a query using the configured provider.
func (c *SearchClient) Search(ctx context.Context, query string, limit int) ([]SearchItem, error) {
	if c == nil || c.store == nil {
		return nil, fmt.Errorf("search client not initialized")
	}
	if limit <= 0 {
		limit = 5
	}

	settings, err := c.store.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("load settings: %w", err)
	}

	provider := strings.ToLower(strings.TrimSpace(settings.SearchProvider))
	apiKey := strings.TrimSpace(settings.SearchAPIKey)

	variants := buildSearchVariants(query)
	if len(variants) == 0 {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}

	type result struct {
		idx   int
		items []SearchItem
		err   error
	}

	maxItems := limit * len(variants)
	if maxItems == 0 {
		maxItems = limit
	}

	resultsChan := make(chan result, len(variants))
	var wg sync.WaitGroup
	for idx, variant := range variants {
		wg.Add(1)
		go func(i int, q string) {
			defer wg.Done()
			items, err := c.searchSingle(ctx, provider, apiKey, q, limit)
			resultsChan <- result{idx: i, items: items, err: err}
		}(idx, variant)
	}

	wg.Wait()
	close(resultsChan)

	ordered := make([][]SearchItem, len(variants))
	var firstErr error
	for res := range resultsChan {
		if res.err != nil {
			if firstErr == nil {
				firstErr = res.err
			}
			log.Printf("[search] variant=%d query=\"%s\" error=%v", res.idx, truncateForLog(variants[res.idx], 60), res.err)
			continue
		}
		ordered[res.idx] = res.items
	}

	combined := make([]SearchItem, 0, maxItems)
	seen := map[string]struct{}{}
	for idx, items := range ordered {
		if len(items) == 0 {
			continue
		}
		log.Printf("[search] variant=%d query=\"%s\" results=%d", idx, truncateForLog(variants[idx], 60), len(items))
		for _, item := range items {
			key := searchItemKey(item)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			combined = append(combined, item)
			if maxItems > 0 && len(combined) >= maxItems {
				break
			}
		}
		if maxItems > 0 && len(combined) >= maxItems {
			break
		}
	}

	if len(combined) == 0 && firstErr != nil {
		return nil, firstErr
	}

	return combined, nil
}

func (c *SearchClient) searchSingle(ctx context.Context, provider, apiKey, query string, limit int) ([]SearchItem, error) {
	providerLabel := provider
	if providerLabel == "" {
		providerLabel = "direct"
	}
	log.Printf("[search] provider=%s query=\"%s\" limit=%d status=started", providerLabel, truncateForLog(query, 60), limit)

	switch provider {
	case "bing":
		results, err := c.searchBing(ctx, query, limit, apiKey)
		log.Printf("[search] provider=bing results=%d error=%v", len(results), err)
		return results, err
	case "serpapi":
		results, err := c.searchSerpAPI(ctx, query, limit, apiKey)
		log.Printf("[search] provider=serpapi results=%d error=%v", len(results), err)
		return results, err
	case "ddg", "duckduckgo":
		results, err := c.searchDuckDuckGo(ctx, query, limit)
		log.Printf("[search] provider=duckduckgo results=%d error=%v", len(results), err)
		return results, err
	case "":
		results := directMode(query)
		log.Printf("[search] provider=direct results=%d", len(results))
		return results, nil
	default:
		log.Printf("[search] provider=%s status=unsupported", provider)
		return nil, fmt.Errorf("暂不支持的搜索提供商: %s", provider)
	}
}

func buildSearchVariants(query string) []string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil
	}

	variants := []string{trimmed}
	contactQuery := trimmed
	if !strings.Contains(trimmed, "联系人") {
		contactQuery = strings.TrimSpace(trimmed + " 联系人")
	}
	if !strings.EqualFold(contactQuery, trimmed) {
		variants = append(variants, contactQuery)
	}
	return variants
}

func searchItemKey(item SearchItem) string {
	if normalized := strings.TrimSpace(strings.ToLower(normalizeMaybe(item.URL))); normalized != "" {
		if strings.Contains(item.Snippet, "搜索未配置") {
			snippetKey := strings.TrimSpace(strings.ToLower(item.Snippet))
			if snippetKey != "" {
				return snippetKey
			}
		}
		return normalized
	}

	snippetKey := strings.TrimSpace(strings.ToLower(item.Snippet))
	if snippetKey != "" {
		return snippetKey
	}

	titleKey := strings.TrimSpace(strings.ToLower(item.Title))
	if titleKey != "" {
		return titleKey
	}

	return ""
}

// TestSearch validates that the configured provider can return at least one result.
func (c *SearchClient) TestSearch(ctx context.Context) error {
	if c == nil || c.store == nil {
		return fmt.Errorf("search client not initialized")
	}
	settings, err := c.store.GetSettings(ctx)
	if err != nil {
		return fmt.Errorf("读取配置失败: %w", err)
	}

	provider := strings.ToLower(strings.TrimSpace(settings.SearchProvider))
	if provider == "" {
		return fmt.Errorf("请先在设置中配置搜索提供商")
	}

	results, err := c.Search(ctx, "Apple Inc. 官网", 5)
	if err != nil {
		return err
	}
	if len(results) < 3 {
		return fmt.Errorf("搜索结果不足 3 条，请检查搜索提供商或关键词")
	}

	if c.fetcher == nil {
		c.fetcher = NewWebFetcher(nil)
	}

	for i := 0; i < 3; i++ {
		item := results[i]
		if strings.TrimSpace(item.URL) == "" {
			return fmt.Errorf("第 %d 个搜索结果缺少有效链接", i+1)
		}
		ctxFetch, cancel := context.WithTimeout(ctx, 20*time.Second)
		summary, fetchErr := c.fetcher.Fetch(ctxFetch, item.URL)
		cancel()
		if fetchErr != nil {
			return fmt.Errorf("抓取第 %d 个搜索结果失败: %w", i+1, fetchErr)
		}
		if summary == nil || strings.TrimSpace(summary.Text) == "" {
			return fmt.Errorf("第 %d 个搜索结果未解析到网页正文内容", i+1)
		}
	}

	return nil
}

func truncateForLog(input string, limit int) string {
	trimmed := strings.TrimSpace(input)
	if len([]rune(trimmed)) <= limit {
		return trimmed
	}
	runes := []rune(trimmed)
	return string(runes[:limit]) + "..."
}

func (c *SearchClient) searchBing(ctx context.Context, query string, limit int, apiKey string) ([]SearchItem, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Bing 搜索需要配置 API Key")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.bing.microsoft.com/v7.0/search", nil)
	if err != nil {
		return nil, err
	}
	params := req.URL.Query()
	params.Set("q", query)
	params.Set("count", strconv.Itoa(limit))
	req.URL.RawQuery = params.Encode()
	req.Header.Set("Ocp-Apim-Subscription-Key", apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Bing API 返回状态码 %d", resp.StatusCode)
	}

	var parsed struct {
		WebPages struct {
			Value []struct {
				Name        string `json:"name"`
				URL         string `json:"url"`
				Snippet     string `json:"snippet"`
				Description string `json:"description"`
			} `json:"value"`
		} `json:"webPages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	results := make([]SearchItem, 0, len(parsed.WebPages.Value))
	for _, item := range parsed.WebPages.Value {
		snippet := item.Snippet
		if snippet == "" {
			snippet = item.Description
		}
		results = append(results, SearchItem{
			Title:   item.Name,
			URL:     item.URL,
			Snippet: snippet,
		})
	}
	return results, nil
}

func (c *SearchClient) searchSerpAPI(ctx context.Context, query string, limit int, apiKey string) ([]SearchItem, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("SerpAPI 搜索需要配置 API Key")
	}

	endpoint, _ := url.Parse("https://serpapi.com/search")
	params := endpoint.Query()
	params.Set("engine", "google")
	params.Set("q", query)
	params.Set("num", strconv.Itoa(limit))
	params.Set("api_key", apiKey)
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("SerpAPI 返回状态码 %d", resp.StatusCode)
	}

	var parsed struct {
		OrganicResults []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic_results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	results := make([]SearchItem, 0, len(parsed.OrganicResults))
	for _, item := range parsed.OrganicResults {
		results = append(results, SearchItem{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
		})
	}
	return results, nil
}

func (c *SearchClient) searchDuckDuckGo(ctx context.Context, query string, limit int) ([]SearchItem, error) {
	endpoint := "https://duckduckgo.com/?q=%s&format=json&no_html=1&no_redirect=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(endpoint, url.QueryEscape(query)), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("DuckDuckGo 接口返回状态码 %d", resp.StatusCode)
	}

	var parsed struct {
		RelatedTopics []struct {
			FirstURL string `json:"FirstURL"`
			Text     string `json:"Text"`
		} `json:"RelatedTopics"`
		AbstractURL  string `json:"AbstractURL"`
		AbstractText string `json:"AbstractText"`
		Redirect     string `json:"Redirect"`
		Results      []struct {
			FirstURL string `json:"FirstURL"`
			Text     string `json:"Text"`
		} `json:"Results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	results := make([]SearchItem, 0, limit)
	if parsed.AbstractURL != "" {
		results = append(results, SearchItem{
			Title:   parsed.AbstractText,
			URL:     parsed.AbstractURL,
			Snippet: parsed.AbstractText,
		})
	}

	for _, item := range parsed.Results {
		if len(results) >= limit {
			break
		}
		results = append(results, SearchItem{
			Title:   item.Text,
			URL:     item.FirstURL,
			Snippet: item.Text,
		})
	}

	for _, item := range parsed.RelatedTopics {
		if len(results) >= limit {
			break
		}
		results = append(results, SearchItem{
			Title:   item.Text,
			URL:     item.FirstURL,
			Snippet: item.Text,
		})
	}

	if len(results) == 0 && parsed.Redirect != "" {
		results = append(results, SearchItem{
			Title:   "DuckDuckGo Redirect",
			URL:     parsed.Redirect,
			Snippet: "DuckDuckGo redirect result",
		})
	}

	return results, nil
}

func directMode(input string) []SearchItem {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}
	urlValue := input
	if !strings.HasPrefix(urlValue, "http://") && !strings.HasPrefix(urlValue, "https://") {
		urlValue = "https://" + urlValue
	}
	return []SearchItem{{
		Title:   "Direct URL",
		URL:     urlValue,
		Snippet: "搜索未配置，已进入直连模式。",
	}}
}

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
}

// NewSearchClient constructs a search client.
func NewSearchClient(st *store.Store, client *http.Client) *SearchClient {
	if client == nil {
		client = &http.Client{Timeout: defaultSearchTimeout}
	}
	return &SearchClient{
		store:      st,
		httpClient: client,
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

	switch provider {
	case "bing":
		return c.searchBing(ctx, query, limit, apiKey)
	case "serpapi":
		return c.searchSerpAPI(ctx, query, limit, apiKey)
	case "ddg", "duckduckgo":
		return c.searchDuckDuckGo(ctx, query, limit)
	case "":
		return directMode(query), nil
	default:
		return nil, fmt.Errorf("暂不支持的搜索提供商: %s", provider)
	}
}

// TestSearch validates that the configured provider can return at least one result.
func (c *SearchClient) TestSearch(ctx context.Context) error {
	_, err := c.Search(ctx, "Apple Inc.", 1)
	if err != nil {
		return err
	}
	return nil
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

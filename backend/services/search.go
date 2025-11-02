package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

const defaultSearchTimeout = 25 * time.Second

// SearchStage enumerates the predefined concurrent search tracks.
type SearchStage string

const (
	SearchStageBroad    SearchStage = "broad_discovery" // 任务A
	SearchStageWebsite  SearchStage = "website_focus"   // 任务B
	SearchStageContacts SearchStage = "decision_makers" // 任务C
	SearchStageLinkedIn SearchStage = "linkedin_audit"  // 任务D
)

type searchTaskSpec struct {
	Stage        SearchStage
	Label        string
	QueryBuilder func(string) string
	Limit        int
}

var defaultSearchPlan = []searchTaskSpec{
	{
		Stage: SearchStageBroad,
		Label: "任务A：基础信息搜索",
		Limit: 6,
		QueryBuilder: func(name string) string {
			return strings.TrimSpace(name)
		},
	},
	{
		Stage: SearchStageWebsite,
		Label: "任务B：官网及业务搜索",
		Limit: 6,
		QueryBuilder: func(name string) string {
			name = strings.TrimSpace(name)
			if name == "" {
				return ""
			}
			return fmt.Sprintf(`%s official website OR about us OR products`, name)
		},
	},
	{
		Stage: SearchStageContacts,
		Label: "任务C：关键联系人搜索",
		Limit: 6,
		QueryBuilder: func(name string) string {
			name = strings.TrimSpace(name)
			if name == "" {
				return ""
			}
			return fmt.Sprintf(`("owner" OR "founder" OR "CEO" OR "purchasing manager" OR "sourcing director") AND "%s"`, name)
		},
	},
	{
		Stage: SearchStageLinkedIn,
		Label: "任务D：社交与职业背景搜索",
		Limit: 6,
		QueryBuilder: func(name string) string {
			name = strings.TrimSpace(name)
			if name == "" {
				return ""
			}
			return fmt.Sprintf(`site:linkedin.com "%s"`, name)
		},
	},
}

func stageLabel(stage SearchStage) string {
	for _, spec := range defaultSearchPlan {
		if spec.Stage == stage {
			return spec.Label
		}
	}
	return strings.ToUpper(string(stage))
}

// SearchItem represents a single search result snippet.
type SearchItem struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// SearchTaskResult captures the outcome of an individual search task.
type SearchTaskResult struct {
	Stage    SearchStage
	Label    string
	Query    string
	Items    []SearchItem
	Error    error
	Duration time.Duration
}

// SearchPlanResult contains the aggregated output of all predefined tasks.
type SearchPlanResult struct {
	Customer     string
	Provider     string
	StageResults map[SearchStage]*SearchTaskResult
	Order        []SearchStage
	combined     []SearchItem
}

// Combined returns the deduplicated aggregate search results in plan order.
func (r *SearchPlanResult) Combined() []SearchItem {
	if r == nil {
		return nil
	}
	cp := make([]SearchItem, len(r.combined))
	copy(cp, r.combined)
	return cp
}

// Result returns the task result for the provided stage.
func (r *SearchPlanResult) Result(stage SearchStage) *SearchTaskResult {
	if r == nil {
		return nil
	}
	return r.StageResults[stage]
}

// Items returns the search items for the provided stage.
func (r *SearchPlanResult) Items(stage SearchStage) []SearchItem {
	res := r.Result(stage)
	if res == nil {
		return nil
	}
	return res.Items
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

// ExecutePlan runs the predefined parallel search plan.
func (c *SearchClient) ExecutePlan(ctx context.Context, customerName string) (*SearchPlanResult, error) {
	if c == nil || c.store == nil {
		return nil, fmt.Errorf("search client not initialized")
	}
	customerName = strings.TrimSpace(customerName)
	if customerName == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}

	settings, err := c.store.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("load settings: %w", err)
	}

	provider, providerLabel := normalizeProvider(settings.SearchProvider)
	apiKey := strings.TrimSpace(settings.SearchAPIKey)

	result := &SearchPlanResult{
		Customer:     customerName,
		Provider:     providerLabel,
		StageResults: make(map[SearchStage]*SearchTaskResult),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, spec := range defaultSearchPlan {
		query := spec.QueryBuilder(customerName)
		if strings.TrimSpace(query) == "" {
			continue
		}
		wg.Add(1)
		go func(spec searchTaskSpec, query string) {
			defer wg.Done()
			start := time.Now()
			items, err := c.searchSingle(ctx, provider, apiKey, query, spec.Limit)
			if err != nil {
				log.Printf("[search] stage=%s query=\"%s\" error=%v", spec.Stage, truncateForLog(query, 80), err)
			} else {
				log.Printf("[search] stage=%s query=\"%s\" provider=%s results=%d", spec.Stage, truncateForLog(query, 80), providerLabel, len(items))
			}
			mu.Lock()
			result.StageResults[spec.Stage] = &SearchTaskResult{
				Stage:    spec.Stage,
				Label:    spec.Label,
				Query:    query,
				Items:    items,
				Error:    err,
				Duration: time.Since(start),
			}
			mu.Unlock()
		}(spec, query)
	}

	wg.Wait()

	seen := map[string]struct{}{}
	for _, spec := range defaultSearchPlan {
		res, ok := result.StageResults[spec.Stage]
		if !ok || len(res.Items) == 0 {
			continue
		}
		result.Order = append(result.Order, spec.Stage)
		result.combined = appendDedup(result.combined, res.Items, seen, 0)
	}

	if len(result.combined) == 0 {
		return nil, fmt.Errorf("所有搜索任务均未返回有效结果")
	}

	return result, nil
}

// Search executes a single ad-hoc query using the configured provider.
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
	provider, _ := normalizeProvider(settings.SearchProvider)
	apiKey := strings.TrimSpace(settings.SearchAPIKey)
	return c.searchSingle(ctx, provider, apiKey, query, limit)
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

	provider := strings.TrimSpace(settings.SearchProvider)
	if provider == "" {
		return fmt.Errorf("请先在设置中配置搜索提供商")
	}

	plan, err := c.ExecutePlan(ctx, "Apple Inc.")
	if err != nil {
		return err
	}
	if len(plan.combined) < 3 {
		return fmt.Errorf("搜索结果不足 3 条，请检查搜索提供商或关键词")
	}
	return nil
}

// ------------------------------------------------------------------------
// Provider adapters and helpers
// ------------------------------------------------------------------------

func (c *SearchClient) searchSingle(ctx context.Context, provider, apiKey, query string, limit int) ([]SearchItem, error) {
	providerLabel := provider
	if providerLabel == "" {
		providerLabel = "direct"
	}
	log.Printf("[search] provider=%s query=\"%s\" limit=%d status=started", providerLabel, truncateForLog(query, 80), limit)

	switch provider {
	case "serpapi":
		return c.searchSerpAPI(ctx, query, limit, apiKey)
	case "bing":
		return c.searchBing(ctx, query, limit, apiKey)
	case "ddg":
		return c.searchDuckDuckGo(ctx, query, limit)
	case "":
		return directMode(query), nil
	default:
		log.Printf("[search] provider=%s status=unsupported, fallback=direct", provider)
		return directMode(query), nil
	}
}

func (c *SearchClient) searchBing(ctx context.Context, query string, limit int, apiKey string) ([]SearchItem, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("Bing 搜索需要配置 API Key")
	}
	if limit <= 0 {
		limit = 5
	}
	endpoint := "https://api.bing.microsoft.com/v7.0/search"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("q", query)
	q.Set("count", strconv.Itoa(limit))
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Ocp-Apim-Subscription-Key", apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Bing 搜索失败: %s", resp.Status)
	}

	var payload struct {
		WebPages struct {
			Value []struct {
				Name        string `json:"name"`
				URL         string `json:"url"`
				Snippet     string `json:"snippet"`
				DisplayURL  string `json:"displayUrl"`
				Description string `json:"description"`
			} `json:"value"`
		} `json:"webPages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	items := make([]SearchItem, 0, len(payload.WebPages.Value))
	for _, value := range payload.WebPages.Value {
		url := strings.TrimSpace(value.URL)
		if url == "" {
			url = strings.TrimSpace(value.DisplayURL)
		}
		items = append(items, SearchItem{
			Title:   strings.TrimSpace(value.Name),
			URL:     url,
			Snippet: strings.TrimSpace(firstNonEmpty(value.Snippet, value.Description)),
		})
	}
	return items, nil
}

func (c *SearchClient) searchSerpAPI(ctx context.Context, query string, limit int, apiKey string) ([]SearchItem, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("SERP API 需要配置 API Key")
	}
	if limit <= 0 {
		limit = 5
	}

	endpoint := "https://serpapi.com/search.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("engine", "google")
	q.Set("google_domain", "google.com")
	q.Set("q", query)
	q.Set("num", strconv.Itoa(limit))
	q.Set("api_key", apiKey)
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("SERP API 搜索失败: %s", resp.Status)
	}

	var payload struct {
		OrganicResults []struct {
			Title       string `json:"title"`
			Link        string `json:"link"`
			Snippet     string `json:"snippet"`
			Description string `json:"description"`
		} `json:"organic_results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	items := make([]SearchItem, 0, len(payload.OrganicResults))
	for _, value := range payload.OrganicResults {
		items = append(items, SearchItem{
			Title:   strings.TrimSpace(value.Title),
			URL:     strings.TrimSpace(value.Link),
			Snippet: strings.TrimSpace(firstNonEmpty(value.Snippet, value.Description)),
		})
	}
	return items, nil
}

func (c *SearchClient) searchDuckDuckGo(ctx context.Context, query string, limit int) ([]SearchItem, error) {
	if limit <= 0 {
		limit = 5
	}
	endpoint := "https://duckduckgo.com/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("q", query)
	q.Set("format", "json")
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("DuckDuckGo 搜索失败: %s", resp.Status)
	}

	var payload struct {
		RelatedTopics []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	items := make([]SearchItem, 0, len(payload.RelatedTopics))
	for _, topic := range payload.RelatedTopics {
		if strings.TrimSpace(topic.FirstURL) == "" {
			continue
		}
		items = append(items, SearchItem{
			Title:   strings.TrimSpace(topic.Text),
			URL:     strings.TrimSpace(topic.FirstURL),
			Snippet: strings.TrimSpace(topic.Text),
		})
		if len(items) >= limit {
			break
		}
	}
	return items, nil
}

// directMode is a minimal fallback used when no provider is configured.
func directMode(query string) []SearchItem {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil
	}
	normalized, err := normalizeURL(trimmed)
	if err != nil {
		normalized = trimmed
	}
	return []SearchItem{{
		Title:   trimmed,
		URL:     normalized,
		Snippet: "用户输入的原始关键词。未配置外部搜索服务，结果有限。",
	}}
}

func appendDedup(dest []SearchItem, items []SearchItem, seen map[string]struct{}, max int) []SearchItem {
	for _, item := range items {
		key := searchItemKey(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dest = append(dest, item)
		if max > 0 && len(dest) >= max {
			break
		}
	}
	return dest
}

func searchItemKey(item SearchItem) string {
	if normalized := strings.TrimSpace(strings.ToLower(normalizeMaybe(item.URL))); normalized != "" {
		return normalized
	}
	if snippet := normalizeSnippet(item.Snippet); snippet != "" {
		return snippet
	}
	if title := strings.TrimSpace(strings.ToLower(item.Title)); title != "" {
		return title
	}
	return ""
}

func normalizeSnippet(snippet string) string {
	clean := strings.TrimSpace(snippet)
	if clean == "" {
		return ""
	}
	if strings.HasPrefix(clean, "[") {
		if idx := strings.Index(clean, "]"); idx >= 0 && idx < len(clean)-1 {
			clean = strings.TrimSpace(clean[idx+1:])
		}
	}
	return strings.ToLower(clean)
}

func normalizeProvider(raw string) (provider string, label string) {
	base := strings.ToLower(strings.TrimSpace(raw))
	switch base {
	case "", "google":
		return "serpapi", "google"
	case "serpapi":
		return "serpapi", "google"
	case "bing":
		return "bing", "bing"
	case "duckduckgo", "ddg":
		return "ddg", "duckduckgo"
	default:
		return base, base
	}
}

func truncateForLog(text string, max int) string {
	text = strings.TrimSpace(text)
	if max <= 0 || len([]rune(text)) <= max {
		return text
	}
	runes := []rune(text)
	return string(runes[:max]) + "…"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

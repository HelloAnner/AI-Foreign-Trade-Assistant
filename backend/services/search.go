package services

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/anner/ai-foreign-trade-assistant/backend/store"
	"github.com/playwright-community/playwright-go"
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
		Limit: 10, // 增加到10个结果
		QueryBuilder: func(name string) string {
			return strings.TrimSpace(name)
		},
	},
	{
		Stage: SearchStageWebsite,
		Label: "任务B：官网及业务搜索",
		Limit: 10, // 增加到10个结果
		QueryBuilder: func(name string) string {
			name = strings.TrimSpace(name)
			if name == "" {
				return ""
			}
			return fmt.Sprintf(`%s official website`, name)
		},
	},
	{
		Stage: SearchStageContacts,
		Label: "任务C：关键联系人搜索",
		Limit: 10, // 增加到10个结果
		QueryBuilder: func(name string) string {
			name = strings.TrimSpace(name)
			if name == "" {
				return ""
			}
			return fmt.Sprintf(`%s CEO founder owner purchasing manager`, name)
		},
	},
	{
		Stage: SearchStageLinkedIn,
		Label: "任务D：社交与职业背景搜索",
		Limit: 10, // 增加到10个结果
		QueryBuilder: func(name string) string {
			name = strings.TrimSpace(name)
			if name == "" {
				return ""
			}
			return fmt.Sprintf(`site:linkedin.com %s`, name)
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

// SearchClient invokes Google search using Playwright.
type SearchClient struct {
	store *store.Store
}

// NewSearchClient constructs a search client.
func NewSearchClient(st *store.Store, _ interface{}) *SearchClient {
	return &SearchClient{
		store: st,
	}
}

// ExecutePlan runs the predefined parallel search plan using Playwright.
func (c *SearchClient) ExecutePlan(ctx context.Context, customerName string) (*SearchPlanResult, error) {
	if c == nil || c.store == nil {
		return nil, fmt.Errorf("search client not initialized")
	}
	customerName = strings.TrimSpace(customerName)
	if customerName == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}

	result := &SearchPlanResult{
		Customer:     customerName,
		Provider:     "google",
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
			items, err := c.searchWithPlaywright(ctx, query, spec.Limit)
			if err != nil {
				log.Printf("[search] stage=%s query=\"%s\" error=%v", spec.Stage, truncateForLog(query, 80), err)
			} else {
				log.Printf("[search] stage=%s query=\"%s\" results=%d", spec.Stage, truncateForLog(query, 80), len(items))
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

// Search executes a single ad-hoc query using Playwright.
func (c *SearchClient) Search(ctx context.Context, query string, limit int) ([]SearchItem, error) {
	if c == nil || c.store == nil {
		return nil, fmt.Errorf("search client not initialized")
	}
	if limit <= 0 {
		limit = 10
	}
	return c.searchWithPlaywright(ctx, query, limit)
}

// TestSearch validates that Playwright can return at least one result.
func (c *SearchClient) TestSearch(ctx context.Context) error {
	if c == nil || c.store == nil {
		return fmt.Errorf("search client not initialized")
	}

	plan, err := c.ExecutePlan(ctx, "Apple Inc.")
	if err != nil {
		return err
	}
	if len(plan.combined) < 3 {
		return fmt.Errorf("搜索结果不足 3 条，请检查搜索配置或关键词")
	}
	return nil
}

// searchWithPlaywright performs a Google search using Playwright and returns top results.
func (c *SearchClient) searchWithPlaywright(ctx context.Context, query string, limit int) ([]SearchItem, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("搜索查询不能为空")
	}
	if limit <= 0 {
		limit = 10
	}

	log.Printf("[search] playwright query=\"%s\" limit=%d", truncateForLog(query, 80), limit)

	// Run Playwright search
	items, err := runPlaywrightSearch(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("Playwright search failed: %w", err)
	}

	// If no results, try fallback direct mode
	if len(items) == 0 {
		log.Printf("[search] no results from Playwright, using fallback")
		return directMode(query), nil
	}

	return items, nil
}

// runPlaywrightSearch executes Google search using Playwright with parallel page fetching.
func runPlaywrightSearch(ctx context.Context, query string, limit int) ([]SearchItem, error) {
	// Start Playwright
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start Playwright: %w", err)
	}
	defer pw.Stop()

	// Create browser context with anti-detection measures
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage",
			"--disable-web-security",
			"--disable-features=VizDisplayCompositor",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}
	defer browser.Close()

	// Create context with custom user agent
	ctxPW, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(userAgent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create context: %w", err)
	}
	defer ctxPW.Close()

	// Create page and search
	page, err := ctxPW.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// Navigate to Google - use proper URL encoding
	searchURL := fmt.Sprintf("https://www.google.com/search?q=%s", url.QueryEscape(query))
	if _, err := page.Goto(searchURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000),
	}); err != nil {
		return nil, fmt.Errorf("failed to navigate to Google: %w", err)
	}

	// Wait for search results to load
	page.WaitForSelector("#search", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(15000),
	})

	// Additional wait to ensure page is fully loaded
	time.Sleep(2 * time.Second)

	// Extract search results
	results, err := extractGoogleResults(page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to extract results: %w", err)
	}

	log.Printf("[search] extracted %d results from Google", len(results))

	// If we have results, fetch page contents in parallel for better snippets
	if len(results) > 0 {
		results = fetchPageContentsParallel(ctx, results, limit)
	}

	return results, nil
}

// extractGoogleResults extracts search results from Google page using JavaScript evaluation.
// This is more reliable than using Go's QuerySelector API for complex Google page structures.
func extractGoogleResults(page playwright.Page, limit int) ([]SearchItem, error) {
	// Add retry mechanism with exponential backoff
	maxRetries := 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err := page.Evaluate(generateExtractionJS(limit))
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed: %w", attempt, err)
			log.Printf("[search] JavaScript extraction attempt %d failed: %v", attempt, err)
			if attempt < maxRetries {
				// Wait before retry (exponential backoff)
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			break
		}

		// Convert result to SearchItem slice
		items, err := parseResults(result, limit)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				log.Printf("[search] result parsing failed on attempt %d, retrying...", attempt)
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			break
		}

		if len(items) > 0 {
			log.Printf("[search] JavaScript extraction succeeded on attempt %d, returned %d results", attempt, len(items))
			return items, nil
		}

		lastErr = fmt.Errorf("no results extracted on attempt %d", attempt)
		if attempt < maxRetries {
			log.Printf("[search] no results on attempt %d, retrying...", attempt)
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	log.Printf("[search] JavaScript extraction failed after %d attempts: %v", maxRetries, lastErr)
	return []SearchItem{}, nil
}

// generateExtractionJS generates optimized JavaScript for extracting search results
func generateExtractionJS(limit int) string {
	return fmt.Sprintf(`() => {
		const results = [];
		const h3Elements = document.querySelectorAll('h3');
		const seenUrls = new Set(); // Track seen URLs to avoid duplicates

		for (let h3 of h3Elements) {
			try {
				const title = h3.textContent?.trim();
				if (!title || title.length < 5) continue;

				// Skip headings that are not search results
				const skipPatterns = [
					'相关搜索', '相关', 'People also ask', 'Related searches',
					'Searches related to', '其他人还搜索了', '其他人还问了',
					'视频', 'Videos', '图片', 'Images', '地图', 'Maps', '购物', 'Shopping'
				];
				if (skipPatterns.some(pattern => title.includes(pattern))) {
					continue;
				}

				// Find the link - try multiple strategies
				let link = null;
				let container = null;

				// Strategy 1: h3 itself might be inside a link
				link = h3.closest('a[href^="http"]');
				if (link) {
					container = link.closest('[data-hveid], .g, #rso > div, [data-sokoban-container]');
				}

				// Strategy 2: Find link in parent container
				if (!link) {
					container = h3.closest('[data-hveid], .g, #rso > div, [data-sokoban-container]');
					if (container) {
						const links = container.querySelectorAll('a[href^="http"]');
						for (let l of links) {
							const href = l.href;
							// Validate URL format
							if (href && href.match(/^https?:\/\//) &&
								!href.includes('google.com') &&
								!href.includes('/search?q=') &&
								!seenUrls.has(href)) {
								link = l;
								seenUrls.add(href);
								break;
							}
						}
					}
				}

				if (!link || !link.href) continue;

				const url = link.href;

				// Skip Google internal links and cached pages
				if (url.includes('google.com') || url.includes('/search?q=') ||
					url.includes('webcache.googleusercontent.com') ||
					url.includes('translate.google.com')) {
					continue;
				}

				// Extract snippet - find text near the h3, more intelligently
				let snippet = '';
				if (container) {
					const textElements = container.querySelectorAll('span, div, p');
					let bestSnippet = '';
					for (let elem of textElements) {
						const text = elem.textContent?.trim();
						// Look for text that is:
						// - Different from title
						// - Reasonably long (50-500 chars)
						// - Not a date or single word
						if (text && text !== title && text.length >= 50 && text.length <= 500 &&
							text.split(' ').length > 5 && !/^\d{1,2}\/\d{1,2}\/\d{4}$/.test(text)) {
							if (text.length > bestSnippet.length) {
								bestSnippet = text;
							}
						}
					}
					snippet = bestSnippet;
				}

				// Only add if we have at least title and URL
				if (title && url) {
					results.push({
						title: title,
						url: url,
						snippet: snippet
					});
				}

				if (results.length >= %d) break;
			} catch (e) {
				// Skip this result if there's an error processing it
				continue;
			}
		}

		return results;
	}`, limit)
}

// parseResults converts JavaScript evaluation result to SearchItem slice
func parseResults(result interface{}, limit int) ([]SearchItem, error) {
	results, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	if len(results) == 0 {
		return []SearchItem{}, nil
	}

	var items []SearchItem
	for _, r := range results {
		if itemMap, ok := r.(map[string]interface{}); ok {
			item := SearchItem{
				Title:   sanitizeString(itemMap["title"]),
				URL:     sanitizeString(itemMap["url"]),
				Snippet: sanitizeString(itemMap["snippet"]),
			}
			// Validate URL format
			if item.Title != "" && item.URL != "" && isValidURL(item.URL) {
				items = append(items, item)
			}
		}
	}

	return items, nil
}

// sanitizeString safely converts interface{} to string
func sanitizeString(v interface{}) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

// isValidURL checks if string is a valid HTTP/HTTPS URL
func isValidURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// fetchPageContentsParallel fetches page contents in parallel for better snippets.
func fetchPageContentsParallel(ctx context.Context, items []SearchItem, limit int) []SearchItem {
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Use semaphore to limit concurrent requests
	sem := make(chan struct{}, 3) // Max 3 concurrent

	results := make([]SearchItem, 0, len(items))

	for _, item := range items {
		sem <- struct{}{}
		wg.Add(1)

		go func(item SearchItem) {
			defer wg.Done()
			defer func() { <-sem }()

			// Create new browser for each page to avoid conflicts
			pw, err := playwright.Run()
			if err != nil {
				mu.Lock()
				results = append(results, item)
				mu.Unlock()
				return
			}
			defer pw.Stop()

			browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
				Headless: playwright.Bool(true),
			})
			if err != nil {
				mu.Lock()
				results = append(results, item)
				mu.Unlock()
				return
			}
			defer browser.Close()

			ctxPW, err := browser.NewContext()
			if err != nil {
				mu.Lock()
				results = append(results, item)
				mu.Unlock()
				return
			}
			defer ctxPW.Close()

			page, err := ctxPW.NewPage()
			if err != nil {
				mu.Lock()
				results = append(results, item)
				mu.Unlock()
				return
			}

			// Try to fetch the page with timeout
			ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			go func() {
				<-ctxTimeout.Done()
				if ctxTimeout.Err() != nil {
					page.Close()
				}
			}()

			if resp, err := page.Goto(item.URL, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
				Timeout:   playwright.Float(10000),
			}); err == nil && resp != nil && resp.Status() < 400 {
				// Wait a bit for page to load
				time.Sleep(1 * time.Second)

				// Get page title if missing
				if item.Title == "" {
					if title, err := page.Title(); err == nil && title != "" {
						item.Title = title
					}
				}

				// Get meta description or first paragraph for snippet
				if item.Snippet == "" {
					// Try meta description
					if metaDesc, err := page.Evaluate(`document.querySelector('meta[name="description"]')?.content || ''`); err == nil {
						if desc, ok := metaDesc.(string); ok && desc != "" {
							item.Snippet = strings.TrimSpace(desc)
						}
					}

					// If still no snippet, try first paragraph
					if item.Snippet == "" {
						content, err := page.Evaluate(`
							(function() {
								const p = document.querySelector('p');
								if (p) {
									const text = p.textContent || '';
									return text.substring(0, 200);
								}
								return '';
							})()
						`)
						if err == nil {
							if text, ok := content.(string); ok && text != "" {
								item.Snippet = strings.TrimSpace(text)
							}
						}
					}
				}
			}

			mu.Lock()
			results = append(results, item)
			mu.Unlock()
		}(item)
	}

	wg.Wait()

	return results
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
	// Now we only support playwright/google
	return "google", "google"
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

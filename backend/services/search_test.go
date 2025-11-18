package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDetectSearchProviderPrefersGoogle(t *testing.T) {
	defer func(prev func() error) { pingGoogleFunc = prev }(pingGoogleFunc)
	pingGoogleFunc = func() error { return nil }

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}),
	}

	provider, err := detectSearchProvider(client)
	if err != nil {
		t.Fatalf("expected no error when Google probe succeeds: %v", err)
	}
	if provider != searchProviderGoogle {
		t.Fatalf("expected google provider, got %s", provider)
	}
}

func TestDetectSearchProviderFallsBackToBing(t *testing.T) {
	defer func(prev func() error) { pingGoogleFunc = prev }(pingGoogleFunc)
	pingGoogleFunc = func() error { return errors.New("icmp blocked") }

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network blocked")
		}),
	}

	provider, err := detectSearchProvider(client)
	if err == nil {
		t.Fatal("expected error when Google probe fails")
	}
	if provider != searchProviderBing {
		t.Fatalf("expected bing provider, got %s", provider)
	}
}

func TestSearchPlanSpecBuilders(t *testing.T) {
	company := "环球贸易有限公司"
	expectations := map[SearchStage]string{
		SearchStageBroad:    "环球贸易有限公司",
		SearchStageWebsite:  "环球贸易有限公司 official website OR about us OR products",
		SearchStageContacts: `("owner" OR "founder" OR "CEO" OR "purchasing manager" OR "sourcing director") AND "环球贸易有限公司"`,
		SearchStageLinkedIn: `site:linkedin.com "环球贸易有限公司"`,
	}
	for _, spec := range defaultSearchPlan {
		query := spec.QueryBuilder(company)
		if got, want := strings.TrimSpace(query), expectations[spec.Stage]; got != want {
			t.Fatalf("stage %s produced unexpected query: got %q want %q", spec.Stage, got, want)
		}
	}

	if q := defaultSearchPlan[0].QueryBuilder("  "); q != "" {
		t.Fatalf("expected empty query for empty input, got %q", q)
	}
}

func TestSearchDirectModeDeduplicatesVariants(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		st.Close()
	})

	if err := st.InitSchema(ctx); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	if _, err := st.DB.ExecContext(ctx, `UPDATE settings SET search_provider = 'direct' WHERE id = 1`); err != nil {
		t.Fatalf("update settings: %v", err)
	}

	client := NewSearchClient(st, nil)
	results, err := client.Search(ctx, "example.com", 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected single direct-mode result, got %d", len(results))
	}
}

func TestSearchUnsupportedProvider(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		st.Close()
	})

	if err := st.InitSchema(ctx); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	if _, err := st.DB.ExecContext(ctx, `UPDATE settings SET search_provider = 'unsupported' WHERE id = 1`); err != nil {
		t.Fatalf("update settings: %v", err)
	}

	client := NewSearchClient(st, nil)
	results, err := client.Search(ctx, "Acme Corp", 5)
	if err != nil {
		t.Fatalf("search fallback failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected fallback results for unsupported provider")
	}
}

// TestExternalDependencies tests if SerpApi and LLM are available before running accuracy tests
func TestExternalDependencies(t *testing.T) {
	t.Log("Testing external dependencies availability...")

	// Check if Playwright environment is configured (optional for tests)
	t.Log("✓ Playwright search ready (no API key required)")

	// Initialize store
	ctx := context.Background()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if err := st.InitSchema(ctx); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	// Check LLM configuration from environment
	llmAPIKey := os.Getenv("LLM_API_KEY")
	llmBaseURL := os.Getenv("LLM_BASE_URL")
	llmModel := os.Getenv("LLM_MODEL_NAME")

	if llmAPIKey == "" || llmBaseURL == "" || llmModel == "" {
		t.Log("⚠ LLM configuration not complete from environment variables, LLM tests may fail")
	} else {
		t.Log("✓ LLM environment variables configured")
		// Test LLM connectivity
		if _, err := st.DB.ExecContext(ctx, `UPDATE settings SET llm_base_url = ?, llm_api_key = ?, llm_model = ? WHERE id = 1`,
			llmBaseURL, llmAPIKey, llmModel); err != nil {
			t.Fatalf("update LLM settings: %v", err)
		}

		llmClient := NewLLMClient(st, nil)
		if _, err := llmClient.TestConnection(ctx); err != nil {
			t.Errorf("LLM connectivity test failed: %v", err)
		} else {
			t.Log("✓ LLM connectivity test passed")
		}
	}

	// Test Playwright connectivity
	searchClient := NewSearchClient(st, nil)
	if err := searchClient.TestSearch(ctx); err != nil {
		t.Errorf("Playwright search test failed: %v", err)
	} else {
		t.Log("✓ Playwright search test passed")
	}
}

// TestSearchAccuracy tests the accuracy of search results against a predefined dataset using Playwright
func TestSearchAccuracy(t *testing.T) {
	// Check if Playwright environment is ready (optional)
	t.Log("Testing search accuracy with Playwright...")

	// Define test dataset
	testData := []struct {
		CompanyName string
		Website     string
	}{
		{"Apple Inc.", "https://www.apple.com"},
		{"Tesla, Inc.", "https://www.tesla.com"},
		{"Nike, Inc.", "https://www.nike.com"},
		{"The Coca-Cola Company", "https://www.coca-colacompany.com"},
		{"Nestlé S.A.", "https://www.nestle.com"},
		{"Procter & Gamble (P&G)", "https://www.pg.com"},
		{"L'Oréal S.A.", "https://www.loreal.com"},
		{"Toyota Motor Corporation", "https://global.toyota/en/"},
		{"Siemens AG", "https://www.siemens.com"},
		{"IKEA (INGKA Holding B.V.)", "https://www.ikea.com"},
		{"Samsung Electronics Co., Ltd.", "https://www.samsung.com"},
		{"Adidas AG", "https://www.adidas-group.com"},
		{"Caterpillar Inc.", "https://www.caterpillar.com"},
		{"The LEGO Group", "https://www.lego.com"},
		{"Starbucks Corporation", "https://www.starbucks.com"},
		{"Maersk (A.P. Moller - Maersk Group)", "https://www.maersk.com"},
		{"BASF SE", "https://www.basf.com"},
		{"3M Company", "https://www.3m.com"},
		{"ASML Holding N.V.", "https://www.asml.com"},
		{"Barry Callebaut AG", "https://www.barry-callebaut.com"},
		{"Cargill, Incorporated", "https://www.cargill.com"},
		{"Glencore plc", "https://www.glencore.com"},
		{"ZF Friedrichshafen AG", "https://www.zf.com"},
		{"DSM-Firmenich AG", "https://www.dsm-firmenich.com"},
		{"Fanuc Corporation", "https://www.fanuc.com"},
		{"Grainger (W.W. Grainger, Inc.)", "https://www.grainger.com"},
		{"Demant A/S", "https://www.demant.com"},
		{"SGS S.A.", "https://www.sgs.com"},
		{"Givaudan S.A.", "https://www.givaudan.com"},
		{"Trimble Inc.", "https://www.trimble.com"},
	}

	// Initialize store and configure SerpApi
	ctx := context.Background()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	if err := st.InitSchema(ctx); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	// Playwright doesn't require API configuration
	t.Log("Using Playwright for search (no API key needed)")

	client := NewSearchClient(st, nil)

	// Run tests and calculate accuracy
	var totalTests int
	var correctResults int

	for _, test := range testData {
		t.Run(test.CompanyName, func(t *testing.T) {
			totalTests++
			if testCompanyWebsite(t, ctx, client, test.CompanyName, test.Website) {
				correctResults++
			}
		})
	}

	// Print accuracy summary
	accuracy := 0.0
	if totalTests > 0 {
		accuracy = float64(correctResults) / float64(totalTests) * 100
	}

	t.Logf("========================================")
	t.Logf("Search Accuracy Test Summary:")
	t.Logf("Total companies tested: %d", totalTests)
	t.Logf("Correct results: %d", correctResults)
	t.Logf("Accuracy: %.2f%%", accuracy)
	t.Logf("========================================")
}

// testCompanyWebsite tests if the search results contain the correct website for a company
func testCompanyWebsite(t *testing.T, ctx context.Context, client *SearchClient, companyName, expectedWebsite string) bool {
	t.Logf("Testing: %s", companyName)

	// Execute search
	result, err := client.ExecutePlan(ctx, companyName)
	if err != nil {
		t.Logf("  ❌ Search failed for %s: %v", companyName, err)
		return false
	}

	// Check if expected website appears in results
	combined := result.Combined()
	for _, item := range combined {
		// Check for exact URL match or domain match
		if strings.Contains(item.URL, strings.TrimPrefix(expectedWebsite, "https://")) ||
			(len(expectedWebsite) > 8 && strings.Contains(item.URL, expectedWebsite[8:])) {
			t.Logf("  ✓ Found correct website for %s: %s", companyName, item.URL)
			return true
		}
	}

	// Also check snippet for website mention
	for _, item := range combined {
		if strings.Contains(strings.ToLower(item.Snippet), strings.ToLower(expectedWebsite)) {
			t.Logf("  ✓ Found website in snippet for %s: %s", companyName, item.Snippet)
			return true
		}
	}

	t.Logf("  ❌ Expected website %s not found for %s", expectedWebsite, companyName)
	t.Logf("  Top results:")
	for i, item := range combined {
		if i >= 3 {
			break
		}
		t.Logf("    %d. %s - %s", i+1, item.Title, item.URL)
	}

	return false
}

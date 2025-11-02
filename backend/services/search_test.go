package services

import (
	"context"
	"strings"
	"testing"

	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

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

package services

import (
	"context"
	"testing"

	"github.com/anner/ai-foreign-trade-assistant/backend/store"
)

func TestBuildSearchVariants(t *testing.T) {
	variants := buildSearchVariants("环球贸易有限公司")
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}
	if variants[0] != "环球贸易有限公司" {
		t.Fatalf("unexpected first variant: %s", variants[0])
	}
	if variants[1] != "环球贸易有限公司 联系人" {
		t.Fatalf("unexpected second variant: %s", variants[1])
	}

	variants = buildSearchVariants("上海制造联系人")
	if len(variants) != 1 {
		t.Fatalf("expected 1 variant when query already contains 联系人, got %d", len(variants))
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
	if _, err := client.Search(ctx, "Acme Corp", 5); err == nil {
		t.Fatal("expected error for unsupported provider, got nil")
	}
}

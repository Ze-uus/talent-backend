package store_test

import (
	"testing"

	"github.com/Ze-uus/talent-backend/internal/store"
)

func TestNormalizeAndValidateContent(t *testing.T) {
	content, err := store.NormalizeAndValidateContent([]store.ContentItem{{
		ID: " hero ",
		Images: []string{
			" https://cdn.example.com/one.jpg ",
			"https://cdn.example.com/two.jpg",
		},
		Links: []store.ContentLink{{
			ID: "cta", Label: "Learn more", URL: "https://example.com/start",
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if content[0].ID != "hero" || content[0].Images[0] != "https://cdn.example.com/one.jpg" {
		t.Fatalf("content was not normalized: %+v", content)
	}
}

func TestNormalizeAndValidateContentRejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name    string
		content []store.ContentItem
	}{
		{"duplicate ids", []store.ContentItem{{ID: "same"}, {ID: "same"}}},
		{"javascript link", []store.ContentItem{{
			ID: "one", Links: []store.ContentLink{{ID: "cta", URL: "javascript:alert(1)"}},
		}}},
		{"html", []store.ContentItem{{ID: "one", Title: "<script>alert(1)</script>"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := store.NormalizeAndValidateContent(tt.content); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestEffectiveContentUsesCycleOverride(t *testing.T) {
	campaign := store.Campaign{Content: []store.ContentItem{{ID: "campaign"}}}
	if got := store.EffectiveContent(campaign, store.Cycle{}); len(got) != 1 || got[0].ID != "campaign" {
		t.Fatalf("expected campaign content, got %+v", got)
	}
	empty := []store.ContentItem{}
	if got := store.EffectiveContent(campaign, store.Cycle{Content_override: &empty}); len(got) != 0 {
		t.Fatalf("expected explicit empty override, got %+v", got)
	}
}

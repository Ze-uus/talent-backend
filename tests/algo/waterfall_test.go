package algo_test

import (
	"testing"

	"github.com/Ze-uus/talent-backend/internal/algo"
)

func TestWaterfall_BasicProportional(t *testing.T) {
	in := algo.WaterfallInput{
		Cycle_budget:    100_000,
		Tiers:           []int{5000, 10000, 15000},
		Talents_by_tier: map[int]int{5000: 10, 10000: 5, 15000: 2},
		Config:          algo.DefaultConfig(),
	}
	out, err := algo.RunWaterfall(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total_allocated > in.Cycle_budget {
		t.Fatalf("allocated %.2f exceeds cycle budget %.2f", out.Total_allocated, in.Cycle_budget)
	}
	if len(out.Slots) == 0 {
		t.Fatal("expected slots, got none")
	}
	if out.Remainder < 0 {
		t.Fatalf("remainder should be non-negative, got %.2f", out.Remainder)
	}
}

func TestWaterfall_ConfigCapFiltering(t *testing.T) {
	cfg := algo.DefaultConfig()
	cfg.Max_tier_cap = 20000

	in := algo.WaterfallInput{
		Cycle_budget:    200_000,
		Tiers:           []int{5000, 10000, 20000, 30000},
		Talents_by_tier: map[int]int{5000: 4, 10000: 4, 20000: 2, 30000: 2},
		Config:          cfg,
	}
	out, err := algo.RunWaterfall(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range out.Slots {
		if s.Tier > 20000 {
			t.Errorf("tier %d should be excluded by max_tier_cap 20000", s.Tier)
		}
	}
}

func TestWaterfall_BudgetSharePctEnforcement(t *testing.T) {
	cfg := algo.DefaultConfig()
	cfg.Budget_share_pct = 0.10
	cfg.Max_tier_cap = 55000

	in := algo.WaterfallInput{
		Cycle_budget:    100_000,
		Tiers:           []int{5000, 10000, 15000},
		Talents_by_tier: map[int]int{5000: 4, 10000: 4, 15000: 2},
		Config:          cfg,
	}
	out, err := algo.RunWaterfall(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 10% of 100k = 10k, so 15k tier should be excluded
	for _, s := range out.Slots {
		if s.Tier > 10000 {
			t.Errorf("tier %d exceeds budget_share_pct limit of 10000", s.Tier)
		}
	}
}

func TestWaterfall_Empty(t *testing.T) {
	out, err := algo.RunWaterfall(algo.WaterfallInput{Config: algo.DefaultConfig()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total_allocated != 0 {
		t.Error("empty input should yield zero allocation")
	}
}

func TestWaterfall_InvalidConfig(t *testing.T) {
	in := algo.WaterfallInput{
		Cycle_budget: 100_000,
		Config:       algo.CycleConfig{}, // all zeros — invalid
	}
	_, err := algo.RunWaterfall(in)
	if err == nil {
		t.Error("expected error for invalid config")
	}
}

func TestWaterfall_NegativeBudget(t *testing.T) {
	in := algo.WaterfallInput{
		Cycle_budget: -50_000,
		Config:       algo.DefaultConfig(),
	}
	_, err := algo.RunWaterfall(in)
	if err == nil {
		t.Error("expected error for negative budget")
	}
}

func TestWaterfall_MinTierFiltering(t *testing.T) {
	cfg := algo.DefaultConfig()
	cfg.Min_tier = 10000

	in := algo.WaterfallInput{
		Cycle_budget:    200_000,
		Tiers:           []int{5000, 10000, 15000},
		Talents_by_tier: map[int]int{5000: 10, 10000: 5, 15000: 3},
		Config:          cfg,
	}
	out, err := algo.RunWaterfall(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, s := range out.Slots {
		if s.Tier < 10000 {
			t.Errorf("tier %d should be excluded by min_tier 10000", s.Tier)
		}
	}
}

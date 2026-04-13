package algo_test

import (
	"testing"

	"github.com/Ze-uus/talent-backend/internal/algo"
)

func TestDefaultConfig_IsValid(t *testing.T) {
	cfg := algo.DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
}

func TestDefaultConfig_Values(t *testing.T) {
	cfg := algo.DefaultConfig()
	if cfg.Max_tier_cap != 55000 {
		t.Errorf("expected max_tier_cap 55000, got %d", cfg.Max_tier_cap)
	}
	if cfg.Min_tier != 5000 {
		t.Errorf("expected min_tier 5000, got %d", cfg.Min_tier)
	}
	if cfg.Tier_increment != 5000 {
		t.Errorf("expected tier_increment 5000, got %d", cfg.Tier_increment)
	}
	if cfg.Budget_share_pct != 0.25 {
		t.Errorf("expected budget_share_pct 0.25, got %f", cfg.Budget_share_pct)
	}
	if cfg.Score_cap_pct != 1.5 {
		t.Errorf("expected score_cap_pct 1.5, got %f", cfg.Score_cap_pct)
	}
}

func TestValidate_ZeroIncrement(t *testing.T) {
	cfg := algo.DefaultConfig()
	cfg.Tier_increment = 0
	if err := cfg.Validate(); err == nil {
		t.Error("zero tier_increment should fail validation")
	}
}

func TestValidate_NegativeCap(t *testing.T) {
	cfg := algo.DefaultConfig()
	cfg.Max_tier_cap = -1
	if err := cfg.Validate(); err == nil {
		t.Error("negative max_tier_cap should fail validation")
	}
}

func TestValidate_MinExceedsMax(t *testing.T) {
	cfg := algo.DefaultConfig()
	cfg.Min_tier = 60000
	cfg.Max_tier_cap = 50000
	if err := cfg.Validate(); err == nil {
		t.Error("min_tier > max_tier_cap should fail validation")
	}
}

func TestValidate_BudgetSharePctOutOfBounds(t *testing.T) {
	cfg := algo.DefaultConfig()

	cfg.Budget_share_pct = 0
	if err := cfg.Validate(); err == nil {
		t.Error("budget_share_pct 0 should fail")
	}

	cfg.Budget_share_pct = 1.5
	if err := cfg.Validate(); err == nil {
		t.Error("budget_share_pct > 1 should fail")
	}

	cfg.Budget_share_pct = -0.1
	if err := cfg.Validate(); err == nil {
		t.Error("negative budget_share_pct should fail")
	}
}

func TestValidate_ScoreCapPctOutOfBounds(t *testing.T) {
	cfg := algo.DefaultConfig()

	cfg.Score_cap_pct = 0
	if err := cfg.Validate(); err == nil {
		t.Error("score_cap_pct 0 should fail")
	}

	cfg.Score_cap_pct = 11
	if err := cfg.Validate(); err == nil {
		t.Error("score_cap_pct > 10 should fail")
	}
}

func TestValidate_NonMultipleMin(t *testing.T) {
	cfg := algo.DefaultConfig()
	cfg.Min_tier = 7000
	if err := cfg.Validate(); err == nil {
		t.Error("min_tier not a multiple of tier_increment should fail")
	}
}

func TestValidate_NonMultipleMax(t *testing.T) {
	cfg := algo.DefaultConfig()
	cfg.Max_tier_cap = 57000
	if err := cfg.Validate(); err == nil {
		t.Error("max_tier_cap not a multiple of tier_increment should fail")
	}
}

func TestValidate_CustomConfig(t *testing.T) {
	cfg := algo.CycleConfig{
		Max_tier_cap:     100000,
		Min_tier:         10000,
		Tier_increment:   10000,
		Budget_share_pct: 0.30,
		Score_cap_pct:    2.0,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("custom config should be valid: %v", err)
	}
}

func TestValidTiers_Default(t *testing.T) {
	cfg := algo.DefaultConfig()
	tiers := algo.ValidTiers(cfg)
	expected := []int{5000, 10000, 15000, 20000, 25000, 30000, 35000, 40000, 45000, 50000, 55000}
	if len(tiers) != len(expected) {
		t.Fatalf("expected %d tiers, got %d", len(expected), len(tiers))
	}
	for i, v := range expected {
		if tiers[i] != v {
			t.Errorf("tier[%d]: expected %d, got %d", i, v, tiers[i])
		}
	}
}

func TestValidTiers_CustomIncrement(t *testing.T) {
	cfg := algo.CycleConfig{
		Max_tier_cap:     30000,
		Min_tier:         10000,
		Tier_increment:   10000,
		Budget_share_pct: 0.25,
		Score_cap_pct:    1.5,
	}
	tiers := algo.ValidTiers(cfg)
	expected := []int{10000, 20000, 30000}
	if len(tiers) != len(expected) {
		t.Fatalf("expected %d tiers, got %d", len(expected), len(tiers))
	}
	for i, v := range expected {
		if tiers[i] != v {
			t.Errorf("tier[%d]: expected %d, got %d", i, v, tiers[i])
		}
	}
}

func TestValidTiers_InvalidConfig(t *testing.T) {
	cfg := algo.CycleConfig{Tier_increment: 0}
	tiers := algo.ValidTiers(cfg)
	if tiers != nil {
		t.Error("invalid config should return nil")
	}
}

package algo_test

import (
	"testing"

	"github.com/Ze-uus/talent-backend/internal/algo"
)

func makeHistory(count int, utilization, avg_perf, waste_ratio, unassign_rate float64) []algo.CycleHistory {
	cfg := algo.DefaultConfig()
	history := make([]algo.CycleHistory, count)
	budget := 200_000.0
	for i := range history {
		allocated := budget * utilization
		history[i] = algo.CycleHistory{
			Cycle_number:     i + 1,
			Cycle_budget:     budget,
			Config:           cfg,
			Total_allocated:  allocated,
			Remainder:        budget * waste_ratio,
			Utilization:      utilization,
			Avg_performance:  avg_perf,
			Talent_count:     20,
			Unassigned_count: int(20.0 * unassign_rate),
			Avg_payout_ratio: 0.7,
		}
	}
	return history
}

func TestSuggestConfig_InsufficientData(t *testing.T) {
	history := makeHistory(5, 0.80, 0.90, 0.10, 0.05)
	_, err := algo.SuggestConfig(history, 200_000)
	if err == nil {
		t.Error("expected error for fewer than 7 cycles")
	}
}

func TestSuggestConfig_StableHistory(t *testing.T) {
	history := makeHistory(10, 0.80, 0.90, 0.10, 0.05)
	result, err := algo.SuggestConfig(history, 200_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	maintain_count := 0
	for _, f := range result.Factors {
		if f.Direction == "maintain" {
			maintain_count++
		}
	}
	if maintain_count < 2 {
		t.Errorf("expected most factors to be 'maintain' for stable history, got %d/4", maintain_count)
	}
	if result.Confidence <= 0 || result.Confidence > 1 {
		t.Errorf("confidence should be in (0, 1], got %f", result.Confidence)
	}
}

func TestSuggestConfig_LowUtilization(t *testing.T) {
	history := makeHistory(8, 0.45, 0.80, 0.40, 0.10)
	result, err := algo.SuggestConfig(history, 200_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range result.Factors {
		if f.Metric == "utilization_trend" {
			if f.Direction != "decrease" {
				t.Errorf("expected utilization_trend direction 'decrease', got '%s'", f.Direction)
			}
			if f.Reasoning == "" {
				t.Error("reasoning should not be empty")
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("expected utilization_trend factor")
	}

	if result.Suggested_config.Max_tier_cap >= algo.DefaultConfig().Max_tier_cap {
		t.Error("low utilization should suggest a lower max_tier_cap")
	}
}

func TestSuggestConfig_HighUnassignment(t *testing.T) {
	cfg := algo.CycleConfig{
		Max_tier_cap:     55000,
		Min_tier:         10000,
		Tier_increment:   5000,
		Budget_share_pct: 0.25,
		Score_cap_pct:    1.5,
	}
	history := make([]algo.CycleHistory, 8)
	for i := range history {
		history[i] = algo.CycleHistory{
			Cycle_number:     i + 1,
			Cycle_budget:     200_000,
			Config:           cfg,
			Total_allocated:  140_000,
			Remainder:        30_000,
			Utilization:      0.70,
			Avg_performance:  0.80,
			Talent_count:     20,
			Unassigned_count: 7,
			Avg_payout_ratio: 0.7,
		}
	}
	result, err := algo.SuggestConfig(history, 200_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range result.Factors {
		if f.Metric == "unassignment_rate" {
			if f.Direction != "decrease" {
				t.Errorf("expected unassignment_rate direction 'decrease', got '%s'", f.Direction)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("expected unassignment_rate factor")
	}

	if result.Suggested_config.Min_tier >= 10000 {
		t.Error("high unassignment should suggest a lower min_tier")
	}
}

func TestSuggestConfig_Exactly7Cycles(t *testing.T) {
	history := makeHistory(7, 0.80, 0.90, 0.10, 0.05)
	result, err := algo.SuggestConfig(history, 200_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Min_cycles_used != 7 {
		t.Errorf("expected min_cycles_used 7, got %d", result.Min_cycles_used)
	}
	if len(result.Factors) != 4 {
		t.Errorf("expected 4 factors, got %d", len(result.Factors))
	}
}

func TestSuggestConfig_ConfidenceIncreasesWithData(t *testing.T) {
	history7 := makeHistory(7, 0.80, 0.90, 0.10, 0.05)
	result7, err := algo.SuggestConfig(history7, 200_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	history15 := makeHistory(15, 0.80, 0.90, 0.10, 0.05)
	result15, err := algo.SuggestConfig(history15, 200_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result15.Confidence <= result7.Confidence {
		t.Errorf("more data should yield higher confidence: 7-cycle=%.2f, 15-cycle=%.2f",
			result7.Confidence, result15.Confidence)
	}
}

func TestSuggestConfig_HighWaste(t *testing.T) {
	history := makeHistory(8, 0.65, 0.80, 0.40, 0.05)
	result, err := algo.SuggestConfig(history, 200_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range result.Factors {
		if f.Metric == "waste_ratio" {
			if f.Direction != "decrease" {
				t.Errorf("expected waste_ratio direction 'decrease', got '%s'", f.Direction)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("expected waste_ratio factor")
	}

	if result.Suggested_config.Budget_share_pct >= algo.DefaultConfig().Budget_share_pct {
		t.Error("high waste should suggest a lower budget_share_pct")
	}
}

func TestSuggestConfig_NearScoreCap(t *testing.T) {
	history := makeHistory(8, 0.80, 1.40, 0.10, 0.05)
	result, err := algo.SuggestConfig(history, 200_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, f := range result.Factors {
		if f.Metric == "performance_distribution" {
			if f.Direction != "increase" {
				t.Errorf("expected performance_distribution direction 'increase', got '%s'", f.Direction)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("expected performance_distribution factor")
	}

	if result.Suggested_config.Score_cap_pct <= algo.DefaultConfig().Score_cap_pct {
		t.Error("near-cap performance should suggest a higher score_cap_pct")
	}
}

func TestSuggestConfig_OutputConfigValid(t *testing.T) {
	history := makeHistory(10, 0.45, 0.40, 0.40, 0.35)
	result, err := algo.SuggestConfig(history, 200_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := result.Suggested_config.Validate(); err != nil {
		t.Errorf("suggested config should be valid: %v", err)
	}
}

func TestSuggestionReady(t *testing.T) {
	if algo.SuggestionReady(6) {
		t.Error("6 cycles should not be ready")
	}
	if !algo.SuggestionReady(7) {
		t.Error("7 cycles should be ready")
	}
	if !algo.SuggestionReady(20) {
		t.Error("20 cycles should be ready")
	}
}

func TestSuggestConfig_ZeroBudget(t *testing.T) {
	history := makeHistory(8, 0.80, 0.90, 0.10, 0.05)
	_, err := algo.SuggestConfig(history, 0)
	if err == nil {
		t.Error("expected error for zero next_budget")
	}
}

func TestSuggestConfig_FactorsHaveReasoning(t *testing.T) {
	history := makeHistory(8, 0.50, 0.80, 0.30, 0.20)
	result, err := algo.SuggestConfig(history, 200_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, f := range result.Factors {
		if f.Reasoning == "" {
			t.Errorf("factor %s has empty reasoning", f.Metric)
		}
		if f.Metric == "" {
			t.Error("factor has empty metric name")
		}
		if f.Direction != "increase" && f.Direction != "decrease" && f.Direction != "maintain" {
			t.Errorf("factor %s has invalid direction: %s", f.Metric, f.Direction)
		}
	}
}

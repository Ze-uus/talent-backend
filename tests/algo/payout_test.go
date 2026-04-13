package algo_test

import (
	"testing"

	"github.com/Ze-uus/talent-backend/internal/algo"
)

func TestPayout_FullScore(t *testing.T) {
	r := algo.CycleRecord{
		Cycle_number:      1,
		Total_conversions: 50, KPI_target: 50,
		Days_active: 5, Rate_per_day: 1000,
		Slot_budget: 5000, PDC1_baseline: 10,
		Score_cap_pct: 1.5,
	}
	p, err := algo.ComputePayout(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Performance_score != 1.0 {
		t.Errorf("expected 1.0, got %.2f", p.Performance_score)
	}
	if p.Final_payout != 5000 {
		t.Errorf("expected 5000, got %.2f", p.Final_payout)
	}
	if p.Flagged {
		t.Error("should not be flagged")
	}
}

func TestPayout_FlaggedLowPerformance(t *testing.T) {
	r := algo.CycleRecord{
		Cycle_number:      2,
		Total_conversions: 10, KPI_target: 50,
		Days_active: 5, Rate_per_day: 1000,
		Slot_budget: 5000, STEC_pdc_next: 8,
		Score_cap_pct: 1.5,
	}
	p, err := algo.ComputePayout(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.Flagged {
		t.Error("expected flagged for <50% performance")
	}
	if p.Flag_reason != "performance_below_50_percent" {
		t.Errorf("expected performance_below_50_percent, got %s", p.Flag_reason)
	}
}

func TestPayout_BudgetCapped(t *testing.T) {
	r := algo.CycleRecord{
		Cycle_number:      1,
		Total_conversions: 75, KPI_target: 50,
		Days_active: 5, Rate_per_day: 2000,
		Slot_budget: 5000, PDC1_baseline: 15,
		Score_cap_pct: 1.5,
	}
	p, err := algo.ComputePayout(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Final_payout > 5000 {
		t.Errorf("payout %.2f exceeded slot budget", p.Final_payout)
	}
}

func TestPayout_Cycle2BranchesCorrectly(t *testing.T) {
	r := algo.CycleRecord{
		Cycle_number:      2,
		Total_conversions: 50, KPI_target: 50,
		Days_active: 5, Rate_per_day: 1000,
		Slot_budget: 5000,
		PDC1_baseline: 0,
		STEC_pdc_next: 10,
		Score_cap_pct: 1.5,
	}
	p, err := algo.ComputePayout(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Flagged && p.Flag_reason == "invalid_pdc_reference" {
		t.Error("cycle 2 should use STEC_pdc_next, not PDC1_baseline")
	}
}

func TestPayout_ConfigurableScoreCap(t *testing.T) {
	r := algo.CycleRecord{
		Cycle_number:      1,
		Total_conversions: 200, KPI_target: 50,
		Days_active: 5, Rate_per_day: 1000,
		Slot_budget: 50000, PDC1_baseline: 10,
		Score_cap_pct: 2.0,
	}
	p, err := algo.ComputePayout(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 200/50 = 4.0, capped at 2.0
	if p.Performance_score != 2.0 {
		t.Errorf("expected score capped at 2.0, got %.2f", p.Performance_score)
	}
	// raw = 1000 * 5 * 2.0 = 10000
	if p.Raw_payout != 10000 {
		t.Errorf("expected raw_payout 10000, got %.2f", p.Raw_payout)
	}
}

func TestPayout_ZeroKPITarget(t *testing.T) {
	r := algo.CycleRecord{
		KPI_target:    0,
		Score_cap_pct: 1.5,
	}
	_, err := algo.ComputePayout(r)
	if err == nil {
		t.Error("expected error for zero KPI_target")
	}
}

func TestPayout_NegativeConversions(t *testing.T) {
	r := algo.CycleRecord{
		Total_conversions: -10,
		KPI_target:        50,
		Score_cap_pct:     1.5,
	}
	_, err := algo.ComputePayout(r)
	if err == nil {
		t.Error("expected error for negative total_conversions")
	}
}

func TestPayout_ZeroScoreCapPct(t *testing.T) {
	r := algo.CycleRecord{
		KPI_target:    50,
		Score_cap_pct: 0,
	}
	_, err := algo.ComputePayout(r)
	if err == nil {
		t.Error("expected error for zero score_cap_pct")
	}
}

func TestPayout_InvalidPDCReference(t *testing.T) {
	r := algo.CycleRecord{
		Cycle_number:      1,
		Total_conversions: 50, KPI_target: 50,
		Days_active: 5, Rate_per_day: 1000,
		Slot_budget: 5000, PDC1_baseline: 0,
		Score_cap_pct: 1.5,
	}
	p, err := algo.ComputePayout(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.Flagged || p.Flag_reason != "invalid_pdc_reference" {
		t.Errorf("expected invalid_pdc_reference flag, got flagged=%v reason=%s", p.Flagged, p.Flag_reason)
	}
}

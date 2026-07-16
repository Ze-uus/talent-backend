package algo_test

import (
	"testing"

	"github.com/Ze-uus/talent-backend/internal/algo"
)

func TestSolve_TwoTalentsTwoSlots(t *testing.T) {
	talents := []algo.TalentProfile{
		{ID: "A", PDC: 10, Max_tier: 5000, Cycle_length: 7},
		{ID: "B", PDC: 6, Max_tier: 5000, Cycle_length: 7},
	}
	slots := []algo.BudgetSlot{
		{ID: "s1", Tier_value: 5000, Target_cost: 100, Max_cost: 200},
		{ID: "s2", Tier_value: 5000, Target_cost: 100, Max_cost: 200},
	}
	out, err := algo.Solve(algo.AssignmentInput{
		Talents: talents, Slots: slots, All_tiers: []int{5000, 10000, 15000},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Assignments) != 2 {
		t.Errorf("expected 2 assignments, got %d", len(out.Assignments))
	}
}

func TestSolve_WeakTalentUnassigned(t *testing.T) {
	talents := []algo.TalentProfile{
		{ID: "C", PDC: 2, Max_tier: 10000, Cycle_length: 7},
	}
	slots := []algo.BudgetSlot{
		{ID: "s1", Tier_value: 10000, Target_cost: 100, Max_cost: 200},
	}
	out, err := algo.Solve(algo.AssignmentInput{
		Talents: talents, Slots: slots, All_tiers: []int{5000, 10000},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Unassigned) == 0 {
		t.Error("expected talent C unassigned (too weak for 10k)")
	}
}

func TestComputeCij_PenaltyFormula(t *testing.T) {
	talent := algo.TalentProfile{ID: "A", PDC: 10, Max_tier: 5000, Cycle_length: 7}
	slot := algo.BudgetSlot{ID: "s1", Tier_value: 5000, Target_cost: 100, Max_cost: 200}
	// MS_t=0 → MatchAdjustment(0, 0.20) = 1.0 → no change to cost
	ms := algo.MatchScoreResult{MS_t: 0}
	c := algo.ComputeCij(talent, slot, []int{5000, 10000, 15000}, ms, algo.DefaultAlpha)
	// bounds.Min = ceil(5000/200) = 25, bounds.Max = floor(5000/100) = 50
	// base_days = ceil(25/10) = 3, actual = 3*10 = 30
	// shortfall = (50-30)/50 = 0.4, penalty = floor(0.4/0.10) = 4
	// Cij = (3 + 4) × 1.0 = 7
	if c != 7 {
		t.Errorf("expected Cij=7 (3 base + 4 penalty), got %.1f", c)
	}
}

func TestSolve_EmptyInputs(t *testing.T) {
	out, err := algo.Solve(algo.AssignmentInput{})
	if err != nil {
		t.Fatalf("both empty should not error: %v", err)
	}
	if len(out.Assignments) != 0 {
		t.Error("expected no assignments for empty input")
	}
}

func TestSolve_NoSlots(t *testing.T) {
	_, err := algo.Solve(algo.AssignmentInput{
		Talents:   []algo.TalentProfile{{ID: "A", PDC: 10, Max_tier: 5000, Cycle_length: 7}},
		All_tiers: []int{5000},
	})
	if err == nil {
		t.Error("expected error for talents with no slots")
	}
}

func TestSolve_NoTiers(t *testing.T) {
	_, err := algo.Solve(algo.AssignmentInput{
		Talents: []algo.TalentProfile{{ID: "A", PDC: 10, Max_tier: 5000, Cycle_length: 7}},
		Slots:   []algo.BudgetSlot{{ID: "s1", Tier_value: 5000, Target_cost: 100, Max_cost: 200}},
	})
	if err == nil {
		t.Error("expected error for empty all_tiers")
	}
}

func TestSolve_MoreTalentsThanSlots(t *testing.T) {
	talents := []algo.TalentProfile{
		{ID: "A", PDC: 10, Max_tier: 5000, Cycle_length: 7},
		{ID: "B", PDC: 8, Max_tier: 5000, Cycle_length: 7},
		{ID: "C", PDC: 6, Max_tier: 5000, Cycle_length: 7},
	}
	slots := []algo.BudgetSlot{
		{ID: "s1", Tier_value: 5000, Target_cost: 100, Max_cost: 200},
	}
	out, err := algo.Solve(algo.AssignmentInput{
		Talents: talents, Slots: slots, All_tiers: []int{5000, 10000},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Assignments) != 1 {
		t.Errorf("expected 1 assignment, got %d", len(out.Assignments))
	}
	if len(out.Unassigned) != 2 {
		t.Errorf("expected 2 unassigned, got %d", len(out.Unassigned))
	}
}

func TestComputeCij_IneligibleReturnsForbidden(t *testing.T) {
	talent := algo.TalentProfile{ID: "weak", PDC: 1, Max_tier: 5000, Cycle_length: 7}
	slot := algo.BudgetSlot{ID: "s1", Tier_value: 50000, Target_cost: 100, Max_cost: 200}
	ms := algo.MatchScoreResult{MS_t: 0}
	c := algo.ComputeCij(talent, slot, []int{5000, 50000}, ms, algo.DefaultAlpha)
	if c != algo.Forbidden {
		t.Errorf("expected Forbidden, got %.1f", c)
	}
}

func TestComputeCij_MatchAdjustmentApplied(t *testing.T) {
	talent := algo.TalentProfile{ID: "A", PDC: 10, Max_tier: 5000, Cycle_length: 7}
	slot := algo.BudgetSlot{ID: "s1", Tier_value: 5000, Target_cost: 100, Max_cost: 200}
	// base cost without match = 7.0 (verified in TestComputeCij_PenaltyFormula)
	// MS_t=1.0, alpha=0.20 → adjustment=0.80 → Cij = 7 × 0.80 = 5.6
	ms := algo.MatchScoreResult{MS_t: 1.0}
	c := algo.ComputeCij(talent, slot, []int{5000, 10000, 15000}, ms, 0.20)
	if c >= 7.0 {
		t.Errorf("expected match-adjusted cost < 7.0, got %.2f", c)
	}
	if c < 5.5 || c > 5.7 {
		t.Errorf("expected Cij ≈ 5.6 (7 × 0.80), got %.2f", c)
	}
}

func TestSolve_WithMatchScores(t *testing.T) {
	// Two talents with identical PDC — one with perfect match, one with zero match.
	// High-match talent should get a lower cost in the assignment.
	talents := []algo.TalentProfile{
		{ID: "high_match", PDC: 10, Max_tier: 5000, Cycle_length: 7},
		{ID: "low_match", PDC: 10, Max_tier: 5000, Cycle_length: 7},
	}
	slots := []algo.BudgetSlot{
		{ID: "s1", Tier_value: 5000, Target_cost: 100, Max_cost: 200},
		{ID: "s2", Tier_value: 5000, Target_cost: 100, Max_cost: 200},
	}
	match_scores := map[string]algo.MatchScoreResult{
		"high_match": {MS_t: 1.0},
		"low_match":  {MS_t: 0.0},
	}
	out, err := algo.Solve(algo.AssignmentInput{
		Talents:     talents,
		Slots:       slots,
		All_tiers:   []int{5000, 10000, 15000},
		MatchScores: match_scores,
		Alpha:       algo.DefaultAlpha,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(out.Assignments))
	}
	// Verify total cost is lower than without match scores (2 × 7.0 = 14.0)
	if out.Total_cost >= 14.0 {
		t.Errorf("expected total_cost < 14.0 with match scores, got %.2f", out.Total_cost)
	}
}

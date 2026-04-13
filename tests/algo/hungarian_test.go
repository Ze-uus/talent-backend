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
	c := algo.ComputeCij(talent, slot, []int{5000, 10000, 15000})
	// bounds.Min = ceil(5000/200) = 25, bounds.Max = floor(5000/100) = 50
	// base_days = ceil(25/10) = 3, actual = 3*10 = 30
	// shortfall = (50-30)/50 = 0.4, penalty = floor(0.4/0.10) = 4
	// Cij = 3 + 4 = 7
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
	c := algo.ComputeCij(talent, slot, []int{5000, 50000})
	if c != algo.Forbidden {
		t.Errorf("expected Forbidden, got %.1f", c)
	}
}

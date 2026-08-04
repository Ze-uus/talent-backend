package algo_test

import (
	"math"
	"testing"

	"github.com/Ze-uus/talent-backend/internal/algo"
)

func TestQualify_Passes(t *testing.T) {
	talent := algo.TalentProfile{ID: "t1", PDC: 10, Max_tier: 5000, Cycle_length: 7}
	slot := algo.BudgetSlot{ID: "s1", Tier_value: 10000, Target_cost: 100, Max_cost: 200}
	r, err := algo.QualifyTalentForSlot(talent, slot, []int{5000, 10000, 15000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.Qualified {
		t.Errorf("expected qualified, got: %s", r.Reason)
	}
}

func TestQualify_DownwardMobilityBlocked(t *testing.T) {
	talent := algo.TalentProfile{ID: "t2", PDC: 15, Max_tier: 15000, Cycle_length: 7}
	slot := algo.BudgetSlot{ID: "s1", Tier_value: 5000, Target_cost: 100, Max_cost: 200}
	r, err := algo.QualifyTalentForSlot(talent, slot, []int{5000, 10000, 15000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Qualified || r.Reason != "mobility_blocked" {
		t.Errorf("expected mobility_blocked, got %v / %s", r.Qualified, r.Reason)
	}
}

func TestQualify_TooWeak(t *testing.T) {
	talent := algo.TalentProfile{ID: "t3", PDC: 2, Max_tier: 10000, Cycle_length: 7}
	slot := algo.BudgetSlot{ID: "s1", Tier_value: 10000, Target_cost: 100, Max_cost: 200}
	r, err := algo.QualifyTalentForSlot(talent, slot, []int{5000, 10000, 15000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Qualified || r.Reason != "too_weak" {
		t.Errorf("expected too_weak, got %v / %s", r.Qualified, r.Reason)
	}
}

func TestDemandTier_Formula(t *testing.T) {
	tier := algo.DemandTier(10, 7, 200, 5000)
	if tier != 10000 {
		t.Errorf("expected 10000, got %d", tier)
	}
}

func TestDemandTier_CustomIncrement(t *testing.T) {
	tier := algo.DemandTier(10, 7, 200, 10000)
	if tier != 10000 {
		t.Errorf("expected 10000, got %d", tier)
	}

	tier2 := algo.DemandTier(10, 7, 200, 3000)
	// raw = 14000, floor(14000/3000) * 3000 = 4*3000 = 12000
	if tier2 != 12000 {
		t.Errorf("expected 12000, got %d", tier2)
	}
}

func TestDemandTier_ZeroInputs(t *testing.T) {
	if algo.DemandTier(0, 7, 200, 5000) != 0 {
		t.Error("zero PDC should return 0")
	}
	if algo.DemandTier(10, 0, 200, 5000) != 0 {
		t.Error("zero cycle_length should return 0")
	}
	if algo.DemandTier(10, 7, 0, 5000) != 0 {
		t.Error("zero max_cpa should return 0")
	}
	if algo.DemandTier(10, 7, 200, 0) != 0 {
		t.Error("zero tier_increment should return 0")
	}
}

func TestBoundedDemandTier_ClampsToTalentAndPlatformLimits(t *testing.T) {
	cfg := algo.DefaultConfig()
	tier := algo.BoundedDemandTier(15, 7, 300000, 300000, 5000, cfg)
	if tier != 5000 {
		t.Fatalf("expected tier 5000, got %d", tier)
	}

	tier = algo.BoundedDemandTier(15, 7, 300000, 300000, 50000, cfg)
	if tier != 50000 {
		t.Fatalf("expected tier 50000, got %d", tier)
	}
}

func TestBoundedDemandTier_RejectsBudgetBelowMinimum(t *testing.T) {
	cfg := algo.DefaultConfig()
	if tier := algo.BoundedDemandTier(2, 7, 100, 10000, 5000, cfg); tier != 0 {
		t.Fatalf("expected no tier, got %d", tier)
	}
}

func TestQualify_NaNPDC(t *testing.T) {
	talent := algo.TalentProfile{ID: "bad", PDC: math.NaN(), Max_tier: 5000, Cycle_length: 7}
	slot := algo.BudgetSlot{ID: "s1", Tier_value: 5000, Target_cost: 100, Max_cost: 200}
	_, err := algo.QualifyTalentForSlot(talent, slot, []int{5000})
	if err == nil {
		t.Error("expected error for NaN PDC")
	}
}

func TestQualify_EmptyTiers(t *testing.T) {
	talent := algo.TalentProfile{ID: "t1", PDC: 10, Max_tier: 5000, Cycle_length: 7}
	slot := algo.BudgetSlot{ID: "s1", Tier_value: 5000, Target_cost: 100, Max_cost: 200}
	_, err := algo.QualifyTalentForSlot(talent, slot, nil)
	if err == nil {
		t.Error("expected error for empty tiers")
	}
}

func TestQualify_InvalidSlot(t *testing.T) {
	talent := algo.TalentProfile{ID: "t1", PDC: 10, Max_tier: 5000, Cycle_length: 7}
	slot := algo.BudgetSlot{ID: "bad", Tier_value: 5000, Target_cost: 0, Max_cost: 200}
	_, err := algo.QualifyTalentForSlot(talent, slot, []int{5000})
	if err == nil {
		t.Error("expected error for zero target_cost")
	}
}

func TestTierBoundsFor_Valid(t *testing.T) {
	slot := algo.BudgetSlot{Tier_value: 10000, Target_cost: 100, Max_cost: 200}
	bounds, err := algo.TierBoundsFor(slot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// min = ceil(10000/200) = 50, max = floor(10000/100) = 100
	if bounds.Min != 50 {
		t.Errorf("expected min 50, got %d", bounds.Min)
	}
	if bounds.Max != 100 {
		t.Errorf("expected max 100, got %d", bounds.Max)
	}
}

func TestTierBoundsFor_InvalidCosts(t *testing.T) {
	_, err := algo.TierBoundsFor(algo.BudgetSlot{Tier_value: 5000, Target_cost: -1, Max_cost: 200})
	if err == nil {
		t.Error("expected error for negative target_cost")
	}
	_, err = algo.TierBoundsFor(algo.BudgetSlot{Tier_value: 5000, Target_cost: 100, Max_cost: 0})
	if err == nil {
		t.Error("expected error for zero max_cost")
	}
}

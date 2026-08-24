package allocation_test

import (
	"testing"

	"github.com/Ze-uus/talent-backend/internal/allocation"
	"github.com/Ze-uus/talent-backend/internal/store"
)

func TestBuildBudgetSlots_MapsRealisticCPAIntoValidTier(t *testing.T) {
	cycle := store.Cycle{ID: "cycle-1", Cycle_budget: 300000}
	campaign := store.Campaign{
		Cycle_length: 7,
		Target_cpa:   150000,
		Max_cpa:      300000,
	}
	talents := []store.Talent{{
		ID:       "talent-1",
		Category: store.Category_community,
		Status:   store.Status_active,
		Max_tier: 0,
	}}

	slots, err := allocation.BuildBudgetSlots(cycle, campaign, talents)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) == 0 {
		t.Fatal("expected budget slots")
	}

	seen := make(map[string]bool, len(slots))
	for _, slot := range slots {
		if slot.ID == "" {
			t.Fatal("slot ID must not be empty")
		}
		if seen[slot.ID] {
			t.Fatalf("duplicate slot ID %q", slot.ID)
		}
		seen[slot.ID] = true
		if slot.Cycle_id != cycle.ID || slot.Tier_value != 300000 {
			t.Fatalf("unexpected slot: %+v", slot)
		}
	}
}

func TestBuildBudgetSlots_RejectsManualCapBelowTargetCPA(t *testing.T) {
	slots, err := allocation.BuildBudgetSlots(
		store.Cycle{ID: "cycle-1", Cycle_budget: 300000},
		store.Campaign{Cycle_length: 7, Target_cpa: 150000, Max_cpa: 300000},
		[]store.Talent{{
			ID: "talent-1", Category: store.Category_community,
			Status: store.Status_active, Max_tier: 5000,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 0 {
		t.Fatalf("expected incompatible talent cap to produce no slot, got %d", len(slots))
	}
}

func TestBuildBudgetSlots_NoActiveTalents(t *testing.T) {
	slots, err := allocation.BuildBudgetSlots(
		store.Cycle{ID: "cycle-1", Cycle_budget: 300000},
		store.Campaign{Cycle_length: 7, Max_cpa: 300000},
		[]store.Talent{{ID: "talent-1", Status: store.Status_pending}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(slots) != 0 {
		t.Fatalf("expected no slots, got %d", len(slots))
	}
}

func TestBuildBudgetSlots_RejectsTierOutsideDatabaseRange(t *testing.T) {
	cycle := store.Cycle{ID: "cycle-1", Cycle_budget: 3_000_000_000}
	campaign := store.Campaign{
		Cycle_length: 7,
		Target_cpa:   1_000_000_000,
		Max_cpa:      1_000_000_000,
	}
	talents := []store.Talent{{
		ID: "talent-1", Category: store.Category_community, Status: store.Status_active,
	}}
	_, err := allocation.BuildBudgetSlots(cycle, campaign, talents)
	if err == nil || err.Error() != "tier_value_exceeds_supported_range" {
		t.Fatalf("expected tier range error, got %v", err)
	}
}

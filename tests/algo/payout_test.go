package algo_test

import (
	"math"
	"testing"

	"github.com/Ze-uus/talent-backend/internal/algo"
)

// ─── MoneyRound ───────────────────────────────────────────────────────────────

func TestMoneyRound_KoboPrecision(t *testing.T) {
	// 1/3 NGN must round to nearest kobo, not floor
	result := algo.MoneyRound(1.0 / 3.0)
	if math.Abs(result-0.33) > 0.005 {
		t.Errorf("expected ~0.33, got %.6f", result)
	}
}

// ─── Traffic Pipeline ─────────────────────────────────────────────────────────

func TestPayout_Traffic_FullPerformance(t *testing.T) {
	talent := algo.TalentPayoutInput{
		Talent_id:        "t1",
		Pipeline:         algo.Pipeline_traffic,
		Allocated_budget: 25000,
		Destinations: []algo.DestinationParams{
			{Destination_id: "d1", Anchor_cost: 500, Cost_variance: 150, Conversions: 80},
		},
		Commission_rate: 0.30,
		Status:          "active",
	}
	// Contribution = 80 × (500+150) = 52000 > 25000 → capped at C_t
	// cost_per_conversion = 25000/80 = 312.5 ≤ 650 cap → no excess
	pool := algo.CyclePoolInput{Available_pool: 100_000}
	out := algo.ComputeCyclePayout([]algo.TalentPayoutInput{talent}, pool)
	if len(out.Results) != 1 {
		t.Fatal("expected 1 result")
	}
	r := out.Results[0]
	if r.Gross_base != 25000 {
		t.Errorf("expected gross_base=25000, got %.2f", r.Gross_base)
	}
	if r.Cap_exceeded {
		t.Error("cap should not be exceeded")
	}
	expected_e_net := algo.MoneyRound(25000 * 0.70)
	if math.Abs(r.E_net-expected_e_net) > 0.01 {
		t.Errorf("expected e_net=%.2f, got %.2f", expected_e_net, r.E_net)
	}
}

func TestPayout_Traffic_CapExceeded(t *testing.T) {
	talent := algo.TalentPayoutInput{
		Talent_id:        "t2",
		Pipeline:         algo.Pipeline_traffic,
		Allocated_budget: 50000,
		Destinations: []algo.DestinationParams{
			{Destination_id: "d1", Anchor_cost: 500, Cost_variance: 100, Conversions: 40},
		},
		Commission_rate: 0.30,
		Status:          "active",
	}
	// cost_per_conversion = 50000/40 = 1250 > cap 600 → forfeits excess
	pool := algo.CyclePoolInput{Available_pool: 100_000}
	out := algo.ComputeCyclePayout([]algo.TalentPayoutInput{talent}, pool)
	r := out.Results[0]
	if !r.Cap_exceeded {
		t.Error("expected cap exceeded")
	}
	expected_capped := algo.MoneyRound(40 * 600.0)
	if math.Abs(r.Gross_base-expected_capped) > 0.01 {
		t.Errorf("expected capped gross=%.2f, got %.2f", expected_capped, r.Gross_base)
	}
}

// ─── Lead Pipeline ────────────────────────────────────────────────────────────

func TestPayout_Lead_FullPerformance(t *testing.T) {
	talent := algo.TalentPayoutInput{
		Talent_id:        "t3",
		Pipeline:         algo.Pipeline_lead,
		Allocated_budget: 25000,
		Valid_leads:      80,
		Anchor_lead_cost: 250,
		Cost_variance_l:  75,
		Commission_rate:  0.30,
		Status:           "active",
	}
	// cost_per_lead = 25000/80 = 312.5 ≤ cap 325 → full budget
	pool := algo.CyclePoolInput{Available_pool: 100_000}
	out := algo.ComputeCyclePayout([]algo.TalentPayoutInput{talent}, pool)
	r := out.Results[0]
	if r.Gross_base != 25000 {
		t.Errorf("expected gross_base=25000, got %.2f", r.Gross_base)
	}
	if r.Cap_exceeded {
		t.Error("cap should not be exceeded")
	}
}

func TestPayout_Lead_WithKPBFromAllocated(t *testing.T) {
	talent := algo.TalentPayoutInput{
		Talent_id:        "t4",
		Pipeline:         algo.Pipeline_lead,
		Allocated_budget: 25000,
		Valid_leads:      50, // cost_per_lead=500 > cap 400 → gross_base=20000; 20000+5000=25000=C_t → KPB not truncated
		Anchor_lead_cost: 300,
		Cost_variance_l:  100,
		KPB_bonuses: []algo.KPBBonusParams{
			{Option_id: "k1", Bonus_amount: 500, Trigger_count: 10},
		},
		KPB_from_pool:   false, // deducted from C_t
		Commission_rate: 0.30,
		Status:          "active",
	}
	pool := algo.CyclePoolInput{Available_pool: 100_000}
	out := algo.ComputeCyclePayout([]algo.TalentPayoutInput{talent}, pool)
	r := out.Results[0]
	if r.KPB_total != 5000 {
		t.Errorf("expected KPB_total=5000, got %.2f", r.KPB_total)
	}
	if math.Abs(r.Gross_total-(r.Gross_base+r.KPB_total)) > 0.01 {
		t.Error("gross_total should equal gross_base + kpb_total")
	}
}

// ─── Pool scaling ─────────────────────────────────────────────────────────────

func TestPayout_PoolScaling_Applied(t *testing.T) {
	make_talent := func(id string) algo.TalentPayoutInput {
		return algo.TalentPayoutInput{
			Talent_id:        id,
			Pipeline:         algo.Pipeline_traffic,
			Allocated_budget: 30000,
			Destinations: []algo.DestinationParams{
				{Anchor_cost: 500, Cost_variance: 100, Conversions: 50},
			},
			Commission_rate: 0.30,
			Status:          "active",
		}
	}
	talents := []algo.TalentPayoutInput{make_talent("t1"), make_talent("t2")}
	pool := algo.CyclePoolInput{Available_pool: 30_000} // less than combined demand
	out := algo.ComputeCyclePayout(talents, pool)
	if out.Scale_factor >= 1.0 {
		t.Error("expected scale_factor < 1.0")
	}
	if out.Total_paid > pool.Available_pool+0.01 {
		t.Errorf("total_paid %.2f exceeds pool %.2f", out.Total_paid, pool.Available_pool)
	}
}

// ─── Forfeit ──────────────────────────────────────────────────────────────────

func TestPayout_Forfeit_ReturnsToPool(t *testing.T) {
	talent := algo.TalentPayoutInput{
		Talent_id:        "t5",
		Pipeline:         algo.Pipeline_traffic,
		Allocated_budget: 15000,
		Commission_rate:  0.30,
		Status:           "removed_forfeit",
	}
	pool := algo.CyclePoolInput{Available_pool: 50_000}
	out := algo.ComputeCyclePayout([]algo.TalentPayoutInput{talent}, pool)
	if out.Results[0].Final_payout != 0 {
		t.Error("forfeited talent should have zero payout")
	}
	if out.Forfeits_returned != 15000 {
		t.Errorf("expected forfeits_returned=15000, got %.2f", out.Forfeits_returned)
	}
}

// ─── Rollover ─────────────────────────────────────────────────────────────────

func TestPayout_Rollover_Computed(t *testing.T) {
	talent := algo.TalentPayoutInput{
		Talent_id:        "t6",
		Pipeline:         algo.Pipeline_traffic,
		Allocated_budget: 10000,
		Destinations: []algo.DestinationParams{
			{Anchor_cost: 200, Cost_variance: 50, Conversions: 20},
		},
		Commission_rate: 0.30,
		Status:          "active",
	}
	pool := algo.CyclePoolInput{Available_pool: 100_000}
	out := algo.ComputeCyclePayout([]algo.TalentPayoutInput{talent}, pool)
	expected_rollover := algo.MoneyRound(out.Unspent + out.Forfeits_returned)
	if math.Abs(out.Unallocated_next-expected_rollover) > 0.01 {
		t.Errorf("rollover mismatch: expected %.2f, got %.2f", expected_rollover, out.Unallocated_next)
	}
}

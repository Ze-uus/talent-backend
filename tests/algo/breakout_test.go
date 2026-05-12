package algo_test

import (
	"testing"

	"github.com/Ze-uus/talent-backend/internal/algo"
)

func breakoutInput() algo.BreakoutInput {
	return algo.BreakoutInput{
		Talent_id:         "t1",
		PDC_allocated:     10,
		Total_conversions: 90, // 15/day over 6 days — 150% of PDC_allocated
		Cycle_days:        6,
		Starting_tier:     5000,
		All_tiers:         []int{5000, 10000, 15000},
		Slot_target_cost:  100,
		Slot_max_cost:     200,
		ZDR:               0,    // no zero days
		REL:               0.95, // stable late performance
	}
}

func TestBreakout_AllConditionsMet(t *testing.T) {
	in := breakoutInput()
	r := algo.DetectBreakout(in, 15.0, algo.DefaultBreakoutThresholds())
	if !r.Is_breakout {
		t.Errorf("expected breakout, failed: %s", r.Reason_failed)
	}
	if r.PDC_override != 15.0 {
		t.Errorf("expected PDC_override=15.0 (mu_st), got %.2f", r.PDC_override)
	}
	if r.Tier_jump != 2 {
		t.Errorf("expected tier_jump=2, got %d", r.Tier_jump)
	}
}

func TestBreakout_Condition1Fails_LowPDC(t *testing.T) {
	in := breakoutInput()
	in.Total_conversions = 55 // ~9.2/day — below 1.20 × 10
	r := algo.DetectBreakout(in, 9.2, algo.DefaultBreakoutThresholds())
	if r.Is_breakout {
		t.Error("should not be breakout when PDC overperformance not met")
	}
}

func TestBreakout_Condition3Fails_Fatigue(t *testing.T) {
	in := breakoutInput()
	in.ZDR = 0.4 // zero days present → fatigue
	r := algo.DetectBreakout(in, 15.0, algo.DefaultBreakoutThresholds())
	if r.Is_breakout {
		t.Error("should not be breakout when fatigue detected (ZDR > 0)")
	}
}

func TestBreakout_RelationshipTag_AllRound(t *testing.T) {
	in := breakoutInput()
	in.Match_dimensions = algo.MatchDimensions{DM: 0.95, GP: 0.95, OH: 0.95}
	r := algo.DetectBreakout(in, 15.0, algo.DefaultBreakoutThresholds())
	if r.Relationship_tag != "strong_all_round_match" {
		t.Errorf("expected strong_all_round_match, got %s", r.Relationship_tag)
	}
}

func TestBreakout_RelationshipTag_AudienceProximity(t *testing.T) {
	in := breakoutInput()
	in.Match_dimensions = algo.MatchDimensions{DM: 0.95, GP: 0.95, OH: 0.3}
	r := algo.DetectBreakout(in, 15.0, algo.DefaultBreakoutThresholds())
	if r.Relationship_tag != "audience_proximity_driven" {
		t.Errorf("expected audience_proximity_driven, got %s", r.Relationship_tag)
	}
}

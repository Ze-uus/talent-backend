package algo

// BreakoutInput holds all data needed to evaluate breakout conditions at cycle end.
type BreakoutInput struct {
	Talent_id         string
	Cycle_id          string
	PDC_allocated     float64
	Total_conversions float64
	Cycle_days        int
	Starting_tier     int
	All_tiers         []int
	Slot_target_cost  float64
	Slot_max_cost     float64
	ZDR               float64
	REL               float64
	Match_dimensions  MatchDimensions
}

// BreakoutResult is the output per talent at cycle end.
type BreakoutResult struct {
	Is_breakout      bool
	Reason_failed    string // which condition(s) not met; empty if breakout confirmed
	PDC_override     float64
	Tier_jump        int    // tiers above starting_tier eligible for next cycle
	Relationship_tag string // insight tag for reporting (Story 18 A.12)
}

// BreakoutThresholds holds configurable breakout detection parameters (Story 19).
type BreakoutThresholds struct {
	PDC_overperformance float64 // default 1.20 (120%)
	REL_min             float64 // default 0.85 — no late-cycle collapse
	Tier_jump           int     // default 2
}

// DefaultBreakoutThresholds returns the platform defaults.
func DefaultBreakoutThresholds() BreakoutThresholds {
	return BreakoutThresholds{
		PDC_overperformance: 1.20,
		REL_min:             0.85,
		Tier_jump:           2,
	}
}

// DetectBreakout evaluates all three breakout conditions for one talent at cycle end.
// mu_st is the weighted short-term mean from ComputeSTEC — used as PDC_override on success.
// All three conditions must hold simultaneously for a breakout to be confirmed.
func DetectBreakout(in BreakoutInput, mu_st float64, thresholds BreakoutThresholds) BreakoutResult {
	result := BreakoutResult{Tier_jump: thresholds.Tier_jump}

	// Condition 1: actual daily average > PDC_allocated × threshold
	actual_daily_avg := in.Total_conversions / float64(in.Cycle_days)
	condition1 := actual_daily_avg > in.PDC_allocated*thresholds.PDC_overperformance

	// Condition 2: total conversions exceeded slot max conversions
	slot := BudgetSlot{
		Tier_value:  in.Starting_tier,
		Target_cost: in.Slot_target_cost,
		Max_cost:    in.Slot_max_cost,
	}
	bounds, err := TierBoundsFor(slot)
	condition2 := err == nil && in.Total_conversions > float64(bounds.Max)

	// Condition 3: no fatigue — ZDR=0 AND REL ≥ threshold
	condition3 := in.ZDR == 0 && in.REL >= thresholds.REL_min

	if !condition1 || !condition2 || !condition3 {
		failed := ""
		if !condition1 {
			failed += "pdc_overperformance_not_met "
		}
		if !condition2 {
			failed += "tier_max_not_exceeded "
		}
		if !condition3 {
			failed += "fatigue_detected"
		}
		result.Is_breakout = false
		result.Reason_failed = failed
		return result
	}

	result.Is_breakout = true
	result.PDC_override = mu_st
	result.Relationship_tag = buildRelationshipTag(in.Match_dimensions)
	return result
}

func buildRelationshipTag(d MatchDimensions) string {
	if d.DM >= 0.9 && d.GP >= 0.9 && d.OH >= 0.9 {
		return "strong_all_round_match"
	}
	if d.DM >= 0.9 && d.GP >= 0.9 {
		return "audience_proximity_driven"
	}
	if d.OH >= 0.9 {
		return "objective_alignment_driven"
	}
	if d.DM >= 0.9 {
		return "demographic_match_driven"
	}
	return "performance_driven"
}

package algo

import (
	"errors"
	"fmt"
	"math"
)

type TalentProfile struct {
	ID           string  `json:"id"`
	PDC          float64 `json:"pdc"`
	Max_tier     int     `json:"max_tier"`
	Cycle_length int     `json:"cycle_length"`
}

type BudgetSlot struct {
	ID          string  `json:"id"`
	Tier_value  int     `json:"tier_value"`
	Target_cost float64 `json:"target_cost"`
	Max_cost    float64 `json:"max_cost"`
}

type TierBounds struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type QualResult struct {
	Qualified bool    `json:"qualified"`
	Reason    string  `json:"reason"`
	Base_days float64 `json:"base_days"`
	Actual    float64 `json:"actual"`
}

func TierBoundsFor(slot BudgetSlot) (TierBounds, error) {
	if slot.Max_cost <= 0 {
		return TierBounds{}, errors.New("max_cost must be positive")
	}
	if slot.Target_cost <= 0 {
		return TierBounds{}, errors.New("target_cost must be positive")
	}
	if slot.Tier_value <= 0 {
		return TierBounds{}, errors.New("tier_value must be positive")
	}
	return TierBounds{
		Min: int(math.Ceil(float64(slot.Tier_value) / slot.Max_cost)),
		Max: int(math.Floor(float64(slot.Tier_value) / slot.Target_cost)),
	}, nil
}

// DemandTier computes a talent's natural demand tier per SOP section 9.
// demand_tier = PDC * cycle_length * max_CPA, rounded down to nearest tier_increment.
func DemandTier(pdc float64, cycle_length int, max_cpa float64, tier_increment int) int {
	if pdc <= 0 || cycle_length <= 0 || max_cpa <= 0 || tier_increment <= 0 {
		return 0
	}
	if !isFinite(pdc) || !isFinite(max_cpa) {
		return 0
	}
	raw := pdc * float64(cycle_length) * max_cpa
	return int(math.Floor(raw/float64(tier_increment))) * tier_increment
}

// BoundedDemandTier maps natural demand to a tier that the waterfall can
// actually consume. It respects the cycle budget share, platform cap, and
// talent-specific cap.
func BoundedDemandTier(
	pdc float64,
	cycle_length int,
	max_cpa float64,
	cycle_budget float64,
	talent_max_tier int,
	cfg CycleConfig,
) int {
	natural := DemandTier(pdc, cycle_length, max_cpa, cfg.Tier_increment)
	if natural == 0 || cycle_budget <= 0 {
		return 0
	}

	upper := cfg.Max_tier_cap
	budgetUpper := int(math.Floor(
		(cycle_budget*cfg.Budget_share_pct)/float64(cfg.Tier_increment),
	)) * cfg.Tier_increment
	if budgetUpper < upper {
		upper = budgetUpper
	}
	if talent_max_tier > 0 && talent_max_tier < upper {
		upper = talent_max_tier
	}
	upper = (upper / cfg.Tier_increment) * cfg.Tier_increment
	if upper < cfg.Min_tier {
		return 0
	}
	if natural > upper {
		return upper
	}
	if natural < cfg.Min_tier {
		return cfg.Min_tier
	}
	return natural
}

func QualifyTalentForSlot(t TalentProfile, slot BudgetSlot, all_tiers []int) (QualResult, error) {
	if !isFinite(t.PDC) {
		return QualResult{}, fmt.Errorf("talent %s has non-finite PDC: %f", t.ID, t.PDC)
	}
	if t.PDC <= 0 {
		return QualResult{Qualified: false, Reason: "too_weak"}, nil
	}
	if t.Cycle_length <= 0 {
		return QualResult{}, fmt.Errorf("talent %s has invalid cycle_length: %d", t.ID, t.Cycle_length)
	}
	if len(all_tiers) == 0 {
		return QualResult{}, errors.New("all_tiers must not be empty")
	}

	bounds, err := TierBoundsFor(slot)
	if err != nil {
		return QualResult{}, fmt.Errorf("invalid slot %s: %w", slot.ID, err)
	}

	tiers := sortedTiers(all_tiers)
	max_idx := tierIndex(tiers, t.Max_tier)
	slot_idx := tierIndex(tiers, slot.Tier_value)

	if max_idx >= 0 && slot_idx >= 0 && max_idx-slot_idx > 1 {
		return QualResult{Qualified: false, Reason: "mobility_blocked"}, nil
	}

	threshold := float64(bounds.Min) * 0.80
	reachable := t.PDC * float64(t.Cycle_length)
	if reachable < threshold {
		return QualResult{Qualified: false, Reason: "too_weak"}, nil
	}

	base_days := math.Ceil(float64(bounds.Min) / t.PDC)
	actual := base_days * t.PDC

	return QualResult{Qualified: true, Reason: "ok", Base_days: base_days, Actual: actual}, nil
}

func sortedTiers(tiers []int) []int {
	sorted := make([]int, len(tiers))
	copy(sorted, tiers)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	return sorted
}

func tierIndex(sorted []int, tier int) int {
	for i, v := range sorted {
		if v == tier {
			return i
		}
	}
	return -1
}

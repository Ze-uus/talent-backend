package algo

import (
	"fmt"
	"math"
)

type WaterfallInput struct {
	Cycle_budget    float64        `json:"cycle_budget"`
	Tiers           []int          `json:"tiers"`
	Talents_by_tier map[int]int    `json:"talents_by_tier"`
	Config          CycleConfig    `json:"config"`
}

type TierSlot struct {
	Tier         int     `json:"tier"`
	Slot_count   int     `json:"slot_count"`
	Budget_share float64 `json:"budget_share"`
	Remainder    float64 `json:"remainder"`
}

type WaterfallOutput struct {
	Slots           []TierSlot `json:"slots"`
	Total_allocated float64    `json:"total_allocated"`
	Remainder       float64    `json:"remainder"`
}

func RunWaterfall(in WaterfallInput) (WaterfallOutput, error) {
	if err := in.Config.Validate(); err != nil {
		return WaterfallOutput{}, fmt.Errorf("invalid config: %w", err)
	}
	if err := requireNonNegative("cycle_budget", in.Cycle_budget); err != nil {
		return WaterfallOutput{}, err
	}
	if in.Cycle_budget == 0 || len(in.Tiers) == 0 {
		return WaterfallOutput{}, nil
	}

	max_tier_value := in.Cycle_budget * in.Config.Budget_share_pct
	if float64(in.Config.Max_tier_cap) < max_tier_value {
		max_tier_value = float64(in.Config.Max_tier_cap)
	}

	demand := make(map[int]float64)
	total_demand := 0.0

	for _, tier := range in.Tiers {
		if tier < in.Config.Min_tier || float64(tier) > max_tier_value {
			continue
		}
		count := float64(in.Talents_by_tier[tier])
		if count <= 0 {
			continue
		}
		d := count * float64(tier)
		demand[tier] = d
		total_demand += d
	}

	if total_demand == 0 {
		return WaterfallOutput{}, nil
	}

	scale_factor := 1.0
	if total_demand > in.Cycle_budget {
		scale_factor = in.Cycle_budget / total_demand
	}

	var slots []TierSlot
	total_allocated := 0.0

	for _, tier := range in.Tiers {
		d, ok := demand[tier]
		if !ok || d == 0 {
			continue
		}
		weight := d / total_demand
		share := weight * in.Cycle_budget * scale_factor
		count := math.Floor(share / float64(tier))
		actual := count * float64(tier)
		total_allocated += actual

		slots = append(slots, TierSlot{
			Tier:         tier,
			Slot_count:   int(count),
			Budget_share: actual,
			Remainder:    share - actual,
		})
	}

	return WaterfallOutput{
		Slots:           slots,
		Total_allocated: total_allocated,
		Remainder:       in.Cycle_budget - total_allocated,
	}, nil
}

package allocation

import (
	"errors"
	"math"

	"github.com/google/uuid"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/store"
)

// ValidateCycle ensures at least one target-CPA unit can be funded.
func ValidateCycle(cycle store.Cycle, campaign store.Campaign) error {
	if campaign.Target_cpa <= 0 || campaign.Max_cpa <= 0 {
		return errors.New("invalid_campaign_cpa")
	}
	if campaign.Target_cpa > campaign.Max_cpa {
		return errors.New("target_cpa_exceeds_max_cpa")
	}
	if cycle.Cycle_budget < campaign.Target_cpa {
		return errors.New("cycle_budget_below_target_cpa")
	}
	return nil
}

// BuildBudgetSlots creates at most one automatically-sized slot per active
// talent. Max_tier=0 means automatic; a positive value is an admin ceiling.
func BuildBudgetSlots(cycle store.Cycle, campaign store.Campaign, talents []store.Talent) ([]store.BudgetSlot, error) {
	var active []store.Talent
	for _, talent := range talents {
		if talent.Status == store.Status_active {
			active = append(active, talent)
		}
	}
	if len(active) == 0 {
		return nil, nil
	}
	if err := ValidateCycle(cycle, campaign); err != nil {
		return nil, err
	}

	cfg := algo.DefaultConfig()
	increment := cfg.Tier_increment
	minTier := int(math.Ceil(campaign.Target_cpa/float64(increment))) * increment
	capacity := int(cycle.Cycle_budget) / minTier
	if capacity == 0 {
		return nil, errors.New("cycle_budget_below_target_cpa")
	}
	if capacity > len(active) {
		capacity = len(active)
	}
	fairBudget := (int(cycle.Cycle_budget) / capacity / increment) * increment

	slots := make([]store.BudgetSlot, 0, capacity)
	tierIndexes := make(map[int]int)
	for _, talent := range active {
		natural := algo.DemandTier(
			float64(categoryPDC(talent.Category)),
			campaign.Cycle_length,
			campaign.Max_cpa,
			increment,
		)
		if natural < minTier {
			natural = minTier
		}
		tier := natural
		if tier > fairBudget {
			tier = fairBudget
		}
		if talent.Max_tier > 0 && tier > talent.Max_tier {
			tier = (talent.Max_tier / increment) * increment
		}
		if tier < minTier {
			continue
		}
		index := tierIndexes[tier]
		tierIndexes[tier] = index + 1
		slots = append(slots, store.BudgetSlot{
			ID:         uuid.NewString(),
			Cycle_id:   cycle.ID,
			Tier_value: tier,
			Slot_index: index,
		})
		if len(slots) == capacity {
			break
		}
	}
	return slots, nil
}

func categoryPDC(category store.Talent_category) int {
	switch category {
	case store.Category_student:
		return 2
	case store.Category_micro:
		return 5
	case store.Category_community:
		return 15
	default:
		return 2
	}
}

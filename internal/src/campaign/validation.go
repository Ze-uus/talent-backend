package campaign

import (
	"math"
	"strings"

	"github.com/Ze-uus/talent-backend/internal/response"
	"github.com/Ze-uus/talent-backend/internal/store"
)

const (
	maxSupportedCPA = 1_000_000_000.0
	maxStoredMoney  = 999_999_999_999.99
)

func validateCampaign(c store.Campaign) error {
	if strings.TrimSpace(c.Brand_id) == "" {
		return response.Validation("brand_id_required")
	}
	if strings.TrimSpace(c.Name) == "" {
		return response.Validation("campaign_name_required")
	}
	if c.Campaign_type != store.Type_direct_traffic && c.Campaign_type != store.Type_lead_validation {
		return response.Validation("invalid_campaign_type")
	}
	if !positiveFinite(c.Total_budget) {
		return response.Validation("total_budget_required")
	}
	if c.Total_budget > maxStoredMoney {
		return response.Validation("total_budget_exceeds_supported_range")
	}
	if !positiveFinite(c.Target_cpa) || !positiveFinite(c.Max_cpa) {
		return response.Validation("invalid_campaign_cpa")
	}
	if c.Target_cpa > maxSupportedCPA || c.Max_cpa > maxSupportedCPA {
		return response.Validation("cpa_exceeds_supported_range")
	}
	if c.Target_cpa > c.Max_cpa {
		return response.Validation("target_cpa_exceeds_max_cpa")
	}
	if c.Max_cpa > c.Total_budget {
		return response.Validation("max_cpa_exceeds_total_budget")
	}
	if c.Cycle_length < 5 || c.Cycle_length > 10 {
		return response.Validation("cycle_length_must_be_between_5_and_10")
	}
	return nil
}

func validateCampaignPatch(patch store.CampaignPatch) error {
	if patch.Name != nil && strings.TrimSpace(*patch.Name) == "" {
		return response.Validation("campaign_name_required")
	}
	if patch.Status != nil && !validCampaignStatus(*patch.Status) {
		return response.Validation("invalid_campaign_status")
	}
	if patch.Urgency_level != nil && !validUrgency(*patch.Urgency_level) {
		return response.Validation("invalid_urgency_level")
	}
	if patch.Cycle_length != nil && (*patch.Cycle_length < 5 || *patch.Cycle_length > 10) {
		return response.Validation("cycle_length_must_be_between_5_and_10")
	}
	return nil
}

func validateCycleBudget(campaign store.Campaign, budget float64) error {
	if !positiveFinite(budget) {
		return response.Validation("cycle_budget_required")
	}
	if budget > maxStoredMoney {
		return response.Validation("cycle_budget_exceeds_supported_range")
	}
	if budget < campaign.Target_cpa {
		return response.Validation("cycle_budget_below_target_cpa")
	}
	if budget < campaign.Max_cpa {
		return response.Validation("cycle_budget_below_max_cpa")
	}
	if budget > campaign.Remaining_budget {
		return response.Validation("cycle_budget_exceeds_remaining_campaign_budget")
	}
	return nil
}

func remainingCycleBudget(campaign store.Campaign, cycles []store.Cycle, excludeCycleID string) float64 {
	total := campaign.Total_budget
	if total <= 0 {
		total = campaign.Remaining_budget
	}
	for _, cycle := range cycles {
		if cycle.ID != excludeCycleID {
			total -= cycle.Cycle_budget
		}
	}
	if total < 0 {
		return 0
	}
	return total
}

func positiveFinite(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func validCampaignStatus(status string) bool {
	switch store.Campaign_status(status) {
	case store.Campaign_draft, store.Campaign_active, store.Campaign_paused,
		store.Campaign_closed, store.Campaign_archived:
		return true
	default:
		return false
	}
}

func validUrgency(urgency string) bool {
	switch store.Urgency_level(urgency) {
	case store.Urgency_low, store.Urgency_normal, store.Urgency_high:
		return true
	default:
		return false
	}
}

func validCycleStatus(status string) bool {
	switch store.Cycle_status(status) {
	case store.Cycle_pending, store.Cycle_active, store.Cycle_paused, store.Cycle_closed:
		return true
	default:
		return false
	}
}

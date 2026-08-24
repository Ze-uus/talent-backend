package campaign

import (
	"testing"

	"github.com/Ze-uus/talent-backend/internal/response"
	"github.com/Ze-uus/talent-backend/internal/store"
)

func validCampaignForValidation() store.Campaign {
	return store.Campaign{
		Brand_id: "brand-1", Name: "Campaign",
		Campaign_type: store.Type_direct_traffic,
		Total_budget:  1_000_000_000, Target_cpa: 500_000_000,
		Max_cpa: 1_000_000_000, Cycle_length: 7,
	}
}

func TestValidateCampaignAcceptsBillionCPABoundary(t *testing.T) {
	if err := validateCampaign(validCampaignForValidation()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCampaignReturnsSpecificCodes(t *testing.T) {
	tests := []struct {
		name string
		edit func(*store.Campaign)
		code string
	}{
		{"over supported CPA", func(c *store.Campaign) { c.Max_cpa++ }, "cpa_exceeds_supported_range"},
		{"target over max", func(c *store.Campaign) {
			c.Target_cpa = 900_000_000
			c.Max_cpa = 800_000_000
		}, "target_cpa_exceeds_max_cpa"},
		{"max over total", func(c *store.Campaign) { c.Total_budget = c.Max_cpa - 1 }, "max_cpa_exceeds_total_budget"},
		{"invalid cycle length", func(c *store.Campaign) { c.Cycle_length = 4 }, "cycle_length_must_be_between_5_and_10"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			campaign := validCampaignForValidation()
			tt.edit(&campaign)
			err := validateCampaign(campaign)
			status, code := response.MapError(err)
			if status != 422 || code != tt.code {
				t.Fatalf("got %d/%s from %v", status, code, err)
			}
		})
	}
}

func TestValidateCycleBudget(t *testing.T) {
	campaign := validCampaignForValidation()
	campaign.Remaining_budget = campaign.Total_budget
	if err := validateCycleBudget(campaign, campaign.Max_cpa); err != nil {
		t.Fatal(err)
	}
	if err := validateCycleBudget(campaign, campaign.Max_cpa-1); err == nil ||
		err.Error() != "cycle_budget_below_max_cpa" {
		t.Fatalf("unexpected error: %v", err)
	}
}

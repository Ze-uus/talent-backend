package campaign

import "github.com/Ze-uus/talent-backend/internal/store"

// CampaignManagerView is the staff summary returned on campaign detail.
type CampaignManagerView struct {
	ID        string            `json:"id"`
	Email     string            `json:"email"`
	Full_name string            `json:"full_name"`
	Role      store.User_role   `json:"role"`
	Status    store.User_status `json:"status"`
	Active    bool              `json:"active"`
}

// CampaignDetail is a campaign plus assigned managers for display.
type CampaignDetail struct {
	store.Campaign
	Campaign_managers []CampaignManagerView `json:"campaign_managers"`
}

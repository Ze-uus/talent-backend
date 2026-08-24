package tracking

import (
	"time"

	"github.com/Ze-uus/talent-backend/internal/store"
)

type PresentationView struct {
	Brand    PresentationBrand    `json:"brand"`
	Campaign PresentationCampaign `json:"campaign"`
	Cycle    PresentationCycle    `json:"cycle"`
	Content  []store.ContentItem  `json:"content"`
}

type PresentationBrand struct {
	Name        string `json:"name"`
	Industry    string `json:"industry"`
	Description string `json:"description"`
	Website     string `json:"website"`
	LogoURL     string `json:"logo_url"`
}

type PresentationCampaign struct {
	HumanID      string              `json:"human_id"`
	Name         string              `json:"name"`
	CampaignType store.Campaign_type `json:"campaign_type"`
	Audience     string              `json:"audience"`
}

type PresentationCycle struct {
	HumanID        string              `json:"human_id"`
	CycleNumber    int                 `json:"cycle_number"`
	CycleObjective string              `json:"cycle_objective"`
	CampaignType   store.Campaign_type `json:"campaign_type"`
	KPBLabels      []string            `json:"kpb_labels"`
	StartDate      time.Time           `json:"start_date"`
	EndDate        time.Time           `json:"end_date"`
}

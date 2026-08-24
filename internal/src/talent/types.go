package talent

import (
	"time"

	"github.com/Ze-uus/talent-backend/internal/store"
)

// CycleView is the talent-safe view of an assigned campaign cycle.
type CycleView struct {
	Assignment       AssignmentView    `json:"assignment"`
	Campaign         CampaignView      `json:"campaign"`
	Brand            BrandView         `json:"brand"`
	Cycle            CycleSummary      `json:"cycle"`
	CampaignManagers []ManagerView     `json:"campaign_managers"`
	TrackingLink     *TrackingLinkView `json:"tracking_link"`
	Stats            CycleStatsView    `json:"stats"`
}

type AssignmentView struct {
	TalentID      string                  `json:"talent_id"`
	CampaignID    string                  `json:"campaign_id"`
	CycleID       string                  `json:"cycle_id"`
	SlotID        string                  `json:"slot_id"`
	RoleLabel     string                  `json:"role_label"`
	Status        string                  `json:"status"`
	Source        store.Assignment_source `json:"assignment_source"`
	EffectiveTier int                     `json:"effective_tier"`
	AssignedAt    time.Time               `json:"assigned_at"`
}

type CampaignView struct {
	ID              string                `json:"id"`
	HumanID         string                `json:"human_id"`
	Name            string                `json:"name"`
	Status          store.Campaign_status `json:"status"`
	CampaignType    store.Campaign_type   `json:"campaign_type"`
	Audience        string                `json:"audience"`
	TargetCPA       float64               `json:"target_cpa"`
	MaxCPA          float64               `json:"max_cpa"`
	CreatorsAllowed bool                  `json:"creators_allowed"`
}

type BrandView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Industry    string `json:"industry"`
	Description string `json:"description"`
	Website     string `json:"website"`
	LogoURL     string `json:"logo_url"`
}

type CycleSummary struct {
	ID             string                `json:"id"`
	HumanID        string                `json:"human_id"`
	CampaignID     string                `json:"campaign_id"`
	CycleNumber    int                   `json:"cycle_number"`
	Status         store.Cycle_status    `json:"status"`
	CycleObjective string                `json:"cycle_objective"`
	CampaignType   store.Campaign_type   `json:"campaign_type"`
	KPBConfig      []store.KPBDefinition `json:"kpb_config"`
	Content        []store.ContentItem   `json:"content"`
	StartDate      time.Time             `json:"start_date"`
	EndDate        time.Time             `json:"end_date"`
}

type ManagerView struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

type TrackingLinkView struct {
	Token         string `json:"token"`
	Active        bool   `json:"active"`
	EventEndpoint string `json:"event_endpoint"`
}

type CycleStatsView struct {
	MyConversions float64 `json:"my_conversions"`
	CycleTotal    float64 `json:"cycle_total"`
}

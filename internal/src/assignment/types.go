package assignment

import (
	"time"

	"github.com/Ze-uus/talent-backend/internal/store"
)

type AddHumanInput struct {
	TalentID      string `json:"talent_id"`
	EffectiveTier int    `json:"effective_tier"`
	RoleLabel     string `json:"role_label,omitempty"`
}

type SlotView struct {
	ID        string `json:"id"`
	TierValue int    `json:"tier_value"`
	SlotIndex int    `json:"slot_index"`
	Allocated bool   `json:"allocated"`
	TalentID  string `json:"talent_id,omitempty"`
}

type HumanAssignmentView struct {
	TalentID     string                  `json:"talent_id"`
	Human        AssignmentHuman         `json:"human"`
	Assignment   AssignmentSummary       `json:"assignment"`
	TrackingLink *AssignmentTrackingLink `json:"tracking_link"`
	Campaign     AssignmentCampaign      `json:"campaign"`
	Cycle        AssignmentCycle         `json:"cycle"`
	Performance  AssignmentPerformance   `json:"performance"`
	Earnings     AssignmentEarnings      `json:"earnings"`
}

type AssignmentHuman struct {
	UserID           string                `json:"user_id"`
	FullName         string                `json:"full_name"`
	Email            string                `json:"email"`
	PhoneNumber      string                `json:"phone_number"`
	AvatarURL        string                `json:"avatar_url"`
	Category         store.Talent_category `json:"category"`
	Status           store.Talent_status   `json:"status"`
	Skills           []string              `json:"skills"`
	RatePerDay       float64               `json:"rate_per_day"`
	ReportCompliance float64               `json:"report_compliance"`
}

type AssignmentSummary struct {
	RoleLabel        string                  `json:"role_label"`
	Status           string                  `json:"status"`
	AssignmentSource store.Assignment_source `json:"assignment_source"`
	PDCMode          store.Pdc_mode          `json:"pdc_mode"`
	PDCValue         float64                 `json:"pdc_value"`
	EffectiveTier    int                     `json:"effective_tier"`
	AssignedAt       time.Time               `json:"assigned_at"`
}

type AssignmentTrackingLink struct {
	Token  string `json:"token"`
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

type AssignmentCampaign struct {
	ID          string                `json:"id"`
	HumanID     string                `json:"human_id"`
	Name        string                `json:"name"`
	Status      store.Campaign_status `json:"status"`
	Type        store.Campaign_type   `json:"campaign_type"`
	TotalBudget float64               `json:"total_budget"`
	TargetCPA   float64               `json:"target_cpa"`
	MaxCPA      float64               `json:"max_cpa"`
}

type AssignmentCycle struct {
	ID              string             `json:"id"`
	HumanID         string             `json:"human_id"`
	CycleNumber     int                `json:"cycle_number"`
	Status          store.Cycle_status `json:"status"`
	CycleLength     int                `json:"cycle_length"`
	CycleBudget     float64            `json:"cycle_budget"`
	AllocatedBudget float64            `json:"allocated_budget"`
	StartDate       time.Time          `json:"start_date"`
	EndDate         time.Time          `json:"end_date"`
}

type AssignmentPerformance struct {
	AOC         float64 `json:"aoc"`
	Conversions float64 `json:"conversions"`
	MatchScore  float64 `json:"match_score"`
	MatchDM     float64 `json:"match_dm"`
	MatchGP     float64 `json:"match_gp"`
	MatchOH     float64 `json:"match_oh"`
}

type AssignmentEarnings struct {
	TotalEarned   float64             `json:"total_earned"`
	PaymentStatus store.Payout_status `json:"payment_status"`
}

package admin

import (
	"time"

	"github.com/Ze-uus/talent-backend/internal/store"
)

// TalentStats are aggregate metrics for the Humans admin list.
type TalentStats struct {
	Human_earnings  float64 `json:"human_earnings"`
	Humans          int     `json:"humans"`
	Active_humans   int     `json:"active_humans"`
	Avg_performance float64 `json:"avg_performance"`
}

// TalentListItem is a talent plus display fields from the linked user.
type TalentListItem struct {
	store.Talent
	Full_name    string       `json:"full_name"`
	Email        string       `json:"email"`
	Phone_number string       `json:"phone_number"`
	Avatar_url   string       `json:"avatar_url"`
	User         SafeUserView `json:"user"`
}

// TalentListResult is the wrapped Humans list response.
type TalentListResult struct {
	Stats   TalentStats      `json:"stats"`
	Talents []TalentListItem `json:"talents"`
}

type SafeUserView struct {
	ID          string              `json:"id"`
	Email       string              `json:"email"`
	FullName    string              `json:"full_name"`
	PhoneNumber string              `json:"phone_number"`
	AvatarURL   string              `json:"avatar_url"`
	Role        store.User_role     `json:"role"`
	Provider    store.Auth_provider `json:"provider"`
	Status      store.User_status   `json:"status"`
	Active      bool                `json:"active"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type TalentView struct {
	ID               string                `json:"id"`
	UserID           string                `json:"user_id"`
	Category         store.Talent_category `json:"category"`
	Status           store.Talent_status   `json:"status"`
	Skills           []string              `json:"skills"`
	RatePerDay       float64               `json:"rate_per_day"`
	MaxTier          int                   `json:"max_tier"`
	Bio              string                `json:"bio"`
	PortfolioURL     string                `json:"portfolio_url"`
	ReportCompliance float64               `json:"report_compliance"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
}

type TalentAssignmentSummary struct {
	CampaignID    string    `json:"campaign_id"`
	CycleID       string    `json:"cycle_id"`
	RoleLabel     string    `json:"role_label"`
	Status        string    `json:"status"`
	EffectiveTier int       `json:"effective_tier"`
	MatchScore    float64   `json:"match_score"`
	AssignedAt    time.Time `json:"assigned_at"`
}

type TalentPayoutSummary struct {
	CampaignID  string              `json:"campaign_id"`
	CycleID     string              `json:"cycle_id"`
	Status      store.Payout_status `json:"status"`
	FinalPayout float64             `json:"final_payout"`
	PaidAt      time.Time           `json:"paid_at"`
}

type TalentDetailStats struct {
	TotalAssignments  int     `json:"total_assignments"`
	ActiveAssignments int     `json:"active_assignments"`
	TotalConversions  float64 `json:"total_conversions"`
	AverageMatchScore float64 `json:"average_match_score"`
	TotalEarned       float64 `json:"total_earned"`
	TotalPaid         float64 `json:"total_paid"`
}

type TalentDetail struct {
	Talent      TalentView                `json:"talent"`
	User        SafeUserView              `json:"user"`
	Stats       TalentDetailStats         `json:"stats"`
	Assignments []TalentAssignmentSummary `json:"assignments"`
	Payouts     []TalentPayoutSummary     `json:"payouts"`
}

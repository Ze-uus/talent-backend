package admin

import "github.com/Ze-uus/talent-backend/internal/store"

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
	Full_name string `json:"full_name"`
	Email     string `json:"email"`
}

// TalentListResult is the wrapped Humans list response.
type TalentListResult struct {
	Stats   TalentStats      `json:"stats"`
	Talents []TalentListItem `json:"talents"`
}

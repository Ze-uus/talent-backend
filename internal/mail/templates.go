package mail

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

// Render builds an HTML body from a named template and data.
// Each call parses layout.html together with the named template so the
// shared "body" define does not collide across emails.
func Render(name string, data any) (string, error) {
	t, err := template.New("").ParseFS(templateFS, "templates/layout.html", "templates/"+name)
	if err != nil {
		return "", fmt.Errorf("mail: load %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("mail: render %s: %w", name, err)
	}
	return buf.String(), nil
}

// BaseData is shared by all email templates.
type BaseData struct {
	AppURL    string
	Name      string
	Subject   string
	Preheader string
}

type InviteData struct {
	BaseData
	Role      string
	InviteURL string
}

type TalentStatusData struct {
	BaseData
	LoginURL string
	Reason   string
}

type PayoutPaidData struct {
	BaseData
	Amount   string
	CycleID  string
	LoginURL string
}

type CampaignCycleData struct {
	BaseData
	CampaignName string
	CycleNumber  int
	HumanID      string
	StartDate    string
	EndDate      string
	DashboardURL string
	StatsSummary string
}

type ManagerAssignData struct {
	BaseData
	CampaignName string
	HumanID      string
	Action       string // "assigned" | "removed"
	DashboardURL string
}

type AssignmentData struct {
	BaseData
	CampaignName string
	CycleNumber  int
	Tier         int
	DashboardURL string
}

type BrandStatsData struct {
	BaseData
	BrandName    string
	CampaignName string
	HumanID      string
	Conversions  float64
	DashboardURL string
	PeriodLabel  string
}

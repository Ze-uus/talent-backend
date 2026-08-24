package mail

import (
	"fmt"
	"log/slog"
)

// Service wraps a Mailer with APP_URL and convenience notify helpers.
type Service struct {
	Mailer Mailer
	AppURL string
	Log    *slog.Logger
}

func NewService(m Mailer, appURL string, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{Mailer: m, AppURL: appURL, Log: log}
}

func (s *Service) send(to, subject, tmplName string, data any) {
	if to == "" {
		return
	}
	html, err := Render(tmplName, data)
	if err != nil {
		s.Log.Error("mail: render failed", "template", tmplName, "err", err)
		return
	}
	s.Mailer.SendAsync(Message{To: to, Subject: subject, HTML: html})
}

func (s *Service) base(name string) BaseData {
	return BaseData{AppURL: s.AppURL, Name: name}
}

// ─── Auth / invite ────────────────────────────────────────────────────────────

func (s *Service) NotifyTalentRegistered(to, name string) {
	d := TalentStatusData{
		BaseData: s.base(name),
		LoginURL: s.AppURL + "/login",
	}
	d.Subject = "Welcome to Scaloo — registration received"
	d.Preheader = "Your Scaloo registration is pending approval."
	s.send(to, d.Subject, "talent_registered.html", d)
}

func (s *Service) NotifyStaffInvite(to, name, role, token string) {
	d := InviteData{
		BaseData:  s.base(name),
		Role:      role,
		InviteURL: fmt.Sprintf("%s/invite?token=%s", s.AppURL, token),
	}
	d.Subject = "You're invited to Scaloo"
	d.Preheader = "You have been invited to Scaloo."
	s.send(to, d.Subject, "staff_invite.html", d)
}

// ─── Talent lifecycle ─────────────────────────────────────────────────────────

func (s *Service) NotifyTalentApproved(to, name string) {
	d := TalentStatusData{BaseData: s.base(name), LoginURL: s.AppURL + "/login"}
	d.Subject = "Your Scaloo account is approved"
	d.Preheader = "Your Scaloo account has been approved."
	s.send(to, d.Subject, "talent_approved.html", d)
}

func (s *Service) NotifyTalentSuspended(to, name string) {
	d := TalentStatusData{BaseData: s.base(name), LoginURL: s.AppURL + "/login"}
	d.Subject = "Your Scaloo account has been suspended"
	d.Preheader = "Your Scaloo account has been suspended."
	s.send(to, d.Subject, "talent_suspended.html", d)
}

func (s *Service) NotifyTalentRejected(to, name string) {
	d := TalentStatusData{BaseData: s.base(name), LoginURL: s.AppURL + "/login"}
	d.Subject = "Scaloo application update"
	d.Preheader = "Your Scaloo application was not approved."
	s.send(to, d.Subject, "talent_rejected.html", d)
}

func (s *Service) NotifyAccountDeactivated(to, name string) {
	d := TalentStatusData{BaseData: s.base(name), LoginURL: s.AppURL + "/login"}
	d.Subject = "Your Scaloo account has been deactivated"
	d.Preheader = "Your Scaloo account has been deactivated."
	s.send(to, d.Subject, "account_deactivated.html", d)
}

func (s *Service) NotifyTalentReinstated(to, name string) {
	d := TalentStatusData{BaseData: s.base(name), LoginURL: s.AppURL + "/login"}
	d.Subject = "Your Scaloo account has been reinstated"
	d.Preheader = "Your Scaloo account has been reinstated."
	s.send(to, d.Subject, "talent_reinstated.html", d)
}

// ─── Payout ───────────────────────────────────────────────────────────────────

func (s *Service) NotifyPayoutPaid(to, name, amount, cycleID string) {
	d := PayoutPaidData{
		BaseData: s.base(name),
		Amount:   amount,
		CycleID:  cycleID,
		LoginURL: s.AppURL + "/login",
	}
	d.Subject = "Your Scaloo payout has been paid"
	d.Preheader = "Your payout has been marked as paid."
	s.send(to, d.Subject, "payout_paid.html", d)
}

// ─── Campaign / cycle ─────────────────────────────────────────────────────────

func (s *Service) NotifyCampaignStarted(to, name, campaignName, humanID string) {
	d := CampaignCycleData{
		BaseData:     s.base(name),
		CampaignName: campaignName,
		HumanID:      humanID,
		DashboardURL: s.AppURL + "/dashboard",
	}
	d.Subject = fmt.Sprintf("Campaign started: %s", campaignName)
	d.Preheader = "A campaign you follow has started."
	s.send(to, d.Subject, "campaign_started.html", d)
}

func (s *Service) NotifyCampaignEnded(to, name, campaignName, humanID, statsSummary string) {
	d := CampaignCycleData{
		BaseData:     s.base(name),
		CampaignName: campaignName,
		HumanID:      humanID,
		DashboardURL: s.AppURL + "/dashboard",
		StatsSummary: statsSummary,
	}
	d.Subject = fmt.Sprintf("Campaign ended: %s", campaignName)
	d.Preheader = "A campaign you follow has ended."
	s.send(to, d.Subject, "campaign_ended.html", d)
}

func (s *Service) NotifyCycleStarted(to, name, campaignName string, cycleNumber int, start, end string) {
	d := CampaignCycleData{
		BaseData:     s.base(name),
		CampaignName: campaignName,
		CycleNumber:  cycleNumber,
		StartDate:    start,
		EndDate:      end,
		DashboardURL: s.AppURL + "/dashboard",
	}
	d.Subject = fmt.Sprintf("Cycle %d started — %s", cycleNumber, campaignName)
	d.Preheader = "A new cycle has started."
	s.send(to, d.Subject, "cycle_started.html", d)
}

func (s *Service) NotifyCycleMid(to, name, campaignName string, cycleNumber int, end string) {
	d := CampaignCycleData{
		BaseData:     s.base(name),
		CampaignName: campaignName,
		CycleNumber:  cycleNumber,
		EndDate:      end,
		DashboardURL: s.AppURL + "/dashboard",
	}
	d.Subject = fmt.Sprintf("Mid-cycle update — %s (cycle %d)", campaignName, cycleNumber)
	d.Preheader = "You are at the midpoint of the cycle."
	s.send(to, d.Subject, "cycle_mid.html", d)
}

func (s *Service) NotifyCycleEnding(to, name, campaignName string, cycleNumber int, end, window string) {
	d := CampaignCycleData{
		BaseData:     s.base(name),
		CampaignName: campaignName,
		CycleNumber:  cycleNumber,
		EndDate:      end,
		DashboardURL: s.AppURL + "/dashboard",
		StatsSummary: window, // "3 days" or "24 hours"
	}
	d.Subject = fmt.Sprintf("Cycle ending in %s — %s", window, campaignName)
	d.Preheader = fmt.Sprintf("Cycle ends in %s.", window)
	s.send(to, d.Subject, "cycle_ending.html", d)
}

func (s *Service) NotifyCycleEnded(to, name, campaignName string, cycleNumber int, statsSummary string) {
	d := CampaignCycleData{
		BaseData:     s.base(name),
		CampaignName: campaignName,
		CycleNumber:  cycleNumber,
		DashboardURL: s.AppURL + "/dashboard",
		StatsSummary: statsSummary,
	}
	d.Subject = fmt.Sprintf("Cycle %d ended — %s", cycleNumber, campaignName)
	d.Preheader = "A cycle has ended."
	s.send(to, d.Subject, "cycle_ended.html", d)
}

// ─── Manager / assignment ─────────────────────────────────────────────────────

func (s *Service) NotifyManagerAssigned(to, name, campaignName, humanID string) {
	d := ManagerAssignData{
		BaseData:     s.base(name),
		CampaignName: campaignName,
		HumanID:      humanID,
		Action:       "assigned",
		DashboardURL: s.AppURL + "/dashboard",
	}
	d.Subject = fmt.Sprintf("Assigned to campaign: %s", campaignName)
	d.Preheader = "You have been assigned to a campaign."
	s.send(to, d.Subject, "manager_assigned.html", d)
}

func (s *Service) NotifyManagerUnassigned(to, name, campaignName, humanID string) {
	d := ManagerAssignData{
		BaseData:     s.base(name),
		CampaignName: campaignName,
		HumanID:      humanID,
		Action:       "removed",
		DashboardURL: s.AppURL + "/dashboard",
	}
	d.Subject = fmt.Sprintf("Removed from campaign: %s", campaignName)
	d.Preheader = "You have been removed from a campaign."
	s.send(to, d.Subject, "manager_unassigned.html", d)
}

func (s *Service) NotifyTalentAssigned(to, name, campaignName string, cycleNumber, tier int) {
	d := AssignmentData{
		BaseData:     s.base(name),
		CampaignName: campaignName,
		CycleNumber:  cycleNumber,
		Tier:         tier,
		DashboardURL: s.AppURL + "/dashboard",
	}
	d.Subject = fmt.Sprintf("Assigned to %s — cycle %d", campaignName, cycleNumber)
	d.Preheader = "You have been assigned to a cycle."
	s.send(to, d.Subject, "talent_assigned.html", d)
}

// ─── Brand stats ──────────────────────────────────────────────────────────────

func (s *Service) NotifyBrandStats(to, name, brandName, campaignName, humanID string, conversions float64, periodLabel string) {
	d := BrandStatsData{
		BaseData:     s.base(name),
		BrandName:    brandName,
		CampaignName: campaignName,
		HumanID:      humanID,
		Conversions:  conversions,
		DashboardURL: s.AppURL + "/dashboard",
		PeriodLabel:  periodLabel,
	}
	d.Subject = fmt.Sprintf("Campaign stats: %s", campaignName)
	d.Preheader = "Campaign performance update."
	s.send(to, d.Subject, "brand_stats.html", d)
}

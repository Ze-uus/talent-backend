package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Ze-uus/talent-backend/internal/mail"
	"github.com/Ze-uus/talent-backend/internal/store"
)

const (
	tmplCycleMid     = "cycle_mid"
	tmplCycleEnding3 = "cycle_ending_3d"
	tmplCycleEnding1 = "cycle_ending_24h"
	tmplBrandStats   = "brand_stats_daily"
)

// CycleRemindersJob sends mid-cycle, ending-soon, and daily brand stats emails.
// Idempotent via email_dispatches unique keys.
type CycleRemindersJob struct {
	st   store.Store
	mail *mail.Service
	log  *slog.Logger
}

func NewCycleRemindersJob(s store.Store, mailSvc *mail.Service, log *slog.Logger) *CycleRemindersJob {
	return &CycleRemindersJob{st: s, mail: mailSvc, log: log}
}

func (j *CycleRemindersJob) Run() {
	if j.mail == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	now := time.Now().UTC()
	cycles, err := j.st.GetActiveCycles(ctx)
	if err != nil {
		j.log.Error("cycle_reminders: get active cycles failed", "err", err)
		return
	}

	for _, cycle := range cycles {
		campaign, err := j.st.GetCampaignByID(ctx, cycle.Campaign_id)
		if err != nil {
			continue
		}
		j.processCycleReminders(ctx, campaign, cycle, now)
	}

	j.sendBrandDailyDigests(ctx, now)
}

func (j *CycleRemindersJob) processCycleReminders(ctx context.Context, campaign store.Campaign, cycle store.Cycle, now time.Time) {
	if cycle.Start_date.IsZero() || cycle.End_date.IsZero() || !cycle.End_date.After(cycle.Start_date) {
		return
	}

	duration := cycle.End_date.Sub(cycle.Start_date)
	midpoint := cycle.Start_date.Add(duration / 2)
	end := cycle.End_date.Format("2006-01-02")

	if !now.Before(midpoint) {
		j.dispatchToCycleStakeholders(ctx, campaign, cycle, tmplCycleMid, func(to, name string) {
			j.mail.NotifyCycleMid(to, name, campaign.Name, cycle.Cycle_number, end)
		})
	}

	untilEnd := cycle.End_date.Sub(now)
	if untilEnd <= 72*time.Hour && untilEnd > 24*time.Hour {
		j.dispatchToCycleStakeholders(ctx, campaign, cycle, tmplCycleEnding3, func(to, name string) {
			j.mail.NotifyCycleEnding(to, name, campaign.Name, cycle.Cycle_number, end, "3 days")
		})
	}
	if untilEnd <= 24*time.Hour && untilEnd > 0 {
		j.dispatchToCycleStakeholders(ctx, campaign, cycle, tmplCycleEnding1, func(to, name string) {
			j.mail.NotifyCycleEnding(to, name, campaign.Name, cycle.Cycle_number, end, "24 hours")
		})
	}
}

func (j *CycleRemindersJob) dispatchToCycleStakeholders(
	ctx context.Context,
	campaign store.Campaign,
	cycle store.Cycle,
	templateKey string,
	send func(to, name string),
) {
	recipients := j.cycleRecipients(ctx, campaign, cycle)
	for _, r := range recipients {
		ok, err := j.st.TryRecordEmailDispatch(ctx, "cycle", cycle.ID, templateKey, r.email)
		if err != nil {
			j.log.Error("cycle_reminders: record dispatch failed", "err", err, "template", templateKey)
			continue
		}
		if !ok {
			continue
		}
		send(r.email, r.name)
	}
}

type recipient struct {
	email string
	name  string
}

func (j *CycleRemindersJob) cycleRecipients(ctx context.Context, campaign store.Campaign, cycle store.Cycle) []recipient {
	seen := map[string]struct{}{}
	var out []recipient

	add := func(email, name string) {
		if email == "" {
			return
		}
		if _, ok := seen[email]; ok {
			return
		}
		seen[email] = struct{}{}
		out = append(out, recipient{email: email, name: name})
	}

	contacts, _ := j.st.ListBrandContacts(ctx, campaign.Brand_id)
	for _, c := range contacts {
		if c.Token_active {
			add(c.Email, c.First_name+" "+c.Last_name)
		}
	}
	managers, _ := j.st.ListManagersByCampaignID(ctx, campaign.ID)
	for _, m := range managers {
		add(m.Email, m.Full_name)
	}
	assignments, _ := j.st.ListAssignedTalents(ctx, cycle.ID)
	for _, a := range assignments {
		talent, err := j.st.GetTalentByID(ctx, a.Talent_id)
		if err != nil {
			continue
		}
		user, err := j.st.GetUserByID(ctx, talent.User_id)
		if err != nil {
			continue
		}
		add(user.Email, user.Full_name)
	}
	return out
}

func (j *CycleRemindersJob) sendBrandDailyDigests(ctx context.Context, now time.Time) {
	campaigns, err := j.st.ListActiveCampaigns(ctx)
	if err != nil {
		j.log.Error("cycle_reminders: list active campaigns failed", "err", err)
		return
	}
	dayKey := now.Format("2006-01-02")
	period := fmt.Sprintf("daily digest %s", dayKey)

	for _, campaign := range campaigns {
		brand, err := j.st.GetBrandByID(ctx, campaign.Brand_id)
		if err != nil {
			continue
		}
		cycles, _ := j.st.ListCyclesByCampaign(ctx, campaign.ID)
		var total float64
		for _, c := range cycles {
			if c.Status != store.Cycle_active && c.Status != store.Cycle_closed {
				continue
			}
			n, _ := j.st.GetTotalConversions(ctx, c.ID)
			total += n
		}
		contacts, _ := j.st.ListBrandContacts(ctx, campaign.Brand_id)
		for _, c := range contacts {
			if !c.Token_active || c.Email == "" {
				continue
			}
			entityID := campaign.ID + ":" + dayKey
			ok, err := j.st.TryRecordEmailDispatch(ctx, "campaign", entityID, tmplBrandStats, c.Email)
			if err != nil || !ok {
				continue
			}
			j.mail.NotifyBrandStats(c.Email, c.First_name+" "+c.Last_name, brand.Name, campaign.Name, campaign.Human_id, total, period)
		}
	}
}

// ReminderWindow reports which reminder keys apply for a cycle at now.
// Exported for unit tests.
func ReminderWindow(start, end, now time.Time) (mid, ending3d, ending24h bool) {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return false, false, false
	}
	midpoint := start.Add(end.Sub(start) / 2)
	mid = !now.Before(midpoint)
	untilEnd := end.Sub(now)
	ending3d = untilEnd <= 72*time.Hour && untilEnd > 24*time.Hour
	ending24h = untilEnd <= 24*time.Hour && untilEnd > 0
	return
}

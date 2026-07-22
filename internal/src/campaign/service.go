package campaign

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/mail"
	payoutsvc "github.com/Ze-uus/talent-backend/internal/src/payout"
	"github.com/Ze-uus/talent-backend/internal/store"
	ws "github.com/Ze-uus/talent-backend/internal/websocket"
)

type CampaignService struct {
	st              store.Store
	payout          *payoutsvc.PayoutService
	cycle_update_ch chan<- domain.CycleUpdateEvent
	log             *slog.Logger
	mail            *mail.Service
}

func New(s store.Store, p *payoutsvc.PayoutService, cycle_update_ch chan<- domain.CycleUpdateEvent, log *slog.Logger, mailSvc *mail.Service) *CampaignService {
	return &CampaignService{st: s, payout: p, cycle_update_ch: cycle_update_ch, log: log, mail: mailSvc}
}

func (s *CampaignService) emitCycleUpdate(cycle store.Cycle, update_type string) {
	ws.TryPush(s.cycle_update_ch, domain.CycleUpdateEvent{
		Cycle_id:    cycle.ID,
		Campaign_id: cycle.Campaign_id,
		Update_type: update_type,
		Shard: domain.CycleUpdateShard{
			Status:           string(cycle.Status),
			Remaining_budget: cycle.Remaining_budget,
		},
	}, s.log, "cycle_update_ch full, event dropped", "cycle_id", cycle.ID, "update_type", update_type)
}

// ─── Campaign CRUD ────────────────────────────────────────────────────────────

func (s *CampaignService) Create(ctx context.Context, c store.Campaign) (store.Campaign, error) {
	human_id, err := s.st.NextCampaignHumanID(ctx, c.Brand_id)
	if err != nil {
		return store.Campaign{}, err
	}
	c.Human_id = human_id
	c.Status = store.Campaign_draft
	if err := s.st.CreateCampaign(ctx, c); err != nil {
		return store.Campaign{}, err
	}
	return s.st.GetCampaignByHumanID(ctx, human_id)
}

func (s *CampaignService) List(ctx context.Context, filter store.CampaignFilter) ([]store.Campaign, error) {
	return s.st.ListCampaigns(ctx, filter)
}

func (s *CampaignService) Get(ctx context.Context, id string) (store.Campaign, error) {
	return s.st.GetCampaignByID(ctx, id)
}

func (s *CampaignService) Patch(ctx context.Context, id string, patch store.CampaignPatch) error {
	prev, _ := s.st.GetCampaignByID(ctx, id)
	if err := s.st.UpdateCampaign(ctx, id, patch); err != nil {
		return err
	}
	if patch.Status != nil && *patch.Status == string(store.Campaign_active) && prev.Status != store.Campaign_active {
		campaign, err := s.st.GetCampaignByID(ctx, id)
		if err == nil {
			s.notifyCampaignStarted(ctx, campaign)
		}
	}
	return nil
}

func (s *CampaignService) Archive(ctx context.Context, id string) error {
	status := string(store.Campaign_archived)
	if err := s.st.UpdateCampaign(ctx, id, store.CampaignPatch{Status: &status}); err != nil {
		return err
	}
	campaign, err := s.st.GetCampaignByID(ctx, id)
	if err != nil {
		return nil
	}
	stats := s.campaignStatsSummary(ctx, campaign)
	s.notifyCampaignStakeholders(ctx, campaign, func(to, name string) {
		s.mail.NotifyCampaignEnded(to, name, campaign.Name, campaign.Human_id, stats)
	})
	return nil
}

// ─── Campaign manager assignments ─────────────────────────────────────────────

func (s *CampaignService) AssignManager(ctx context.Context, manager_id, campaign_id, assigned_by string) error {
	if err := s.st.AssignManagerToCampaign(ctx, manager_id, campaign_id, assigned_by); err != nil {
		return err
	}
	if s.mail == nil {
		return nil
	}
	manager, err := s.st.GetUserByID(ctx, manager_id)
	if err != nil {
		return nil
	}
	campaign, err := s.st.GetCampaignByID(ctx, campaign_id)
	if err != nil {
		return nil
	}
	s.mail.NotifyManagerAssigned(manager.Email, manager.Full_name, campaign.Name, campaign.Human_id)
	return nil
}

func (s *CampaignService) UnassignManager(ctx context.Context, manager_id, campaign_id string) error {
	manager, _ := s.st.GetUserByID(ctx, manager_id)
	campaign, _ := s.st.GetCampaignByID(ctx, campaign_id)
	if err := s.st.UnassignManagerFromCampaign(ctx, manager_id, campaign_id); err != nil {
		return err
	}
	if s.mail != nil && manager.Email != "" {
		s.mail.NotifyManagerUnassigned(manager.Email, manager.Full_name, campaign.Name, campaign.Human_id)
	}
	return nil
}

// ─── Cycle management ─────────────────────────────────────────────────────────

// CreateCycle runs the waterfall algorithm and persists budget slots.
func (s *CampaignService) CreateCycle(ctx context.Context, c store.Cycle) (store.Cycle, error) {
	campaign, err := s.st.GetCampaignByID(ctx, c.Campaign_id)
	if err != nil {
		return store.Cycle{}, err
	}

	talents, err := s.st.ListAllTalents(ctx)
	if err != nil {
		return store.Cycle{}, err
	}
	cfg := algo.DefaultConfig()
	all_tiers := algo.ValidTiers(cfg)
	talents_by_tier := make(map[int]int)
	for _, t := range talents {
		if t.Status == store.Status_active {
			dt := algo.DemandTier(float64(categoryPDC(t.Category)), campaign.Cycle_length, campaign.Max_cpa, cfg.Tier_increment)
			if dt > 0 {
				talents_by_tier[dt]++
			}
		}
	}

	wf_out, err := algo.RunWaterfall(algo.WaterfallInput{
		Cycle_budget:    c.Cycle_budget,
		Tiers:           all_tiers,
		Talents_by_tier: talents_by_tier,
		Config:          cfg,
	})
	if err != nil {
		return store.Cycle{}, err
	}

	c.Campaign_type = campaign.Campaign_type
	c.Status = store.Cycle_pending
	c.Start_date = time.Now().UTC()
	if c.End_date.IsZero() {
		c.End_date = c.Start_date.Add(time.Duration(campaign.Cycle_length) * 24 * time.Hour)
	}

	if err := s.st.CreateCycle(ctx, c); err != nil {
		return store.Cycle{}, err
	}

	var slots []store.BudgetSlot
	for _, ts := range wf_out.Slots {
		for i := 0; i < ts.Slot_count; i++ {
			slots = append(slots, store.BudgetSlot{
				Cycle_id:   c.ID,
				Tier_value: ts.Tier,
				Slot_index: i,
			})
		}
	}
	if len(slots) > 0 {
		if err := s.st.CreateBudgetSlots(ctx, slots); err != nil {
			return store.Cycle{}, err
		}
	}

	created, err := s.st.GetCycleByID(ctx, c.ID)
	if err != nil {
		return store.Cycle{}, err
	}
	s.emitCycleUpdate(created, "created")
	return created, nil
}

func (s *CampaignService) ListCycles(ctx context.Context, campaign_id string) ([]store.Cycle, error) {
	return s.st.ListCyclesByCampaign(ctx, campaign_id)
}

func (s *CampaignService) GetCycle(ctx context.Context, id string) (store.Cycle, error) {
	return s.st.GetCycleByID(ctx, id)
}

func (s *CampaignService) PatchCycle(ctx context.Context, id string, patch store.CyclePatch) error {
	if err := s.st.UpdateCycle(ctx, id, patch); err != nil {
		return err
	}
	cycle, err := s.st.GetCycleByID(ctx, id)
	if err != nil {
		return err
	}
	s.emitCycleUpdate(cycle, "patched")
	return nil
}

func (s *CampaignService) ActivateCycle(ctx context.Context, id string) error {
	status := string(store.Cycle_active)
	if err := s.st.UpdateCycle(ctx, id, store.CyclePatch{Status: &status}); err != nil {
		return err
	}
	cycle, err := s.st.GetCycleByID(ctx, id)
	if err != nil {
		return err
	}
	s.emitCycleUpdate(cycle, "activated")

	campaign, err := s.st.GetCampaignByID(ctx, cycle.Campaign_id)
	if err == nil {
		if campaign.Status != store.Campaign_active {
			active := string(store.Campaign_active)
			_ = s.st.UpdateCampaign(ctx, campaign.ID, store.CampaignPatch{Status: &active})
			campaign.Status = store.Campaign_active
			s.notifyCampaignStarted(ctx, campaign)
		}
		start := cycle.Start_date.Format("2006-01-02")
		end := cycle.End_date.Format("2006-01-02")
		s.notifyCycleStakeholders(ctx, campaign, cycle, func(to, name string) {
			s.mail.NotifyCycleStarted(to, name, campaign.Name, cycle.Cycle_number, start, end)
		})
	}
	return nil
}

func (s *CampaignService) PauseCycle(ctx context.Context, id string) error {
	status := "paused"
	if err := s.st.UpdateCycle(ctx, id, store.CyclePatch{Status: &status}); err != nil {
		return err
	}
	cycle, err := s.st.GetCycleByID(ctx, id)
	if err != nil {
		return err
	}
	s.emitCycleUpdate(cycle, "paused")
	return nil
}

func (s *CampaignService) CloseCycle(ctx context.Context, id string) error {
	status := string(store.Cycle_closed)
	if err := s.st.UpdateCycle(ctx, id, store.CyclePatch{Status: &status}); err != nil {
		return err
	}
	cycle, err := s.st.GetCycleByID(ctx, id)
	if err != nil {
		return err
	}
	s.emitCycleUpdate(cycle, "closed")

	campaign, err := s.st.GetCampaignByID(ctx, cycle.Campaign_id)
	if err == nil {
		total, _ := s.st.GetTotalConversions(ctx, cycle.ID)
		stats := fmt.Sprintf("Cycle conversions: %.0f", total)
		s.notifyCycleStakeholders(ctx, campaign, cycle, func(to, name string) {
			s.mail.NotifyCycleEnded(to, name, campaign.Name, cycle.Cycle_number, stats)
		})
		s.notifyBrandStats(ctx, campaign, cycle, total, fmt.Sprintf("cycle %d closed", cycle.Cycle_number))
	}
	return nil
}

// FinalisePayouts (Story 24): locks fallback conversions then computes payouts.
func (s *CampaignService) FinalisePayouts(ctx context.Context, cycle_id string) error {
	cycle, err := s.st.GetCycleByID(ctx, cycle_id)
	if err != nil {
		return err
	}
	if cycle.Status != store.Cycle_active {
		return errors.New("cycle_not_active")
	}
	if err := s.st.LockFallbackConversions(ctx, cycle_id); err != nil {
		return err
	}
	if err := s.payout.ComputeAndStoreCyclePayout(ctx, cycle_id); err != nil {
		return err
	}
	updated, err := s.st.GetCycleByID(ctx, cycle_id)
	if err != nil {
		return err
	}
	s.emitCycleUpdate(updated, "payouts_finalised")
	return nil
}

// ─── Mail helpers ─────────────────────────────────────────────────────────────

func (s *CampaignService) notifyCampaignStarted(ctx context.Context, campaign store.Campaign) {
	s.notifyCampaignStakeholders(ctx, campaign, func(to, name string) {
		s.mail.NotifyCampaignStarted(to, name, campaign.Name, campaign.Human_id)
	})
}

func (s *CampaignService) notifyCampaignStakeholders(ctx context.Context, campaign store.Campaign, fn func(to, name string)) {
	if s.mail == nil {
		return
	}
	contacts, _ := s.st.ListBrandContacts(ctx, campaign.Brand_id)
	for _, c := range contacts {
		if c.Email != "" && c.Token_active {
			fn(c.Email, c.First_name+" "+c.Last_name)
		}
	}
	managers, _ := s.st.ListManagersByCampaignID(ctx, campaign.ID)
	for _, m := range managers {
		if m.Email != "" {
			fn(m.Email, m.Full_name)
		}
	}
}

func (s *CampaignService) notifyCycleStakeholders(ctx context.Context, campaign store.Campaign, cycle store.Cycle, fn func(to, name string)) {
	if s.mail == nil {
		return
	}
	s.notifyCampaignStakeholders(ctx, campaign, fn)
	assignments, _ := s.st.ListAssignedTalents(ctx, cycle.ID)
	for _, a := range assignments {
		talent, err := s.st.GetTalentByID(ctx, a.Talent_id)
		if err != nil {
			continue
		}
		user, err := s.st.GetUserByID(ctx, talent.User_id)
		if err != nil || user.Email == "" {
			continue
		}
		fn(user.Email, user.Full_name)
	}
}

func (s *CampaignService) notifyBrandStats(ctx context.Context, campaign store.Campaign, cycle store.Cycle, conversions float64, period string) {
	if s.mail == nil {
		return
	}
	brand, err := s.st.GetBrandByID(ctx, campaign.Brand_id)
	if err != nil {
		return
	}
	contacts, _ := s.st.ListBrandContacts(ctx, campaign.Brand_id)
	for _, c := range contacts {
		if c.Email != "" && c.Token_active {
			s.mail.NotifyBrandStats(c.Email, c.First_name+" "+c.Last_name, brand.Name, campaign.Name, campaign.Human_id, conversions, period)
		}
	}
	_ = cycle
}

func (s *CampaignService) campaignStatsSummary(ctx context.Context, campaign store.Campaign) string {
	cycles, err := s.st.ListCyclesByCampaign(ctx, campaign.ID)
	if err != nil {
		return ""
	}
	var total float64
	for _, c := range cycles {
		n, _ := s.st.GetTotalConversions(ctx, c.ID)
		total += n
	}
	return fmt.Sprintf("Total conversions across cycles: %.0f", total)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func categoryPDC(cat store.Talent_category) int {
	switch cat {
	case store.Category_student:
		return 2
	case store.Category_micro:
		return 5
	case store.Category_community:
		return 15
	}
	return 2
}

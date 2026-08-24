package campaign

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Ze-uus/talent-backend/internal/allocation"
	"github.com/Ze-uus/talent-backend/internal/audit"
	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/mail"
	"github.com/Ze-uus/talent-backend/internal/media"
	"github.com/Ze-uus/talent-backend/internal/response"
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
	auditor         *audit.Recorder
	media           media.Uploader
}

func New(s store.Store, p *payoutsvc.PayoutService, cycle_update_ch chan<- domain.CycleUpdateEvent, log *slog.Logger, mailSvc *mail.Service, auditor *audit.Recorder, uploader media.Uploader) *CampaignService {
	if uploader == nil {
		uploader = media.DisabledUploader{}
	}
	return &CampaignService{st: s, payout: p, cycle_update_ch: cycle_update_ch, log: log, mail: mailSvc, auditor: auditor, media: uploader}
}

const MaxCampaignImagesPerUpload = 10

type ContentImageFile struct {
	Body        io.Reader
	ContentType string
	Size        int64
}

func (s *CampaignService) UploadContentImages(ctx context.Context, campaignID string, files []ContentImageFile) ([]string, error) {
	if _, err := s.st.GetCampaignByID(ctx, campaignID); err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, response.Validation("content_images_required")
	}
	if len(files) > MaxCampaignImagesPerUpload {
		return nil, response.Validation("too_many_content_images")
	}

	urls := make([]string, 0, len(files))
	for _, file := range files {
		if file.Size <= 0 || file.Size > media.MaxImageBytes {
			return nil, media.ErrTooLarge
		}
		if err := media.ValidateContentType(file.ContentType); err != nil {
			return nil, err
		}
		fileName, err := media.NewFileName(uuid.NewString(), file.ContentType)
		if err != nil {
			return nil, err
		}
		result, err := s.media.Upload(ctx, media.UploadInput{
			Folder: media.FolderCampaignContent(campaignID), FileName: fileName,
			ContentType: file.ContentType, Body: file.Body,
		})
		if err != nil {
			return nil, err
		}
		urls = append(urls, result.URL)
	}
	return urls, nil
}

func (s *CampaignService) emitCycleUpdate(ctx context.Context, cycle store.Cycle, update_type string) {
	ws.TryPush(s.cycle_update_ch, domain.CycleUpdateEvent{
		Cycle_id:    cycle.ID,
		Campaign_id: cycle.Campaign_id,
		Update_type: update_type,
		Shard: domain.CycleUpdateShard{
			Status:           string(cycle.Status),
			Remaining_budget: cycle.Remaining_budget,
		},
	}, s.log, "cycle_update_ch full, event dropped", "cycle_id", cycle.ID, "update_type", update_type)
	if s.auditor != nil {
		_ = s.auditor.Record(ctx, audit.Entry{
			Action:      "cycle_" + update_type,
			Entity_type: "cycle",
			Entity_id:   cycle.ID,
			After:       cycle,
		})
	}
}

// ─── Campaign CRUD ────────────────────────────────────────────────────────────

func (s *CampaignService) Create(ctx context.Context, c store.Campaign) (store.Campaign, error) {
	if err := validateCampaign(c); err != nil {
		return store.Campaign{}, err
	}
	content, err := store.NormalizeAndValidateContent(c.Content)
	if err != nil {
		return store.Campaign{}, err
	}
	c.Content = content
	human_id, err := s.st.NextCampaignHumanID(ctx, c.Brand_id)
	if err != nil {
		return store.Campaign{}, err
	}
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	c.Human_id = human_id
	c.Status = store.Campaign_draft
	if c.Remaining_budget == 0 {
		c.Remaining_budget = c.Total_budget
	}
	if c.Market_cap == "" {
		c.Market_cap = "M"
	}
	if c.Urgency_level == "" {
		c.Urgency_level = store.Urgency_normal
	}
	if c.Start_date.IsZero() {
		c.Start_date = time.Now().UTC()
	}
	if c.End_date.IsZero() {
		days := c.Cycle_length
		if days <= 0 {
			days = 7
		}
		c.End_date = c.Start_date.Add(time.Duration(days) * 24 * time.Hour)
	}
	if err := s.st.CreateCampaign(ctx, c); err != nil {
		return store.Campaign{}, err
	}
	return s.st.GetCampaignByHumanID(ctx, human_id)
}

func (s *CampaignService) List(ctx context.Context, filter store.CampaignFilter) ([]store.Campaign, error) {
	return s.st.ListCampaigns(ctx, filter)
}

func (s *CampaignService) Get(ctx context.Context, id string) (CampaignDetail, error) {
	c, err := s.st.GetCampaignByID(ctx, id)
	if err != nil {
		return CampaignDetail{}, err
	}
	managers, err := s.st.ListManagersByCampaignID(ctx, id)
	if err != nil {
		return CampaignDetail{}, err
	}
	views := make([]CampaignManagerView, 0, len(managers))
	for _, m := range managers {
		views = append(views, CampaignManagerView{
			ID:        m.ID,
			Email:     m.Email,
			Full_name: m.Full_name,
			Role:      m.Role,
			Status:    m.Status,
			Active:    m.Active,
		})
	}
	return CampaignDetail{Campaign: c, Campaign_managers: views}, nil
}

func (s *CampaignService) Patch(ctx context.Context, id string, patch store.CampaignPatch) error {
	if err := validateCampaignPatch(patch); err != nil {
		return err
	}
	if patch.Content != nil {
		content, err := store.NormalizeAndValidateContent(*patch.Content)
		if err != nil {
			return err
		}
		patch.Content = &content
	}
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
	if c.Content_override != nil {
		content, err := store.NormalizeAndValidateContent(*c.Content_override)
		if err != nil {
			return store.Cycle{}, err
		}
		c.Content_override = &content
	}
	campaign, err := s.st.GetCampaignByID(ctx, c.Campaign_id)
	if err != nil {
		return store.Cycle{}, err
	}

	existing, err := s.st.ListCyclesByCampaign(ctx, c.Campaign_id)
	if err != nil {
		return store.Cycle{}, err
	}
	campaign.Remaining_budget = remainingCycleBudget(campaign, existing, "")
	if err := validateCycleBudget(campaign, c.Cycle_budget); err != nil {
		return store.Cycle{}, err
	}

	talents, err := s.st.ListAllTalents(ctx)
	if err != nil {
		return store.Cycle{}, err
	}

	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	c.Cycle_number = len(existing) + 1
	if c.Human_id == "" {
		c.Human_id = fmt.Sprintf("%s-C%d", campaign.Human_id, c.Cycle_number)
	}
	c.Campaign_type = campaign.Campaign_type
	c.Status = store.Cycle_pending
	if c.Remaining_budget == 0 {
		c.Remaining_budget = c.Cycle_budget
	}
	if c.Z_factor == 0 {
		c.Z_factor = zFactorForUrgency(campaign.Urgency_level)
	}
	c.Start_date = time.Now().UTC()
	if c.End_date.IsZero() {
		c.End_date = c.Start_date.Add(time.Duration(campaign.Cycle_length) * 24 * time.Hour)
	}
	if err := allocation.ValidateCycle(c, campaign); err != nil {
		return store.Cycle{}, err
	}

	if err := s.st.CreateCycle(ctx, c); err != nil {
		return store.Cycle{}, err
	}

	slots, err := allocation.BuildBudgetSlots(c, campaign, talents)
	if err != nil {
		return store.Cycle{}, err
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
	s.emitCycleUpdate(ctx, created, "created")
	return created, nil
}

func zFactorForUrgency(level store.Urgency_level) float64 {
	switch level {
	case store.Urgency_low:
		return 0.8
	case store.Urgency_high:
		return 1.2
	default:
		return 1.0
	}
}

func (s *CampaignService) ListCycles(ctx context.Context, campaign_id string) ([]store.Cycle, error) {
	return s.st.ListCyclesByCampaign(ctx, campaign_id)
}

func (s *CampaignService) GetCycle(ctx context.Context, id string) (store.Cycle, error) {
	return s.st.GetCycleByID(ctx, id)
}

func (s *CampaignService) PatchCycle(ctx context.Context, id string, patch store.CyclePatch) error {
	if patch.Status != nil && !validCycleStatus(*patch.Status) {
		return response.Validation("invalid_cycle_status")
	}
	if patch.Z_factor != nil && (!positiveFinite(*patch.Z_factor) || *patch.Z_factor > 99.99) {
		return response.Validation("invalid_z_factor")
	}
	if patch.Cycle_budget != nil {
		cycle, err := s.st.GetCycleByID(ctx, id)
		if err != nil {
			return err
		}
		campaign, err := s.st.GetCampaignByID(ctx, cycle.Campaign_id)
		if err != nil {
			return err
		}
		cycles, err := s.st.ListCyclesByCampaign(ctx, cycle.Campaign_id)
		if err != nil {
			return err
		}
		campaign.Remaining_budget = remainingCycleBudget(campaign, cycles, cycle.ID)
		if err := validateCycleBudget(campaign, *patch.Cycle_budget); err != nil {
			return err
		}
	}
	if patch.Content_override != nil && *patch.Content_override != nil {
		content, err := store.NormalizeAndValidateContent(*patch.Content_override)
		if err != nil {
			return err
		}
		patch.Content_override = &content
	}
	if err := s.st.UpdateCycle(ctx, id, patch); err != nil {
		return err
	}
	cycle, err := s.st.GetCycleByID(ctx, id)
	if err != nil {
		return err
	}
	s.emitCycleUpdate(ctx, cycle, "patched")
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
	s.emitCycleUpdate(ctx, cycle, "activated")

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
	s.emitCycleUpdate(ctx, cycle, "paused")
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
	s.emitCycleUpdate(ctx, cycle, "closed")

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
		return response.Validation("cycle_not_active")
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
	s.emitCycleUpdate(ctx, updated, "payouts_finalised")
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

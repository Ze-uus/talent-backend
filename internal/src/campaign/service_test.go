package campaign_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/media"
	"github.com/Ze-uus/talent-backend/internal/src/campaign"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type mockStore struct {
	cycle                store.Cycle
	campaign             store.Campaign
	managers             []store.User
	cycles               []store.Cycle
	nextHumanIDArg       string
	createdCampaign      store.Campaign
	createdCycle         store.Cycle
	createCampaignCalled bool
	createCycleCalled    bool
}

func (m *mockStore) UpdateCycle(_ context.Context, id string, patch store.CyclePatch) error {
	if m.cycle.ID != id {
		return errors.New("not_found")
	}
	if patch.Status != nil {
		m.cycle.Status = store.Cycle_status(*patch.Status)
	}
	if patch.Content_override != nil {
		if *patch.Content_override == nil {
			m.cycle.Content_override = nil
		} else {
			content := *patch.Content_override
			m.cycle.Content_override = &content
		}
	}
	return nil
}
func (m *mockStore) GetCycleByID(_ context.Context, id string) (store.Cycle, error) {
	if m.createdCycle.ID == id {
		return m.createdCycle, nil
	}
	if m.cycle.ID == id {
		return m.cycle, nil
	}
	return store.Cycle{}, errors.New("not_found")
}

// stub remaining Store methods
func (m *mockStore) Ping(_ context.Context) error                     { return nil }
func (m *mockStore) CreateUser(_ context.Context, _ store.User) error { return nil }
func (m *mockStore) GetUserByID(_ context.Context, _ string) (store.User, error) {
	return store.User{}, nil
}
func (m *mockStore) GetUserByEmail(_ context.Context, _ string) (store.User, error) {
	return store.User{}, nil
}
func (m *mockStore) GetUserByGoogleID(_ context.Context, _ string) (store.User, error) {
	return store.User{}, nil
}
func (m *mockStore) GetUserByInviteToken(_ context.Context, _ string) (store.User, error) {
	return store.User{}, nil
}
func (m *mockStore) UpdateUser(_ context.Context, _ string, _ store.UserPatch) error { return nil }
func (m *mockStore) ListUsers(_ context.Context, _ store.UserFilter) ([]store.User, error) {
	return nil, nil
}
func (m *mockStore) CreateSession(_ context.Context, _ store.Session) error { return nil }
func (m *mockStore) GetSession(_ context.Context, _ string) (store.Session, error) {
	return store.Session{}, nil
}
func (m *mockStore) TouchSession(_ context.Context, _ string, _ time.Time) error { return nil }
func (m *mockStore) InvalidateSession(_ context.Context, _ string) error         { return nil }
func (m *mockStore) InvalidateAllUserSessions(_ context.Context, _ string) error { return nil }
func (m *mockStore) ListSessionsByUser(_ context.Context, _ string) ([]store.Session, error) {
	return nil, nil
}
func (m *mockStore) CreateBrand(_ context.Context, _ store.Brand) error { return nil }
func (m *mockStore) GetBrandByID(_ context.Context, _ string) (store.Brand, error) {
	return store.Brand{}, nil
}
func (m *mockStore) GetBrandByShortcode(_ context.Context, _ string) (store.Brand, error) {
	return store.Brand{}, nil
}
func (m *mockStore) ListBrands(_ context.Context, _ store.BrandFilter) ([]store.Brand, error) {
	return nil, nil
}
func (m *mockStore) UpdateBrand(_ context.Context, _ string, _ store.BrandPatch) error { return nil }
func (m *mockStore) ShortcodeExists(_ context.Context, _ string) (bool, error)         { return false, nil }
func (m *mockStore) CreateBrandContact(_ context.Context, _ store.BrandContact) error  { return nil }
func (m *mockStore) GetBrandContactByID(_ context.Context, _ string) (store.BrandContact, error) {
	return store.BrandContact{}, nil
}
func (m *mockStore) GetBrandContactByViewerToken(_ context.Context, _ string) (store.BrandContact, error) {
	return store.BrandContact{}, nil
}
func (m *mockStore) ListBrandContacts(_ context.Context, _ string) ([]store.BrandContact, error) {
	return nil, nil
}
func (m *mockStore) UpdateBrandContact(_ context.Context, _ string, _ store.BrandContactPatch) error {
	return nil
}
func (m *mockStore) DeactivateBrandContact(_ context.Context, _ string) error { return nil }
func (m *mockStore) RegenerateBrandContactPassword(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockStore) ValidateBrandContactAccess(_ context.Context, _, _ string) (store.BrandContact, error) {
	return store.BrandContact{}, nil
}
func (m *mockStore) CreateTalent(_ context.Context, _ store.Talent) error { return nil }
func (m *mockStore) GetTalentByID(_ context.Context, _ string) (store.Talent, error) {
	return store.Talent{}, nil
}
func (m *mockStore) GetTalentByUserID(_ context.Context, _ string) (store.Talent, error) {
	return store.Talent{}, nil
}
func (m *mockStore) ListTalents(_ context.Context, _ store.TalentFilter) ([]store.Talent, error) {
	return nil, nil
}
func (m *mockStore) UpdateTalent(_ context.Context, _ string, _ store.TalentPatch) error { return nil }
func (m *mockStore) CreateCampaign(_ context.Context, c store.Campaign) error {
	m.createCampaignCalled = true
	m.createdCampaign = c
	return nil
}
func (m *mockStore) GetCampaignByID(_ context.Context, id string) (store.Campaign, error) {
	if m.campaign.ID == id {
		return m.campaign, nil
	}
	if m.createdCampaign.ID == id {
		return m.createdCampaign, nil
	}
	return store.Campaign{}, errors.New("not_found")
}
func (m *mockStore) GetCampaignByHumanID(_ context.Context, human_id string) (store.Campaign, error) {
	c := m.createdCampaign
	c.Human_id = human_id
	return c, nil
}
func (m *mockStore) ListCampaigns(_ context.Context, _ store.CampaignFilter) ([]store.Campaign, error) {
	return nil, nil
}
func (m *mockStore) ListActiveCampaigns(_ context.Context) ([]store.Campaign, error) { return nil, nil }
func (m *mockStore) UpdateCampaign(_ context.Context, id string, patch store.CampaignPatch) error {
	if m.campaign.ID == id && patch.Content != nil {
		m.campaign.Content = *patch.Content
	}
	return nil
}
func (m *mockStore) DecrementRemainingBudget(_ context.Context, _ string, _ float64) error {
	return nil
}
func (m *mockStore) NextCampaignHumanID(_ context.Context, brand_id string) (string, error) {
	m.nextHumanIDArg = brand_id
	return "CRD-26-01", nil
}
func (m *mockStore) GetCampaignsByManagerID(_ context.Context, _ string) ([]store.Campaign, error) {
	return nil, nil
}
func (m *mockStore) AssignManagerToCampaign(_ context.Context, _, _, _ string) error  { return nil }
func (m *mockStore) UnassignManagerFromCampaign(_ context.Context, _, _ string) error { return nil }
func (m *mockStore) ListManagersByCampaignID(_ context.Context, campaign_id string) ([]store.User, error) {
	if m.campaign.ID != "" && campaign_id != m.campaign.ID {
		return nil, nil
	}
	return m.managers, nil
}
func (m *mockStore) TryRecordEmailDispatch(_ context.Context, _, _, _, _ string) (bool, error) {
	return true, nil
}
func (m *mockStore) EmailDispatchExists(_ context.Context, _, _, _, _ string) (bool, error) {
	return false, nil
}

func (m *mockStore) CreateCycle(_ context.Context, c store.Cycle) error {
	m.createCycleCalled = true
	m.createdCycle = c
	return nil
}
func (m *mockStore) ListCyclesByCampaign(_ context.Context, _ string) ([]store.Cycle, error) {
	return m.cycles, nil
}
func (m *mockStore) GetActiveCycles(_ context.Context) ([]store.Cycle, error)        { return nil, nil }
func (m *mockStore) CloseCycle(_ context.Context, _ string, _ float64) error         { return nil }
func (m *mockStore) CreateBudgetSlots(_ context.Context, _ []store.BudgetSlot) error { return nil }
func (m *mockStore) ListSlotsByCycle(_ context.Context, _ string) ([]store.BudgetSlot, error) {
	return nil, nil
}
func (m *mockStore) AssignSlot(_ context.Context, _, _ string) error { return nil }
func (m *mockStore) CreateAssignment(_ context.Context, _ store.TalentAssignment) error {
	return nil
}
func (m *mockStore) GetAssignment(_ context.Context, _, _ string) (store.TalentAssignment, error) {
	return store.TalentAssignment{}, nil
}
func (m *mockStore) ListAssignedTalents(_ context.Context, _ string) ([]store.TalentAssignment, error) {
	return nil, nil
}
func (m *mockStore) ListAssignmentsByTalent(_ context.Context, _ string) ([]store.TalentAssignment, error) {
	return nil, nil
}
func (m *mockStore) UpdateAssignment(_ context.Context, _, _ string, _ store.AssignmentPatch) error {
	return nil
}
func (m *mockStore) CreateTrackingLink(_ context.Context, _ store.TrackingLink) error { return nil }
func (m *mockStore) GetTrackingLinkByToken(_ context.Context, _ string) (store.TrackingLink, error) {
	return store.TrackingLink{}, nil
}
func (m *mockStore) ListTrackingLinksByCycle(_ context.Context, _ string) ([]store.TrackingLink, error) {
	return nil, nil
}
func (m *mockStore) LogConversionEvent(_ context.Context, _ store.ConversionEvent) error { return nil }
func (m *mockStore) GetDailyConversionCount(_ context.Context, _, _ string, _ time.Time) (float64, error) {
	return 0, nil
}
func (m *mockStore) GetTotalConversions(_ context.Context, _ string) (float64, error) { return 0, nil }
func (m *mockStore) GetTalentConversions(_ context.Context, _, _ string) (float64, error) {
	return 0, nil
}
func (m *mockStore) FlagFallbackConversions(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (m *mockStore) LockFallbackConversions(_ context.Context, _ string) error { return nil }
func (m *mockStore) GetCycleState(_ context.Context, _, _ string) (store.CycleState, error) {
	return store.CycleState{}, nil
}
func (m *mockStore) UpsertCycleState(_ context.Context, _ store.CycleState) error { return nil }
func (m *mockStore) GetDailyOutputs(_ context.Context, _, _ string) ([]float64, error) {
	return nil, nil
}
func (m *mockStore) GetTalentBaseline(_ context.Context, _ string) (store.TalentBaseline, error) {
	return store.TalentBaseline{}, nil
}
func (m *mockStore) UpsertTalentBaseline(_ context.Context, _ store.TalentBaseline) error { return nil }
func (m *mockStore) GetCategoryBaseline(_ context.Context, _ string) (algo.CategoryBaseline, error) {
	return algo.CategoryBaseline{}, nil
}
func (m *mockStore) GetTalentTodayOutput(_ context.Context, _ string, _ time.Time) (float64, error) {
	return 0, nil
}
func (m *mockStore) GetTalentOutputWindow(_ context.Context, _ string, _ int) ([]float64, error) {
	return nil, nil
}
func (m *mockStore) CreatePayoutRecord(_ context.Context, _ store.PayoutRecord) error { return nil }
func (m *mockStore) GetPayoutRecord(_ context.Context, _, _ string) (store.PayoutRecord, error) {
	return store.PayoutRecord{}, nil
}
func (m *mockStore) ListPayoutsByCycle(_ context.Context, _ string) ([]store.PayoutRecord, error) {
	return nil, nil
}
func (m *mockStore) ListPayoutsByTalent(_ context.Context, _ string) ([]store.PayoutRecord, error) {
	return nil, nil
}
func (m *mockStore) UpdatePayoutRecord(_ context.Context, _ string, _ store.PayoutPatch) error {
	return nil
}
func (m *mockStore) CreateViewer(_ context.Context, _ store.CampaignViewer) error { return nil }
func (m *mockStore) GetViewerByToken(_ context.Context, _ string) (store.CampaignViewer, error) {
	return store.CampaignViewer{}, nil
}
func (m *mockStore) ListViewersByCampaign(_ context.Context, _ string) ([]store.CampaignViewer, error) {
	return nil, nil
}
func (m *mockStore) UpdateViewer(_ context.Context, _ string, _ store.ViewerPatch) error { return nil }
func (m *mockStore) AddViewerPassword(_ context.Context, _ store.ViewerPassword) error   { return nil }
func (m *mockStore) ListViewerPasswords(_ context.Context, _ string) ([]store.ViewerPassword, error) {
	return nil, nil
}
func (m *mockStore) DeactivateViewerPassword(_ context.Context, _ string) error { return nil }
func (m *mockStore) ValidateViewerPassword(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockStore) ListAuditLog(_ context.Context, _, _ string) ([]store.AuditLog, error) {
	return nil, nil
}
func (m *mockStore) ListAllTalents(_ context.Context) ([]store.Talent, error) { return nil, nil }
func (m *mockStore) WriteAuditLog(_ context.Context, _ store.AuditLog) error  { return nil }

func (m *mockStore) GetAuditLogByID(_ context.Context, _ string) (store.AuditLog, error) {
	return store.AuditLog{}, nil
}
func (m *mockStore) ListAuditLogFiltered(_ context.Context, _ store.AuditFilter) ([]store.AuditLog, error) {
	return nil, nil
}
func (m *mockStore) AppendAuditLog(_ context.Context, e store.AuditLog) (store.AuditLog, error) {
	return e, nil
}
func (m *mockStore) GetAuditChainTip(_ context.Context) (string, int64, error) {
	return "0000000000000000000000000000000000000000000000000000000000000000", 0, nil
}

func TestCloseCycle_EmitsCycleUpdate(t *testing.T) {
	ch := make(chan domain.CycleUpdateEvent, 1)
	ms := &mockStore{cycle: store.Cycle{
		ID:               "cycle-1",
		Campaign_id:      "camp-1",
		Status:           store.Cycle_active,
		Remaining_budget: 1200,
	}}
	svc := campaign.New(ms, nil, ch, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, nil)

	if err := svc.CloseCycle(context.Background(), "cycle-1"); err != nil {
		t.Fatalf("CloseCycle error: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Update_type != "closed" {
			t.Fatalf("expected closed, got %q", ev.Update_type)
		}
		if ev.Shard.Status != "closed" {
			t.Fatalf("expected shard status closed, got %q", ev.Shard.Status)
		}
	default:
		t.Fatal("expected cycle_update event")
	}
}

func TestCreate_LooksUpBrandByUUID(t *testing.T) {
	brandID := "28f52e45-49d5-4585-b9ff-95c8aaf0699c"
	ms := &mockStore{}
	svc := campaign.New(ms, nil, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, nil)

	created, err := svc.Create(context.Background(), store.Campaign{
		Brand_id:      brandID,
		Name:          "New campaign",
		Campaign_type: store.Type_direct_traffic,
		Total_budget:  5_000_000,
		Target_cpa:    1_200_000,
		Max_cpa:       5_000_000,
		Audience:      "all,state:Enugu,zone:SE",
		Cycle_length:  7,
		Content: []store.ContentItem{{
			ID: "hero", Images: []string{"https://cdn.example.com/hero.png"},
			Links: []store.ContentLink{},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ms.nextHumanIDArg != brandID {
		t.Fatalf("NextCampaignHumanID got %q, want brand UUID", ms.nextHumanIDArg)
	}
	if !ms.createCampaignCalled {
		t.Fatal("expected CreateCampaign")
	}
	if created.Human_id != "CRD-26-01" {
		t.Fatalf("human_id=%q", created.Human_id)
	}
	if ms.createdCampaign.ID == "" {
		t.Fatal("expected campaign UUID")
	}
	if ms.createdCampaign.Remaining_budget != 5_000_000 {
		t.Fatalf("remaining_budget=%v", ms.createdCampaign.Remaining_budget)
	}
	if ms.createdCampaign.Market_cap != "M" || ms.createdCampaign.Urgency_level != store.Urgency_normal {
		t.Fatalf("defaults market=%q urgency=%q", ms.createdCampaign.Market_cap, ms.createdCampaign.Urgency_level)
	}
	if ms.createdCampaign.Start_date.IsZero() || ms.createdCampaign.End_date.IsZero() {
		t.Fatal("expected start/end dates set")
	}
	if len(ms.createdCampaign.Content) != 1 || ms.createdCampaign.Content[0].ID != "hero" {
		t.Fatalf("content was not persisted: %+v", ms.createdCampaign.Content)
	}
}

func TestGet_IncludesCampaignManagers(t *testing.T) {
	ms := &mockStore{
		campaign: store.Campaign{
			ID:   "camp-1",
			Name: "New campaign",
		},
		managers: []store.User{
			{ID: "mgr-1", Email: "mgr@scaloo.com", Full_name: "Ada Manager", Role: store.Role_campaign_manager, Status: store.User_status_active, Active: true},
		},
	}
	svc := campaign.New(ms, nil, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, nil)
	detail, err := svc.Get(context.Background(), "camp-1")
	if err != nil {
		t.Fatal(err)
	}
	if detail.ID != "camp-1" || detail.Name != "New campaign" {
		t.Fatalf("campaign fields: %+v", detail.Campaign)
	}
	if len(detail.Campaign_managers) != 1 {
		t.Fatalf("managers=%d", len(detail.Campaign_managers))
	}
	if detail.Campaign_managers[0].ID != "mgr-1" || detail.Campaign_managers[0].Email != "mgr@scaloo.com" {
		t.Fatalf("manager=%+v", detail.Campaign_managers[0])
	}
}

func TestCreateCycle_SetsIDAndNumber(t *testing.T) {
	ms := &mockStore{
		campaign: store.Campaign{
			ID:               "camp-1",
			Human_id:         "NEW-26-01",
			Campaign_type:    store.Type_direct_traffic,
			Cycle_length:     7,
			Target_cpa:       150000,
			Max_cpa:          300000,
			Remaining_budget: 300000,
			Urgency_level:    store.Urgency_normal,
		},
		cycles: []store.Cycle{},
	}
	svc := campaign.New(ms, nil, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, nil)
	override := []store.ContentItem{{ID: "cycle-hero", Images: []string{}, Links: []store.ContentLink{}}}
	created, err := svc.CreateCycle(context.Background(), store.Cycle{
		Campaign_id:      "camp-1",
		Cycle_budget:     300000,
		Cycle_objective:  "traffic",
		Content_override: &override,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ms.createCycleCalled {
		t.Fatal("expected CreateCycle")
	}
	if created.ID == "" {
		t.Fatal("expected cycle UUID")
	}
	if created.Cycle_number != 1 {
		t.Fatalf("cycle_number=%d", created.Cycle_number)
	}
	if created.Remaining_budget != 300000 {
		t.Fatalf("remaining_budget=%v", created.Remaining_budget)
	}
	if created.Human_id != "NEW-26-01-C1" {
		t.Fatalf("human_id=%q", created.Human_id)
	}
	if created.Z_factor != 1.0 {
		t.Fatalf("z_factor=%v", created.Z_factor)
	}
	if created.Content_override == nil || len(*created.Content_override) != 1 {
		t.Fatalf("expected cycle content override, got %+v", created.Content_override)
	}
}

func TestPatchCycleClearsContentOverride(t *testing.T) {
	override := []store.ContentItem{{ID: "override"}}
	ms := &mockStore{cycle: store.Cycle{ID: "cycle-1", Content_override: &override}}
	svc := campaign.New(ms, nil, nil, slog.Default(), nil, nil, nil)
	var inherit []store.ContentItem

	if err := svc.PatchCycle(context.Background(), "cycle-1", store.CyclePatch{Content_override: &inherit}); err != nil {
		t.Fatal(err)
	}
	if ms.cycle.Content_override != nil {
		t.Fatalf("expected override to be cleared, got %+v", ms.cycle.Content_override)
	}
}

func TestPauseCycleUsesSupportedPausedStatus(t *testing.T) {
	ms := &mockStore{cycle: store.Cycle{ID: "cycle-1", Status: store.Cycle_active}}
	svc := campaign.New(ms, nil, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, nil)
	if err := svc.PauseCycle(context.Background(), "cycle-1"); err != nil {
		t.Fatal(err)
	}
	if ms.cycle.Status != store.Cycle_paused {
		t.Fatalf("status=%q", ms.cycle.Status)
	}
}

func TestUploadContentImagesPreservesOrder(t *testing.T) {
	ms := &mockStore{campaign: store.Campaign{ID: "camp-1"}}
	uploader := &media.RecordingUploader{}
	svc := campaign.New(ms, nil, nil, slog.Default(), nil, nil, uploader)

	files := []campaign.ContentImageFile{
		{Body: bytes.NewReader([]byte("one")), ContentType: "image/png", Size: 3},
		{Body: bytes.NewReader([]byte("two")), ContentType: "image/jpeg", Size: 3},
	}
	urls, err := svc.UploadContentImages(context.Background(), "camp-1", files)
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 2 || !strings.HasSuffix(urls[0], ".png") || !strings.HasSuffix(urls[1], ".jpg") {
		t.Fatalf("unexpected URLs: %+v", urls)
	}
	if len(uploader.Calls) != 2 || uploader.Calls[0].Folder != "/scaloo/campaigns/camp-1/content" {
		t.Fatalf("unexpected upload calls: %+v", uploader.Calls)
	}
}

func TestUploadContentImagesReturnsUploaderFailure(t *testing.T) {
	ms := &mockStore{campaign: store.Campaign{ID: "camp-1"}}
	uploader := &media.RecordingUploader{Err: media.ErrNotConfigured}
	svc := campaign.New(ms, nil, nil, slog.Default(), nil, nil, uploader)

	_, err := svc.UploadContentImages(context.Background(), "camp-1", []campaign.ContentImageFile{{
		Body: bytes.NewReader([]byte("one")), ContentType: "image/png", Size: 3,
	}})
	if !errors.Is(err, media.ErrNotConfigured) {
		t.Fatalf("expected uploader failure, got %v", err)
	}
}

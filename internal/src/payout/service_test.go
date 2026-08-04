package payout_test

import (
	"context"
	"testing"
	"time"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/src/payout"
	"github.com/Ze-uus/talent-backend/internal/store"
)

// ─── Mock store ───────────────────────────────────────────────────────────────

type mockStore struct {
	cycle        store.Cycle
	campaign     store.Campaign
	assignments  []store.TalentAssignment
	conversions  map[string]float64 // talent_id → count
	payout_recs  []store.PayoutRecord
	audit_log    []store.AuditLog
}

func newMock(cycle store.Cycle, assignments []store.TalentAssignment, conversions map[string]float64) *mockStore {
	return &mockStore{cycle: cycle, assignments: assignments, conversions: conversions}
}

func (m *mockStore) GetCycleByID(_ context.Context, _ string) (store.Cycle, error) { return m.cycle, nil }
func (m *mockStore) ListAssignedTalents(_ context.Context, _ string) ([]store.TalentAssignment, error) { return m.assignments, nil }
func (m *mockStore) ListAssignmentsByTalent(_ context.Context, _ string) ([]store.TalentAssignment, error) { return m.assignments, nil }
func (m *mockStore) GetTalentConversions(_ context.Context, talent_id, _ string) (float64, error) {
	return m.conversions[talent_id], nil
}
func (m *mockStore) CreatePayoutRecord(_ context.Context, p store.PayoutRecord) error {
	m.payout_recs = append(m.payout_recs, p)
	return nil
}
func (m *mockStore) GetPayoutRecord(_ context.Context, talent_id, _ string) (store.PayoutRecord, error) {
	for _, r := range m.payout_recs {
		if r.Talent_id == talent_id {
			return r, nil
		}
	}
	return store.PayoutRecord{}, nil
}
func (m *mockStore) UpdatePayoutRecord(_ context.Context, _ string, _ store.PayoutPatch) error { return nil }
func (m *mockStore) WriteAuditLog(_ context.Context, e store.AuditLog) error {
	m.audit_log = append(m.audit_log, e)
	return nil
}

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

// stub remaining Store interface methods
func (m *mockStore) Ping(_ context.Context) error                                        { return nil }
func (m *mockStore) CreateUser(_ context.Context, _ store.User) error                    { return nil }
func (m *mockStore) GetUserByID(_ context.Context, _ string) (store.User, error)         { return store.User{}, nil }
func (m *mockStore) GetUserByEmail(_ context.Context, _ string) (store.User, error)      { return store.User{}, nil }
func (m *mockStore) GetUserByGoogleID(_ context.Context, _ string) (store.User, error)   { return store.User{}, nil }
func (m *mockStore) GetUserByInviteToken(_ context.Context, _ string) (store.User, error) { return store.User{}, nil }
func (m *mockStore) UpdateUser(_ context.Context, _ string, _ store.UserPatch) error     { return nil }
func (m *mockStore) ListUsers(_ context.Context, _ store.UserFilter) ([]store.User, error) { return nil, nil }
func (m *mockStore) CreateSession(_ context.Context, _ store.Session) error               { return nil }
func (m *mockStore) GetSession(_ context.Context, _ string) (store.Session, error)        { return store.Session{}, nil }
func (m *mockStore) TouchSession(_ context.Context, _ string, _ time.Time) error          { return nil }
func (m *mockStore) InvalidateSession(_ context.Context, _ string) error                  { return nil }
func (m *mockStore) InvalidateAllUserSessions(_ context.Context, _ string) error          { return nil }
func (m *mockStore) ListSessionsByUser(_ context.Context, _ string) ([]store.Session, error) { return nil, nil }
func (m *mockStore) CreateBrand(_ context.Context, _ store.Brand) error                   { return nil }
func (m *mockStore) GetBrandByID(_ context.Context, _ string) (store.Brand, error)        { return store.Brand{}, nil }
func (m *mockStore) GetBrandByShortcode(_ context.Context, _ string) (store.Brand, error) { return store.Brand{}, nil }
func (m *mockStore) ListBrands(_ context.Context, _ store.BrandFilter) ([]store.Brand, error) { return nil, nil }
func (m *mockStore) UpdateBrand(_ context.Context, _ string, _ store.BrandPatch) error    { return nil }
func (m *mockStore) ShortcodeExists(_ context.Context, _ string) (bool, error)            { return false, nil }
func (m *mockStore) CreateBrandContact(_ context.Context, _ store.BrandContact) error     { return nil }
func (m *mockStore) GetBrandContactByID(_ context.Context, _ string) (store.BrandContact, error) { return store.BrandContact{}, nil }
func (m *mockStore) GetBrandContactByViewerToken(_ context.Context, _ string) (store.BrandContact, error) { return store.BrandContact{}, nil }
func (m *mockStore) ListBrandContacts(_ context.Context, _ string) ([]store.BrandContact, error) { return nil, nil }
func (m *mockStore) UpdateBrandContact(_ context.Context, _ string, _ store.BrandContactPatch) error { return nil }
func (m *mockStore) DeactivateBrandContact(_ context.Context, _ string) error             { return nil }
func (m *mockStore) RegenerateBrandContactPassword(_ context.Context, _, _ string) error  { return nil }
func (m *mockStore) ValidateBrandContactAccess(_ context.Context, _, _ string) (store.BrandContact, error) { return store.BrandContact{}, nil }
func (m *mockStore) CreateTalent(_ context.Context, _ store.Talent) error                 { return nil }
func (m *mockStore) GetTalentByID(_ context.Context, _ string) (store.Talent, error)      { return store.Talent{}, nil }
func (m *mockStore) GetTalentByUserID(_ context.Context, _ string) (store.Talent, error)  { return store.Talent{}, nil }
func (m *mockStore) ListTalents(_ context.Context, _ store.TalentFilter) ([]store.Talent, error) { return nil, nil }
func (m *mockStore) UpdateTalent(_ context.Context, _ string, _ store.TalentPatch) error  { return nil }
func (m *mockStore) ListAllTalents(_ context.Context) ([]store.Talent, error)             { return nil, nil }
func (m *mockStore) CreateCampaign(_ context.Context, _ store.Campaign) error             { return nil }
func (m *mockStore) GetCampaignByID(_ context.Context, _ string) (store.Campaign, error)  { return m.campaign, nil }
func (m *mockStore) GetCampaignByHumanID(_ context.Context, _ string) (store.Campaign, error) { return store.Campaign{}, nil }
func (m *mockStore) ListCampaigns(_ context.Context, _ store.CampaignFilter) ([]store.Campaign, error) { return nil, nil }
func (m *mockStore) ListActiveCampaigns(_ context.Context) ([]store.Campaign, error)      { return nil, nil }
func (m *mockStore) UpdateCampaign(_ context.Context, _ string, _ store.CampaignPatch) error { return nil }
func (m *mockStore) DecrementRemainingBudget(_ context.Context, _ string, _ float64) error { return nil }
func (m *mockStore) NextCampaignHumanID(_ context.Context, _ string) (string, error)      { return "", nil }
func (m *mockStore) GetCampaignsByManagerID(_ context.Context, _ string) ([]store.Campaign, error) { return nil, nil }
func (m *mockStore) AssignManagerToCampaign(_ context.Context, _, _, _ string) error      { return nil }
func (m *mockStore) UnassignManagerFromCampaign(_ context.Context, _, _ string) error     { return nil }
func (m *mockStore) ListManagersByCampaignID(_ context.Context, _ string) ([]store.User, error) { return nil, nil }
func (m *mockStore) TryRecordEmailDispatch(_ context.Context, _, _, _, _ string) (bool, error) { return true, nil }
func (m *mockStore) EmailDispatchExists(_ context.Context, _, _, _, _ string) (bool, error) { return false, nil }

func (m *mockStore) CreateCycle(_ context.Context, _ store.Cycle) error                   { return nil }
func (m *mockStore) ListCyclesByCampaign(_ context.Context, _ string) ([]store.Cycle, error) { return nil, nil }
func (m *mockStore) GetActiveCycles(_ context.Context) ([]store.Cycle, error)             { return nil, nil }
func (m *mockStore) UpdateCycle(_ context.Context, _ string, _ store.CyclePatch) error    { return nil }
func (m *mockStore) CloseCycle(_ context.Context, _ string, _ float64) error              { return nil }
func (m *mockStore) CreateBudgetSlots(_ context.Context, _ []store.BudgetSlot) error      { return nil }
func (m *mockStore) ListSlotsByCycle(_ context.Context, _ string) ([]store.BudgetSlot, error) { return nil, nil }
func (m *mockStore) AssignSlot(_ context.Context, _, _ string) error                      { return nil }
func (m *mockStore) CreateAssignment(_ context.Context, _ store.TalentAssignment) error   { return nil }
func (m *mockStore) GetAssignment(_ context.Context, _, _ string) (store.TalentAssignment, error) { return store.TalentAssignment{}, nil }
func (m *mockStore) UpdateAssignment(_ context.Context, _, _ string, _ store.AssignmentPatch) error { return nil }
func (m *mockStore) CreateTrackingLink(_ context.Context, _ store.TrackingLink) error     { return nil }
func (m *mockStore) GetTrackingLinkByToken(_ context.Context, _ string) (store.TrackingLink, error) { return store.TrackingLink{}, nil }
func (m *mockStore) ListTrackingLinksByCycle(_ context.Context, _ string) ([]store.TrackingLink, error) { return nil, nil }
func (m *mockStore) LogConversionEvent(_ context.Context, _ store.ConversionEvent) error  { return nil }
func (m *mockStore) GetDailyConversionCount(_ context.Context, _, _ string, _ time.Time) (float64, error) { return 0, nil }
func (m *mockStore) GetTotalConversions(_ context.Context, _ string) (float64, error)     { return 0, nil }
func (m *mockStore) FlagFallbackConversions(_ context.Context, _ string, _ time.Time) error { return nil }
func (m *mockStore) LockFallbackConversions(_ context.Context, _ string) error            { return nil }
func (m *mockStore) GetCycleState(_ context.Context, _, _ string) (store.CycleState, error) { return store.CycleState{}, nil }
func (m *mockStore) UpsertCycleState(_ context.Context, _ store.CycleState) error         { return nil }
func (m *mockStore) GetDailyOutputs(_ context.Context, _, _ string) ([]float64, error)    { return nil, nil }
func (m *mockStore) GetTalentBaseline(_ context.Context, _ string) (store.TalentBaseline, error) { return store.TalentBaseline{}, nil }
func (m *mockStore) UpsertTalentBaseline(_ context.Context, _ store.TalentBaseline) error { return nil }
func (m *mockStore) GetCategoryBaseline(_ context.Context, _ string) (algo.CategoryBaseline, error) { return algo.CategoryBaseline{}, nil }
func (m *mockStore) GetTalentTodayOutput(_ context.Context, _ string, _ time.Time) (float64, error) { return 0, nil }
func (m *mockStore) GetTalentOutputWindow(_ context.Context, _ string, _ int) ([]float64, error) { return nil, nil }
func (m *mockStore) ListPayoutsByCycle(_ context.Context, _ string) ([]store.PayoutRecord, error) { return m.payout_recs, nil }
func (m *mockStore) ListPayoutsByTalent(_ context.Context, _ string) ([]store.PayoutRecord, error) { return m.payout_recs, nil }
func (m *mockStore) CreateViewer(_ context.Context, _ store.CampaignViewer) error         { return nil }
func (m *mockStore) GetViewerByToken(_ context.Context, _ string) (store.CampaignViewer, error) { return store.CampaignViewer{}, nil }
func (m *mockStore) ListViewersByCampaign(_ context.Context, _ string) ([]store.CampaignViewer, error) { return nil, nil }
func (m *mockStore) UpdateViewer(_ context.Context, _ string, _ store.ViewerPatch) error  { return nil }
func (m *mockStore) AddViewerPassword(_ context.Context, _ store.ViewerPassword) error    { return nil }
func (m *mockStore) ListViewerPasswords(_ context.Context, _ string) ([]store.ViewerPassword, error) { return nil, nil }
func (m *mockStore) DeactivateViewerPassword(_ context.Context, _ string) error           { return nil }
func (m *mockStore) ValidateViewerPassword(_ context.Context, _, _ string) (bool, error)  { return false, nil }
func (m *mockStore) ListAuditLog(_ context.Context, _, _ string) ([]store.AuditLog, error) { return nil, nil }

// ─── Tests ────────────────────────────────────────────────────────────────────

func makeTrafficSetup(budget float64, tier int, conversions float64) (*mockStore, string) {
	cycle := store.Cycle{
		ID:               "cycle-1",
		Campaign_id:      "camp-1",
		Campaign_type:    store.Type_direct_traffic,
		Remaining_budget: budget,
	}
	assignments := []store.TalentAssignment{
		{Talent_id: "t1", Cycle_id: "cycle-1", Status: "active", Effective_tier: tier},
	}
	convs := map[string]float64{"t1": conversions}
	return newMock(cycle, assignments, convs), "cycle-1"
}

func TestComputeAndStorePayout_TrafficPipeline(t *testing.T) {
	ms, cycle_id := makeTrafficSetup(50000, 5000, 10)
	svc := payout.New(ms, nil, nil)
	if err := svc.ComputeAndStoreCyclePayout(context.Background(), cycle_id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ms.payout_recs) == 0 {
		t.Fatal("expected payout records to be created")
	}
	r := ms.payout_recs[0]
	if r.Talent_id != "t1" {
		t.Errorf("expected talent t1, got %s", r.Talent_id)
	}
	if r.Pipeline_type != store.Type_direct_traffic {
		t.Errorf("expected direct_traffic pipeline, got %s", r.Pipeline_type)
	}
	if r.Commission_rate != 0.30 {
		t.Errorf("expected commission_rate 0.30, got %f", r.Commission_rate)
	}
	if r.Final_payout < 0 {
		t.Errorf("final_payout should not be negative, got %f", r.Final_payout)
	}
}

func TestComputeAndStorePayout_LeadPipeline(t *testing.T) {
	cycle := store.Cycle{
		ID:               "cycle-2",
		Campaign_id:      "camp-1",
		Campaign_type:    store.Type_lead_validation,
		Remaining_budget: 50000,
		KPB_config:       []store.KPBDefinition{{Label: "signup", Cost: 500, Quantity: 1}},
	}
	assignments := []store.TalentAssignment{
		{Talent_id: "t2", Cycle_id: "cycle-2", Status: "active", Effective_tier: 10000},
	}
	convs := map[string]float64{"t2": 5}
	ms := newMock(cycle, assignments, convs)
	svc := payout.New(ms, nil, nil)
	if err := svc.ComputeAndStoreCyclePayout(context.Background(), "cycle-2"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ms.payout_recs) == 0 {
		t.Fatal("expected payout records")
	}
	r := ms.payout_recs[0]
	if r.Pipeline_type != store.Type_lead_validation {
		t.Errorf("expected lead_validation pipeline, got %s", r.Pipeline_type)
	}
	if r.KPB_pool_source == "" {
		t.Error("kpb_pool_source should be set for lead pipeline")
	}
}

func TestComputeAndStorePayout_ScaleFactorApplied(t *testing.T) {
	// Pool=1000, two talents each with tier=1000 and 20 conversions at target_cpa=50.
	// gross_base = min(20*50, 1000) = 1000; e_net = 700 each → total=1400 > pool=1000 → scale_factor < 1.
	cycle := store.Cycle{
		ID:               "cycle-3",
		Campaign_id:      "camp-1",
		Campaign_type:    store.Type_direct_traffic,
		Remaining_budget: 1000,
	}
	campaign := store.Campaign{ID: "camp-1", Target_cpa: 50}
	assignments := []store.TalentAssignment{
		{Talent_id: "ta", Cycle_id: "cycle-3", Status: "active", Effective_tier: 1000},
		{Talent_id: "tb", Cycle_id: "cycle-3", Status: "active", Effective_tier: 1000},
	}
	convs := map[string]float64{"ta": 20, "tb": 20}
	ms := newMock(cycle, assignments, convs)
	ms.campaign = campaign
	svc := payout.New(ms, nil, nil)
	if err := svc.ComputeAndStoreCyclePayout(context.Background(), "cycle-3"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range ms.payout_recs {
		if r.Scale_factor >= 1.0 {
			t.Errorf("expected scale_factor < 1 when pool < demand, got %f for %s", r.Scale_factor, r.Talent_id)
		}
	}
}

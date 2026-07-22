package assignment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/src/assignment"
	"github.com/Ze-uus/talent-backend/internal/store"
)

// ─── Mock store ───────────────────────────────────────────────────────────────

type mockStore struct {
	cycle       store.Cycle
	campaign    store.Campaign
	slots       []store.BudgetSlot
	talents     []store.Talent
	baselines   map[string]store.TalentBaseline
	assignments []store.TalentAssignment
	links       []store.TrackingLink
}

func newMock(cycle store.Cycle, campaign store.Campaign, slots []store.BudgetSlot, talents []store.Talent, baselines map[string]store.TalentBaseline) *mockStore {
	return &mockStore{cycle: cycle, campaign: campaign, slots: slots, talents: talents, baselines: baselines}
}

func (m *mockStore) GetCycleByID(_ context.Context, id string) (store.Cycle, error) {
	if m.cycle.ID == id {
		return m.cycle, nil
	}
	return store.Cycle{}, errors.New("not_found")
}
func (m *mockStore) GetCampaignByID(_ context.Context, id string) (store.Campaign, error) {
	if m.campaign.ID == id {
		return m.campaign, nil
	}
	return store.Campaign{}, errors.New("not_found")
}
func (m *mockStore) ListSlotsByCycle(_ context.Context, _ string) ([]store.BudgetSlot, error) {
	return m.slots, nil
}
func (m *mockStore) ListAllTalents(_ context.Context) ([]store.Talent, error) {
	return m.talents, nil
}
func (m *mockStore) GetTalentBaseline(_ context.Context, talent_id string) (store.TalentBaseline, error) {
	b, ok := m.baselines[talent_id]
	if !ok {
		return store.TalentBaseline{}, errors.New("not_found")
	}
	return b, nil
}
func (m *mockStore) CreateAssignment(_ context.Context, a store.TalentAssignment) error {
	m.assignments = append(m.assignments, a)
	return nil
}
func (m *mockStore) AssignSlot(_ context.Context, _, _ string) error { return nil }
func (m *mockStore) CreateTrackingLink(_ context.Context, l store.TrackingLink) error {
	m.links = append(m.links, l)
	return nil
}
func (m *mockStore) WriteAuditLog(_ context.Context, _ store.AuditLog) error { return nil }

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
func (m *mockStore) CreateCampaign(_ context.Context, _ store.Campaign) error             { return nil }
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
func (m *mockStore) GetAssignment(_ context.Context, _, _ string) (store.TalentAssignment, error) { return store.TalentAssignment{}, nil }
func (m *mockStore) ListAssignedTalents(_ context.Context, _ string) ([]store.TalentAssignment, error) { return m.assignments, nil }
func (m *mockStore) ListAssignmentsByTalent(_ context.Context, _ string) ([]store.TalentAssignment, error) { return m.assignments, nil }
func (m *mockStore) UpdateAssignment(_ context.Context, _, _ string, _ store.AssignmentPatch) error { return nil }
func (m *mockStore) GetTrackingLinkByToken(_ context.Context, _ string) (store.TrackingLink, error) { return store.TrackingLink{}, nil }
func (m *mockStore) ListTrackingLinksByCycle(_ context.Context, _ string) ([]store.TrackingLink, error) { return nil, nil }
func (m *mockStore) LogConversionEvent(_ context.Context, _ store.ConversionEvent) error  { return nil }
func (m *mockStore) GetDailyConversionCount(_ context.Context, _, _ string, _ time.Time) (float64, error) { return 0, nil }
func (m *mockStore) GetTotalConversions(_ context.Context, _ string) (float64, error)     { return 0, nil }
func (m *mockStore) GetTalentConversions(_ context.Context, _, _ string) (float64, error) { return 0, nil }
func (m *mockStore) FlagFallbackConversions(_ context.Context, _ string, _ time.Time) error { return nil }
func (m *mockStore) LockFallbackConversions(_ context.Context, _ string) error            { return nil }
func (m *mockStore) GetCycleState(_ context.Context, _, _ string) (store.CycleState, error) { return store.CycleState{}, nil }
func (m *mockStore) UpsertCycleState(_ context.Context, _ store.CycleState) error         { return nil }
func (m *mockStore) GetDailyOutputs(_ context.Context, _, _ string) ([]float64, error)    { return nil, nil }
func (m *mockStore) UpsertTalentBaseline(_ context.Context, _ store.TalentBaseline) error { return nil }
func (m *mockStore) GetCategoryBaseline(_ context.Context, _ string) (algo.CategoryBaseline, error) { return algo.CategoryBaseline{}, nil }
func (m *mockStore) GetTalentTodayOutput(_ context.Context, _ string, _ time.Time) (float64, error) { return 0, nil }
func (m *mockStore) GetTalentOutputWindow(_ context.Context, _ string, _ int) ([]float64, error) { return nil, nil }
func (m *mockStore) CreatePayoutRecord(_ context.Context, _ store.PayoutRecord) error     { return nil }
func (m *mockStore) GetPayoutRecord(_ context.Context, _, _ string) (store.PayoutRecord, error) { return store.PayoutRecord{}, nil }
func (m *mockStore) ListPayoutsByCycle(_ context.Context, _ string) ([]store.PayoutRecord, error) { return nil, nil }
func (m *mockStore) ListPayoutsByTalent(_ context.Context, _ string) ([]store.PayoutRecord, error) { return nil, nil }
func (m *mockStore) UpdatePayoutRecord(_ context.Context, _ string, _ store.PayoutPatch) error { return nil }
func (m *mockStore) CreateViewer(_ context.Context, _ store.CampaignViewer) error         { return nil }
func (m *mockStore) GetViewerByToken(_ context.Context, _ string) (store.CampaignViewer, error) { return store.CampaignViewer{}, nil }
func (m *mockStore) ListViewersByCampaign(_ context.Context, _ string) ([]store.CampaignViewer, error) { return nil, nil }
func (m *mockStore) UpdateViewer(_ context.Context, _ string, _ store.ViewerPatch) error  { return nil }
func (m *mockStore) AddViewerPassword(_ context.Context, _ store.ViewerPassword) error    { return nil }
func (m *mockStore) ListViewerPasswords(_ context.Context, _ string) ([]store.ViewerPassword, error) { return nil, nil }
func (m *mockStore) DeactivateViewerPassword(_ context.Context, _ string) error           { return nil }
func (m *mockStore) ValidateViewerPassword(_ context.Context, _, _ string) (bool, error)  { return false, nil }
func (m *mockStore) ListAuditLog(_ context.Context, _, _ string) ([]store.AuditLog, error) { return nil, nil }

// ─── Test helpers ─────────────────────────────────────────────────────────────

func makeCycleAndCampaign() (store.Cycle, store.Campaign) {
	campaign := store.Campaign{
		ID:            "camp-1",
		Campaign_type: store.Type_direct_traffic,
		Target_cpa:    100,
		Max_cpa:       150,
		Audience:      "student",
	}
	cycle := store.Cycle{
		ID:            "cycle-1",
		Campaign_id:   "camp-1",
		Campaign_type: store.Type_direct_traffic,
		Cycle_budget:  50000,
		Start_date:    time.Now(),
		End_date:      time.Now().Add(7 * 24 * time.Hour),
	}
	return cycle, campaign
}

func makeSlots() []store.BudgetSlot {
	return []store.BudgetSlot{
		{ID: "slot-1", Tier_value: 5000, Cycle_id: "cycle-1"},
		{ID: "slot-2", Tier_value: 10000, Cycle_id: "cycle-1"},
	}
}

func makeTalentsAndBaselines() ([]store.Talent, map[string]store.TalentBaseline) {
	talents := []store.Talent{
		{ID: "t1", Category: store.Category_student, Status: store.Status_active, Max_tier: 15000},
		{ID: "t2", Category: store.Category_micro, Status: store.Status_active, Max_tier: 15000},
	}
	baselines := map[string]store.TalentBaseline{
		"t1": {Talent_id: "t1", Lambda_lt: 5.0},
		"t2": {Talent_id: "t2", Lambda_lt: 2.0},
	}
	return talents, baselines
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestRunSolver_MatchScoresComputed(t *testing.T) {
	cycle, campaign := makeCycleAndCampaign()
	slots := makeSlots()
	talents, baselines := makeTalentsAndBaselines()

	svc := assignment.New(newMock(cycle, campaign, slots, talents, baselines), nil, nil, nil)
	output, err := svc.RunSolver(context.Background(), "cycle-1")
	if err != nil {
		t.Fatalf("RunSolver error: %v", err)
	}
	if len(output.Match_scores) == 0 {
		t.Fatal("expected match_scores to be populated")
	}
	for tid, ms := range output.Match_scores {
		if ms.MS_t <= 0 {
			t.Errorf("talent %s: expected MS_t > 0, got %f", tid, ms.MS_t)
		}
	}
}

func TestRunSolver_MatchAdjustmentLowersCost(t *testing.T) {
	cycle, campaign := makeCycleAndCampaign()
	// t1 has category "student" matching audience "student" → DM=1.0 → higher MS_t → lower cost
	// t2 has category "micro" not matching "student" → DM=0.3 → lower MS_t → higher cost
	slots := makeSlots()
	talents, baselines := makeTalentsAndBaselines()

	svc := assignment.New(newMock(cycle, campaign, slots, talents, baselines), nil, nil, nil)
	output, err := svc.RunSolver(context.Background(), "cycle-1")
	if err != nil {
		t.Fatalf("RunSolver error: %v", err)
	}

	ms_t1 := output.Match_scores["t1"].MS_t
	ms_t2 := output.Match_scores["t2"].MS_t
	if ms_t1 <= ms_t2 {
		t.Errorf("t1 (category matches audience) should have higher MS_t than t2: t1=%f t2=%f", ms_t1, ms_t2)
	}
}

func TestConfirmAssignments_EmitsTalentUpdate(t *testing.T) {
	cycle, campaign := makeCycleAndCampaign()
	slots := makeSlots()
	talents, baselines := makeTalentsAndBaselines()
	ms := newMock(cycle, campaign, slots, talents, baselines)

	ch := make(chan domain.TalentUpdateEvent, 4)
	svc := assignment.New(ms, ch, nil, nil)

	solver_out, err := svc.RunSolver(context.Background(), "cycle-1")
	if err != nil {
		t.Fatalf("RunSolver error: %v", err)
	}

	confirmed := []assignment.ConfirmedAssignment{{
		Talent_id:      "t1",
		Slot_id:        "slot-1",
		Role_label:     "primary",
		PDC_mode:       store.Pdc_cold_start,
		PDC_value:      5.0,
		Effective_tier: 5000,
	}}
	if err := svc.ConfirmAssignments(context.Background(), "cycle-1", "camp-1", "actor-1", confirmed, solver_out, false); err != nil {
		t.Fatalf("ConfirmAssignments error: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Update_type != "assigned" {
			t.Fatalf("expected assigned, got %q", ev.Update_type)
		}
		if ev.Shard.Tracking_token == "" {
			t.Fatal("expected tracking token in shard")
		}
	default:
		t.Fatal("expected talent_update event")
	}
}

func TestConfirmAssignments_MatchFieldsPersisted(t *testing.T) {
	cycle, campaign := makeCycleAndCampaign()
	slots := makeSlots()
	talents, baselines := makeTalentsAndBaselines()
	ms := newMock(cycle, campaign, slots, talents, baselines)

	svc := assignment.New(ms, nil, nil, nil)
	solver_out, err := svc.RunSolver(context.Background(), "cycle-1")
	if err != nil {
		t.Fatalf("RunSolver error: %v", err)
	}

	confirmed := []assignment.ConfirmedAssignment{
		{
			Talent_id:      "t1",
			Slot_id:        "slot-1",
			Role_label:     "primary",
			PDC_mode:       store.Pdc_cold_start,
			PDC_value:      5.0,
			Effective_tier: 5000,
		},
	}
	if err := svc.ConfirmAssignments(context.Background(), "cycle-1", "camp-1", "actor-1", confirmed, solver_out, false); err != nil {
		t.Fatalf("ConfirmAssignments error: %v", err)
	}
	if len(ms.assignments) == 0 {
		t.Fatal("expected at least one assignment to be created")
	}
	if ms.assignments[0].Match_score <= 0 {
		t.Errorf("expected Match_score > 0, got %f", ms.assignments[0].Match_score)
	}
}

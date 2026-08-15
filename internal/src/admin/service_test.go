package admin_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/src/admin"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type mockStore struct {
	talents     map[string]store.Talent
	users       map[string]store.User
	baselines   map[string]store.TalentBaseline
	assignments map[string][]store.TalentAssignment
	payouts     map[string][]store.PayoutRecord
	conversions map[string]float64
}

func newMock() *mockStore {
	return &mockStore{
		talents:     make(map[string]store.Talent),
		users:       make(map[string]store.User),
		baselines:   make(map[string]store.TalentBaseline),
		assignments: make(map[string][]store.TalentAssignment),
		payouts:     make(map[string][]store.PayoutRecord),
		conversions: make(map[string]float64),
	}
}

func (m *mockStore) UpdateTalent(_ context.Context, id string, patch store.TalentPatch) error {
	t, ok := m.talents[id]
	if !ok {
		return errors.New("not_found")
	}
	if patch.Status != nil {
		t.Status = store.Talent_status(*patch.Status)
	}
	if patch.Category != nil {
		t.Category = store.Talent_category(*patch.Category)
	}
	m.talents[id] = t
	return nil
}
func (m *mockStore) GetTalentByID(_ context.Context, id string) (store.Talent, error) {
	t, ok := m.talents[id]
	if !ok {
		return store.Talent{}, errors.New("not_found")
	}
	return t, nil
}

// stub remaining Store methods
func (m *mockStore) Ping(_ context.Context) error { return nil }
func (m *mockStore) CreateUser(_ context.Context, u store.User) error {
	if u.ID == "" {
		u.ID = "u1"
	}
	m.users[u.ID] = u
	return nil
}
func (m *mockStore) GetUserByID(_ context.Context, id string) (store.User, error) {
	u, ok := m.users[id]
	if !ok {
		return store.User{}, errors.New("not_found")
	}
	return u, nil
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
func (m *mockStore) UpdateUser(_ context.Context, id string, patch store.UserPatch) error {
	u, ok := m.users[id]
	if !ok {
		return errors.New("not_found")
	}
	if patch.Active != nil {
		u.Active = *patch.Active
	}
	if patch.Status != nil {
		u.Status = store.User_status(*patch.Status)
	}
	if patch.Deleted_at != nil {
		u.Deleted_at = patch.Deleted_at
	}
	if patch.Full_name != nil {
		u.Full_name = *patch.Full_name
	}
	m.users[id] = u
	return nil
}
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
func (m *mockStore) CreateTalent(_ context.Context, t store.Talent) error {
	m.talents[t.ID] = t
	return nil
}
func (m *mockStore) GetTalentByUserID(_ context.Context, user_id string) (store.Talent, error) {
	for _, t := range m.talents {
		if t.User_id == user_id {
			return t, nil
		}
	}
	return store.Talent{}, errors.New("not_found")
}
func (m *mockStore) ListTalents(_ context.Context, _ store.TalentFilter) ([]store.Talent, error) {
	out := make([]store.Talent, 0, len(m.talents))
	for _, t := range m.talents {
		out = append(out, t)
	}
	return out, nil
}
func (m *mockStore) CreateCampaign(_ context.Context, _ store.Campaign) error { return nil }
func (m *mockStore) GetCampaignByID(_ context.Context, _ string) (store.Campaign, error) {
	return store.Campaign{}, nil
}
func (m *mockStore) GetCampaignByHumanID(_ context.Context, _ string) (store.Campaign, error) {
	return store.Campaign{}, nil
}
func (m *mockStore) ListCampaigns(_ context.Context, _ store.CampaignFilter) ([]store.Campaign, error) {
	return nil, nil
}
func (m *mockStore) ListActiveCampaigns(_ context.Context) ([]store.Campaign, error) { return nil, nil }
func (m *mockStore) UpdateCampaign(_ context.Context, _ string, _ store.CampaignPatch) error {
	return nil
}
func (m *mockStore) DecrementRemainingBudget(_ context.Context, _ string, _ float64) error {
	return nil
}
func (m *mockStore) NextCampaignHumanID(_ context.Context, _ string) (string, error) { return "", nil }
func (m *mockStore) GetCampaignsByManagerID(_ context.Context, _ string) ([]store.Campaign, error) {
	return nil, nil
}
func (m *mockStore) AssignManagerToCampaign(_ context.Context, _, _, _ string) error  { return nil }
func (m *mockStore) UnassignManagerFromCampaign(_ context.Context, _, _ string) error { return nil }
func (m *mockStore) ListManagersByCampaignID(_ context.Context, _ string) ([]store.User, error) {
	return nil, nil
}
func (m *mockStore) TryRecordEmailDispatch(_ context.Context, _, _, _, _ string) (bool, error) {
	return true, nil
}
func (m *mockStore) EmailDispatchExists(_ context.Context, _, _, _, _ string) (bool, error) {
	return false, nil
}

func (m *mockStore) CreateCycle(_ context.Context, _ store.Cycle) error { return nil }
func (m *mockStore) GetCycleByID(_ context.Context, _ string) (store.Cycle, error) {
	return store.Cycle{}, nil
}
func (m *mockStore) ListCyclesByCampaign(_ context.Context, _ string) ([]store.Cycle, error) {
	return nil, nil
}
func (m *mockStore) GetActiveCycles(_ context.Context) ([]store.Cycle, error)          { return nil, nil }
func (m *mockStore) UpdateCycle(_ context.Context, _ string, _ store.CyclePatch) error { return nil }
func (m *mockStore) CloseCycle(_ context.Context, _ string, _ float64) error           { return nil }
func (m *mockStore) CreateBudgetSlots(_ context.Context, _ []store.BudgetSlot) error   { return nil }
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
func (m *mockStore) ListAssignmentsByTalent(_ context.Context, talentID string) ([]store.TalentAssignment, error) {
	return m.assignments[talentID], nil
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
func (m *mockStore) GetTalentConversions(_ context.Context, talentID, cycleID string) (float64, error) {
	return m.conversions[talentID+":"+cycleID], nil
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
func (m *mockStore) GetTalentBaseline(_ context.Context, talent_id string) (store.TalentBaseline, error) {
	b, ok := m.baselines[talent_id]
	if !ok {
		return store.TalentBaseline{}, errors.New("not_found")
	}
	return b, nil
}
func (m *mockStore) UpsertTalentBaseline(_ context.Context, b store.TalentBaseline) error {
	m.baselines[b.Talent_id] = b
	return nil
}
func (m *mockStore) GetCategoryBaseline(_ context.Context, category string) (algo.CategoryBaseline, error) {
	medians := map[string]float64{
		"student":   2,
		"micro":     5,
		"community": 15,
	}
	median, ok := medians[category]
	if !ok {
		return algo.CategoryBaseline{}, errors.New("not_found")
	}
	return algo.CategoryBaseline{Category: category, Median: median}, nil
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
func (m *mockStore) ListPayoutsByTalent(_ context.Context, talentID string) ([]store.PayoutRecord, error) {
	return m.payouts[talentID], nil
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

func TestApproveTalent_EmitsTalentUpdate(t *testing.T) {
	ch := make(chan domain.TalentUpdateEvent, 1)
	ms := newMock()
	ms.users["user-1"] = store.User{ID: "user-1", Email: "t@test.com", Role: store.Role_talent, Status: store.User_status_pending, Active: true}
	ms.talents["talent-1"] = store.Talent{ID: "talent-1", User_id: "user-1", Status: store.Status_pending, Category: store.Category_student, Report_compliance: 1}

	svc := admin.New(ms, ch, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, 0.97)
	if err := svc.ApproveTalent(context.Background(), "talent-1", store.Category_micro); err != nil {
		t.Fatalf("ApproveTalent error: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Update_type != "approved" {
			t.Fatalf("expected approved, got %q", ev.Update_type)
		}
		if ev.Shard.Status != "active" || ev.Shard.Category != "micro" {
			t.Fatalf("unexpected shard: %+v", ev.Shard)
		}
	default:
		t.Fatal("expected talent_update event")
	}
	if ms.users["user-1"].Status != store.User_status_active {
		t.Fatalf("user status=%s", ms.users["user-1"].Status)
	}
	if ms.talents["talent-1"].Category != store.Category_micro {
		t.Fatalf("category=%s", ms.talents["talent-1"].Category)
	}
}

func TestApproveTalent_ByUserID(t *testing.T) {
	ms := newMock()
	ms.users["user-1"] = store.User{ID: "user-1", Email: "franklin@test.com", Role: store.Role_talent, Status: store.User_status_pending, Active: true}
	ms.talents["talent-1"] = store.Talent{ID: "talent-1", User_id: "user-1", Status: store.Status_pending, Category: store.Category_student}
	svc := admin.New(ms, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, 0.97)
	if err := svc.ApproveTalent(context.Background(), "user-1", store.Category_community); err != nil {
		t.Fatal(err)
	}
	if ms.talents["talent-1"].Status != store.Status_active {
		t.Fatalf("talent status=%s", ms.talents["talent-1"].Status)
	}
	if ms.users["user-1"].Status != store.User_status_active {
		t.Fatalf("user status=%s", ms.users["user-1"].Status)
	}
}

func TestApproveTalent_UnknownID(t *testing.T) {
	ms := newMock()
	svc := admin.New(ms, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, 0.97)
	err := svc.ApproveTalent(context.Background(), "missing", store.Category_student)
	if err == nil || err.Error() != "not_found" {
		t.Fatalf("expected not_found, got %v", err)
	}
}

func TestListTalents_IncludesStats(t *testing.T) {
	ms := newMock()
	ms.users["user-1"] = store.User{ID: "user-1", Email: "a@b.com", Full_name: "Ada", Role: store.Role_talent}
	ms.talents["talent-1"] = store.Talent{
		ID: "talent-1", User_id: "user-1", Status: store.Status_active,
		Category: store.Category_student, Report_compliance: 0.8,
	}
	ms.users["user-2"] = store.User{ID: "user-2", Email: "c@d.com", Full_name: "Bob", Role: store.Role_talent}
	ms.talents["talent-2"] = store.Talent{
		ID: "talent-2", User_id: "user-2", Status: store.Status_pending,
		Category: store.Category_micro, Report_compliance: 1.0,
	}
	svc := admin.New(ms, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, 0.97)
	out, err := svc.ListTalents(context.Background(), store.TalentFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if out.Stats.Humans != 2 || out.Stats.Active_humans != 1 {
		t.Fatalf("stats=%+v", out.Stats)
	}
	if out.Stats.Avg_performance != 90 { // (0.8+1.0)/2 * 100
		t.Fatalf("avg_performance=%v", out.Stats.Avg_performance)
	}
	if len(out.Talents) != 2 {
		t.Fatalf("talents=%d", len(out.Talents))
	}
	found := false
	for _, item := range out.Talents {
		if item.ID == "talent-1" && item.Email == "a@b.com" && item.Full_name == "Ada" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected enriched talent: %+v", out.Talents)
	}
}

func TestInviteAdmin_SetsInvitedStatus(t *testing.T) {
	// covered in auth package; admin soft-delete/patch tested below
}

func TestSuspendUser_SetsStatus(t *testing.T) {
	ms := newMock()
	ms.users["u1"] = store.User{ID: "u1", Email: "a@b.com", Role: store.Role_admin, Active: true, Status: store.User_status_active}
	svc := admin.New(ms, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, 0.97)
	if err := svc.SuspendUser(context.Background(), "u1"); err != nil {
		t.Fatal(err)
	}
	u := ms.users["u1"]
	if u.Status != store.User_status_suspended || u.Active {
		t.Fatalf("got status=%s active=%v", u.Status, u.Active)
	}
}

func TestSoftDeleteUser_SetsDeletedAt(t *testing.T) {
	ms := newMock()
	ms.users["u1"] = store.User{ID: "u1", Email: "a@b.com", Role: store.Role_admin, Active: true, Status: store.User_status_active}
	svc := admin.New(ms, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, 0.97)
	if err := svc.SoftDeleteUser(context.Background(), "u1"); err != nil {
		t.Fatal(err)
	}
	u := ms.users["u1"]
	if u.Status != store.User_status_deleted || u.Active || u.Deleted_at == nil {
		t.Fatalf("got status=%s active=%v deleted_at=%v", u.Status, u.Active, u.Deleted_at)
	}
}

func TestPatchUser_ActiveFalse_Suspends(t *testing.T) {
	ms := newMock()
	ms.users["u1"] = store.User{ID: "u1", Active: true, Status: store.User_status_active}
	svc := admin.New(ms, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, 0.97)
	active := false
	if err := svc.PatchUser(context.Background(), "u1", store.UserPatch{Active: &active}); err != nil {
		t.Fatal(err)
	}
	u := ms.users["u1"]
	if u.Status != store.User_status_suspended || u.Active {
		t.Fatalf("got status=%s active=%v", u.Status, u.Active)
	}
}

func TestPatchUser_ActiveTrue_OnInvited_Rejected(t *testing.T) {
	ms := newMock()
	ms.users["u1"] = store.User{ID: "u1", Active: false, Status: store.User_status_invited}
	svc := admin.New(ms, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, 0.97)
	active := true
	err := svc.PatchUser(context.Background(), "u1", store.UserPatch{Active: &active})
	if err == nil || err.Error() != "invalid_active_transition" {
		t.Fatalf("expected invalid_active_transition, got %v", err)
	}
}

func TestApproveTalent_BootstrapsColdStartBaseline(t *testing.T) {
	ms := newMock()
	ms.users["user-1"] = store.User{ID: "user-1", Email: "t@test.com", Role: store.Role_talent, Status: store.User_status_pending, Active: true}
	ms.talents["talent-1"] = store.Talent{ID: "talent-1", User_id: "user-1", Status: store.Status_pending, Category: store.Category_student}
	svc := admin.New(ms, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, 0.97)
	if err := svc.ApproveTalent(context.Background(), "talent-1", store.Category_community); err != nil {
		t.Fatal(err)
	}
	b, ok := ms.baselines["talent-1"]
	if !ok {
		t.Fatal("expected baseline")
	}
	if b.Lambda_lt != 15 || b.Alpha_lt != 15 || b.Beta_lt != 1 || b.Delta_lt != 0.97 {
		t.Fatalf("baseline=%+v", b)
	}
}

func TestApproveTalent_PreservesExistingBaseline(t *testing.T) {
	ms := newMock()
	ms.users["user-1"] = store.User{ID: "user-1", Email: "t@test.com", Role: store.Role_talent, Status: store.User_status_pending, Active: true}
	ms.talents["talent-1"] = store.Talent{ID: "talent-1", User_id: "user-1", Status: store.Status_pending, Category: store.Category_student}
	ms.baselines["talent-1"] = store.TalentBaseline{Talent_id: "talent-1", Alpha_lt: 40, Beta_lt: 4, Lambda_lt: 10, Delta_lt: 0.97}
	svc := admin.New(ms, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, 0.97)
	if err := svc.ApproveTalent(context.Background(), "talent-1", store.Category_community); err != nil {
		t.Fatal(err)
	}
	b := ms.baselines["talent-1"]
	if b.Lambda_lt != 10 || b.Alpha_lt != 40 {
		t.Fatalf("baseline overwritten: %+v", b)
	}
}

func TestGetTalentDetail_ReturnsSafeUserStatsAssignmentsAndPayouts(t *testing.T) {
	ms := newMock()
	ms.users["user-1"] = store.User{
		ID: "user-1", Email: "ada@example.com", Full_name: "Ada Human",
		Phone_number: "+2348012345678", Avatar_url: "https://example.com/ada.jpg",
		Role: store.Role_talent, Status: store.User_status_active, Active: true,
	}
	ms.talents["talent-1"] = store.Talent{
		ID: "talent-1", User_id: "user-1", Status: store.Status_active,
		Category: store.Category_micro, Skills: []string{"video"}, Report_compliance: 0.95,
	}
	ms.assignments["talent-1"] = []store.TalentAssignment{
		{Talent_id: "talent-1", Campaign_id: "campaign-1", Cycle_id: "cycle-1", Status: "active", Match_score: 0.8},
		{Talent_id: "talent-1", Campaign_id: "campaign-2", Cycle_id: "cycle-2", Status: "completed", Match_score: 1},
	}
	ms.conversions["talent-1:cycle-1"] = 8
	ms.conversions["talent-1:cycle-2"] = 4
	ms.payouts["talent-1"] = []store.PayoutRecord{
		{Campaign_id: "campaign-1", Cycle_id: "cycle-1", Status: store.Payout_paid, Final_payout: 1000},
		{Campaign_id: "campaign-2", Cycle_id: "cycle-2", Status: store.Payout_approved, Final_payout: 500},
	}

	svc := admin.New(ms, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil, nil, 0.97)
	detail, err := svc.GetTalentDetail(context.Background(), "talent-1")
	if err != nil {
		t.Fatal(err)
	}
	if detail.User.PhoneNumber != "+2348012345678" || detail.User.FullName != "Ada Human" {
		t.Fatalf("user=%+v", detail.User)
	}
	if detail.Stats.TotalAssignments != 2 || detail.Stats.ActiveAssignments != 1 ||
		detail.Stats.TotalConversions != 12 || detail.Stats.TotalEarned != 1500 ||
		detail.Stats.TotalPaid != 1000 {
		t.Fatalf("stats=%+v", detail.Stats)
	}
}

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
	talents map[string]store.Talent
}

func newMock() *mockStore {
	return &mockStore{talents: make(map[string]store.Talent)}
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
func (m *mockStore) CreateTalent(_ context.Context, t store.Talent) error {
	m.talents[t.ID] = t
	return nil
}
func (m *mockStore) GetTalentByUserID(_ context.Context, _ string) (store.Talent, error) {
	return store.Talent{}, nil
}
func (m *mockStore) ListTalents(_ context.Context, _ store.TalentFilter) ([]store.Talent, error) {
	return nil, nil
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
func (m *mockStore) DecrementRemainingBudget(_ context.Context, _ string, _ float64) error { return nil }
func (m *mockStore) NextCampaignHumanID(_ context.Context, _ string) (string, error)     { return "", nil }
func (m *mockStore) GetCampaignsByManagerID(_ context.Context, _ string) ([]store.Campaign, error) {
	return nil, nil
}
func (m *mockStore) AssignManagerToCampaign(_ context.Context, _, _, _ string) error { return nil }
func (m *mockStore) UnassignManagerFromCampaign(_ context.Context, _, _ string) error { return nil }
func (m *mockStore) ListManagersByCampaignID(_ context.Context, _ string) ([]store.User, error) { return nil, nil }
func (m *mockStore) TryRecordEmailDispatch(_ context.Context, _, _, _, _ string) (bool, error) { return true, nil }
func (m *mockStore) EmailDispatchExists(_ context.Context, _, _, _, _ string) (bool, error) { return false, nil }

func (m *mockStore) CreateCycle(_ context.Context, _ store.Cycle) error                { return nil }
func (m *mockStore) GetCycleByID(_ context.Context, _ string) (store.Cycle, error) {
	return store.Cycle{}, nil
}
func (m *mockStore) ListCyclesByCampaign(_ context.Context, _ string) ([]store.Cycle, error) {
	return nil, nil
}
func (m *mockStore) GetActiveCycles(_ context.Context) ([]store.Cycle, error) { return nil, nil }
func (m *mockStore) UpdateCycle(_ context.Context, _ string, _ store.CyclePatch) error { return nil }
func (m *mockStore) CloseCycle(_ context.Context, _ string, _ float64) error           { return nil }
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
func (m *mockStore) FlagFallbackConversions(_ context.Context, _ string, _ time.Time) error { return nil }
func (m *mockStore) LockFallbackConversions(_ context.Context, _ string) error               { return nil }
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
func (m *mockStore) WriteAuditLog(_ context.Context, _ store.AuditLog) error    { return nil }

func TestApproveTalent_EmitsTalentUpdate(t *testing.T) {
	ch := make(chan domain.TalentUpdateEvent, 1)
	ms := newMock()
	ms.talents["talent-1"] = store.Talent{ID: "talent-1", Status: store.Status_pending}

	svc := admin.New(ms, ch, slog.New(slog.NewTextHandler(os.Stderr, nil)), nil)
	if err := svc.ApproveTalent(context.Background(), "talent-1", store.Category_student); err != nil {
		t.Fatalf("ApproveTalent error: %v", err)
	}

	select {
	case ev := <-ch:
		if ev.Update_type != "approved" {
			t.Fatalf("expected approved, got %q", ev.Update_type)
		}
		if ev.Shard.Status != "active" || ev.Shard.Category != "student" {
			t.Fatalf("unexpected shard: %+v", ev.Shard)
		}
	default:
		t.Fatal("expected talent_update event")
	}
}

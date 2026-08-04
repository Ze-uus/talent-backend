package brand_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/media"
	"github.com/Ze-uus/talent-backend/internal/src/brand"
	"github.com/Ze-uus/talent-backend/internal/store"
)

// ─── Mock store ───────────────────────────────────────────────────────────────

type mockStore struct {
	brands         map[string]store.Brand
	contacts       map[string]store.BrandContact
	audit_log      []store.AuditLog
	regen_called   bool
	regen_new_hash string
}

func newMock() *mockStore {
	return &mockStore{
		brands:   make(map[string]store.Brand),
		contacts: make(map[string]store.BrandContact),
	}
}

func (m *mockStore) CreateBrand(_ context.Context, b store.Brand) error {
	if b.ID == "" {
		b.ID = "brand-" + b.Shortcode
	}
	m.brands[b.Shortcode] = b
	return nil
}
func (m *mockStore) GetBrandByID(_ context.Context, id string) (store.Brand, error) {
	for _, b := range m.brands {
		if b.ID == id {
			return b, nil
		}
	}
	return store.Brand{}, errors.New("not_found")
}
func (m *mockStore) GetBrandByShortcode(_ context.Context, sc string) (store.Brand, error) {
	b, ok := m.brands[sc]
	if !ok {
		return store.Brand{}, errors.New("not_found")
	}
	return b, nil
}
func (m *mockStore) ListBrands(_ context.Context, _ store.BrandFilter) ([]store.Brand, error) {
	out := make([]store.Brand, 0, len(m.brands))
	for _, b := range m.brands {
		out = append(out, b)
	}
	return out, nil
}
func (m *mockStore) UpdateBrand(_ context.Context, id string, p store.BrandPatch) error {
	for sc, b := range m.brands {
		if b.ID == id {
			if p.Name != nil {
				b.Name = *p.Name
			}
			if p.Industry != nil {
				b.Industry = *p.Industry
			}
			if p.Description != nil {
				b.Description = *p.Description
			}
			if p.Website != nil {
				b.Website = *p.Website
			}
			if p.Logo_url != nil {
				b.Logo_url = *p.Logo_url
			}
			if p.Status != nil {
				b.Status = *p.Status
			}
			m.brands[sc] = b
			return nil
		}
	}
	return errors.New("not_found")
}
func (m *mockStore) ShortcodeExists(_ context.Context, sc string) (bool, error) {
	_, ok := m.brands[sc]
	return ok, nil
}
func (m *mockStore) CreateBrandContact(_ context.Context, c store.BrandContact) error {
	m.contacts[c.Viewer_token] = c
	return nil
}
func (m *mockStore) GetBrandContactByID(_ context.Context, id string) (store.BrandContact, error) {
	for _, c := range m.contacts {
		if c.ID == id {
			return c, nil
		}
	}
	return store.BrandContact{}, errors.New("not_found")
}
func (m *mockStore) GetBrandContactByViewerToken(_ context.Context, token string) (store.BrandContact, error) {
	c, ok := m.contacts[token]
	if !ok {
		return store.BrandContact{}, errors.New("not_found")
	}
	return c, nil
}
func (m *mockStore) ListBrandContacts(_ context.Context, brand_id string) ([]store.BrandContact, error) {
	var out []store.BrandContact
	for _, c := range m.contacts {
		if c.Brand_id == brand_id {
			out = append(out, c)
		}
	}
	return out, nil
}
func (m *mockStore) UpdateBrandContact(_ context.Context, _ string, _ store.BrandContactPatch) error {
	return nil
}
func (m *mockStore) DeactivateBrandContact(_ context.Context, id string) error {
	for token, c := range m.contacts {
		if c.ID == id {
			c.Token_active = false
			m.contacts[token] = c
			return nil
		}
	}
	return nil
}
func (m *mockStore) RegenerateBrandContactPassword(_ context.Context, id, hash string) error {
	m.regen_called = true
	m.regen_new_hash = hash
	for token, c := range m.contacts {
		if c.ID == id {
			c.Access_password_hash = hash
			m.contacts[token] = c
		}
	}
	return nil
}
func (m *mockStore) ValidateBrandContactAccess(_ context.Context, _, _ string) (store.BrandContact, error) {
	return store.BrandContact{}, nil
}
func (m *mockStore) WriteAuditLog(_ context.Context, entry store.AuditLog) error {
	m.audit_log = append(m.audit_log, entry)
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
func (m *mockStore) ListAuditLog(_ context.Context, _, _ string) ([]store.AuditLog, error) {
	return m.audit_log, nil
}

// stub out the rest of the Store interface
func (m *mockStore) Ping(_ context.Context) error                                      { return nil }
func (m *mockStore) CreateUser(_ context.Context, _ store.User) error                  { return nil }
func (m *mockStore) GetUserByID(_ context.Context, _ string) (store.User, error)       { return store.User{}, nil }
func (m *mockStore) GetUserByEmail(_ context.Context, _ string) (store.User, error)    { return store.User{}, nil }
func (m *mockStore) GetUserByGoogleID(_ context.Context, _ string) (store.User, error) { return store.User{}, nil }
func (m *mockStore) GetUserByInviteToken(_ context.Context, _ string) (store.User, error) { return store.User{}, nil }
func (m *mockStore) UpdateUser(_ context.Context, _ string, _ store.UserPatch) error  { return nil }
func (m *mockStore) ListUsers(_ context.Context, _ store.UserFilter) ([]store.User, error) { return nil, nil }
func (m *mockStore) CreateSession(_ context.Context, _ store.Session) error            { return nil }
func (m *mockStore) GetSession(_ context.Context, _ string) (store.Session, error)     { return store.Session{}, nil }
func (m *mockStore) TouchSession(_ context.Context, _ string, _ time.Time) error       { return nil }
func (m *mockStore) InvalidateSession(_ context.Context, _ string) error               { return nil }
func (m *mockStore) InvalidateAllUserSessions(_ context.Context, _ string) error       { return nil }
func (m *mockStore) ListSessionsByUser(_ context.Context, _ string) ([]store.Session, error) { return nil, nil }
func (m *mockStore) CreateTalent(_ context.Context, _ store.Talent) error              { return nil }
func (m *mockStore) GetTalentByID(_ context.Context, _ string) (store.Talent, error)   { return store.Talent{}, nil }
func (m *mockStore) GetTalentByUserID(_ context.Context, _ string) (store.Talent, error) { return store.Talent{}, nil }
func (m *mockStore) ListTalents(_ context.Context, _ store.TalentFilter) ([]store.Talent, error) { return nil, nil }
func (m *mockStore) UpdateTalent(_ context.Context, _ string, _ store.TalentPatch) error { return nil }
func (m *mockStore) ListAllTalents(_ context.Context) ([]store.Talent, error)          { return nil, nil }
func (m *mockStore) CreateCampaign(_ context.Context, _ store.Campaign) error          { return nil }
func (m *mockStore) GetCampaignByID(_ context.Context, _ string) (store.Campaign, error) { return store.Campaign{}, nil }
func (m *mockStore) GetCampaignByHumanID(_ context.Context, _ string) (store.Campaign, error) { return store.Campaign{}, nil }
func (m *mockStore) ListCampaigns(_ context.Context, _ store.CampaignFilter) ([]store.Campaign, error) { return nil, nil }
func (m *mockStore) ListActiveCampaigns(_ context.Context) ([]store.Campaign, error)   { return nil, nil }
func (m *mockStore) UpdateCampaign(_ context.Context, _ string, _ store.CampaignPatch) error { return nil }
func (m *mockStore) DecrementRemainingBudget(_ context.Context, _ string, _ float64) error { return nil }
func (m *mockStore) NextCampaignHumanID(_ context.Context, _ string) (string, error)   { return "", nil }
func (m *mockStore) GetCampaignsByManagerID(_ context.Context, _ string) ([]store.Campaign, error) { return nil, nil }
func (m *mockStore) AssignManagerToCampaign(_ context.Context, _, _, _ string) error   { return nil }
func (m *mockStore) UnassignManagerFromCampaign(_ context.Context, _, _ string) error  { return nil }
func (m *mockStore) ListManagersByCampaignID(_ context.Context, _ string) ([]store.User, error) { return nil, nil }
func (m *mockStore) TryRecordEmailDispatch(_ context.Context, _, _, _, _ string) (bool, error) { return true, nil }
func (m *mockStore) EmailDispatchExists(_ context.Context, _, _, _, _ string) (bool, error) { return false, nil }

func (m *mockStore) CreateCycle(_ context.Context, _ store.Cycle) error                { return nil }
func (m *mockStore) GetCycleByID(_ context.Context, _ string) (store.Cycle, error)     { return store.Cycle{}, nil }
func (m *mockStore) ListCyclesByCampaign(_ context.Context, _ string) ([]store.Cycle, error) { return nil, nil }
func (m *mockStore) GetActiveCycles(_ context.Context) ([]store.Cycle, error)          { return nil, nil }
func (m *mockStore) UpdateCycle(_ context.Context, _ string, _ store.CyclePatch) error { return nil }
func (m *mockStore) CloseCycle(_ context.Context, _ string, _ float64) error           { return nil }
func (m *mockStore) CreateBudgetSlots(_ context.Context, _ []store.BudgetSlot) error   { return nil }
func (m *mockStore) ListSlotsByCycle(_ context.Context, _ string) ([]store.BudgetSlot, error) { return nil, nil }
func (m *mockStore) AssignSlot(_ context.Context, _, _ string) error                   { return nil }
func (m *mockStore) CreateAssignment(_ context.Context, _ store.TalentAssignment) error { return nil }
func (m *mockStore) GetAssignment(_ context.Context, _, _ string) (store.TalentAssignment, error) { return store.TalentAssignment{}, nil }
func (m *mockStore) ListAssignedTalents(_ context.Context, _ string) ([]store.TalentAssignment, error) { return nil, nil }
func (m *mockStore) ListAssignmentsByTalent(_ context.Context, _ string) ([]store.TalentAssignment, error) { return nil, nil }
func (m *mockStore) UpdateAssignment(_ context.Context, _, _ string, _ store.AssignmentPatch) error { return nil }
func (m *mockStore) CreateTrackingLink(_ context.Context, _ store.TrackingLink) error  { return nil }
func (m *mockStore) GetTrackingLinkByToken(_ context.Context, _ string) (store.TrackingLink, error) { return store.TrackingLink{}, nil }
func (m *mockStore) ListTrackingLinksByCycle(_ context.Context, _ string) ([]store.TrackingLink, error) { return nil, nil }
func (m *mockStore) LogConversionEvent(_ context.Context, _ store.ConversionEvent) error { return nil }
func (m *mockStore) GetDailyConversionCount(_ context.Context, _, _ string, _ time.Time) (float64, error) { return 0, nil }
func (m *mockStore) GetTotalConversions(_ context.Context, _ string) (float64, error)  { return 0, nil }
func (m *mockStore) GetTalentConversions(_ context.Context, _, _ string) (float64, error) { return 0, nil }
func (m *mockStore) FlagFallbackConversions(_ context.Context, _ string, _ time.Time) error { return nil }
func (m *mockStore) LockFallbackConversions(_ context.Context, _ string) error         { return nil }
func (m *mockStore) GetCycleState(_ context.Context, _, _ string) (store.CycleState, error) { return store.CycleState{}, nil }
func (m *mockStore) UpsertCycleState(_ context.Context, _ store.CycleState) error     { return nil }
func (m *mockStore) GetDailyOutputs(_ context.Context, _, _ string) ([]float64, error) { return nil, nil }
func (m *mockStore) GetTalentBaseline(_ context.Context, _ string) (store.TalentBaseline, error) { return store.TalentBaseline{}, nil }
func (m *mockStore) UpsertTalentBaseline(_ context.Context, _ store.TalentBaseline) error { return nil }
func (m *mockStore) GetCategoryBaseline(_ context.Context, _ string) (algo.CategoryBaseline, error) { return algo.CategoryBaseline{}, nil }
func (m *mockStore) GetTalentTodayOutput(_ context.Context, _ string, _ time.Time) (float64, error) { return 0, nil }
func (m *mockStore) GetTalentOutputWindow(_ context.Context, _ string, _ int) ([]float64, error) { return nil, nil }
func (m *mockStore) CreatePayoutRecord(_ context.Context, _ store.PayoutRecord) error  { return nil }
func (m *mockStore) GetPayoutRecord(_ context.Context, _, _ string) (store.PayoutRecord, error) { return store.PayoutRecord{}, nil }
func (m *mockStore) ListPayoutsByCycle(_ context.Context, _ string) ([]store.PayoutRecord, error) { return nil, nil }
func (m *mockStore) ListPayoutsByTalent(_ context.Context, _ string) ([]store.PayoutRecord, error) { return nil, nil }
func (m *mockStore) UpdatePayoutRecord(_ context.Context, _ string, _ store.PayoutPatch) error { return nil }
func (m *mockStore) CreateViewer(_ context.Context, _ store.CampaignViewer) error      { return nil }
func (m *mockStore) GetViewerByToken(_ context.Context, _ string) (store.CampaignViewer, error) { return store.CampaignViewer{}, nil }
func (m *mockStore) ListViewersByCampaign(_ context.Context, _ string) ([]store.CampaignViewer, error) { return nil, nil }
func (m *mockStore) UpdateViewer(_ context.Context, _ string, _ store.ViewerPatch) error { return nil }
func (m *mockStore) AddViewerPassword(_ context.Context, _ store.ViewerPassword) error { return nil }
func (m *mockStore) ListViewerPasswords(_ context.Context, _ string) ([]store.ViewerPassword, error) { return nil, nil }
func (m *mockStore) DeactivateViewerPassword(_ context.Context, _ string) error        { return nil }
func (m *mockStore) ValidateViewerPassword(_ context.Context, _, _ string) (bool, error) { return false, nil }

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestAddContact_GeneratesCredentials(t *testing.T) {
	svc := brand.New(newMock(), nil, nil)
	contact, plain, err := svc.AddContact(context.Background(), "brand-1", "Ada", "Obi", "CMO", "ada@test.com", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contact.Viewer_token == "" {
		t.Error("viewer_token should not be empty")
	}
	if contact.Access_password_hash == "" {
		t.Error("access_password_hash should not be empty")
	}
	if plain == "" {
		t.Error("plain password should be returned once")
	}
}

func TestAddContact_MaxContactsEnforced(t *testing.T) {
	ms := newMock()
	// pre-seed 3 active contacts
	for i := 0; i < 3; i++ {
		token := string(rune('a' + i))
		ms.contacts[token] = store.BrandContact{Brand_id: "brand-1", Token_active: true, Viewer_token: token}
	}
	svc := brand.New(ms, nil, nil)
	_, _, err := svc.AddContact(context.Background(), "brand-1", "X", "Y", "role", "x@y.com", "")
	if err == nil || err.Error() != "max_contacts_reached" {
		t.Errorf("expected max_contacts_reached, got %v", err)
	}
}

func TestRegeneratePassword_OldHashReplaced(t *testing.T) {
	ms := newMock()
	ms.contacts["tok1"] = store.BrandContact{ID: "c1", Brand_id: "brand-1", Viewer_token: "tok1", Access_password_hash: "old_hash", Token_active: true}
	svc := brand.New(ms, nil, nil)
	plain, err := svc.RegeneratePassword(context.Background(), "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plain == "" {
		t.Error("plain password should not be empty")
	}
	if !ms.regen_called {
		t.Error("RegenerateBrandContactPassword should have been called")
	}
	if ms.regen_new_hash == "old_hash" {
		t.Error("hash should have changed")
	}
}

func TestRemoveContact_SoftDelete(t *testing.T) {
	ms := newMock()
	ms.contacts["tok2"] = store.BrandContact{ID: "c2", Brand_id: "brand-1", Viewer_token: "tok2", Token_active: true}
	svc := brand.New(ms, nil, nil)
	if err := svc.RemoveContact(context.Background(), "c2", "actor-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms.contacts["tok2"].Token_active {
		t.Error("token_active should be false after removal")
	}
	if len(ms.audit_log) == 0 {
		t.Error("audit log should have an entry")
	}
}

func TestCreate_WithLogo(t *testing.T) {
	ms := newMock()
	up := &media.RecordingUploader{Result: media.UploadResult{URL: "https://ik.imagekit.io/test/logo.png"}}
	svc := brand.New(ms, up, nil)
	png := []byte{0x89, 0x50, 0x4e, 0x47}
	b, err := svc.Create(context.Background(), "Acme", "fintech", "", "https://acme.test", &brand.LogoFile{
		Body:        bytes.NewReader(png),
		ContentType: "image/png",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Logo_url != "https://ik.imagekit.io/test/logo.png" {
		t.Fatalf("logo_url=%q", b.Logo_url)
	}
	if len(up.Calls) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(up.Calls))
	}
	if !strings.HasPrefix(up.Calls[0].Folder, "/scaloo/brands/") {
		t.Fatalf("folder=%q", up.Calls[0].Folder)
	}
}

func TestCreate_LogoUploadFailureKeepsBrand(t *testing.T) {
	ms := newMock()
	up := &media.RecordingUploader{Err: errors.New("ik_down")}
	svc := brand.New(ms, up, nil)
	b, err := svc.Create(context.Background(), "Beta", "retail", "", "", &brand.LogoFile{
		Body:        bytes.NewReader([]byte("x")),
		ContentType: "image/jpeg",
	})
	if err == nil || !errors.Is(err, brand.ErrLogoUploadFailed) {
		t.Fatalf("expected logo_upload_failed, got %v", err)
	}
	if b.Shortcode == "" {
		t.Fatal("brand should still be returned after logo failure")
	}
	if _, ok := ms.brands[b.Shortcode]; !ok {
		t.Fatal("brand should remain in store")
	}
}


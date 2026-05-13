package auth_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/src/auth"
	"github.com/Ze-uus/talent-backend/internal/store"
)

// ─── Mock store ───────────────────────────────────────────────────────────────

type mockStore struct {
	mu       sync.Mutex
	users    map[string]store.User    // keyed by ID
	sessions map[string]store.Session // keyed by token

	// index helpers
	byEmail  map[string]string // email → id
	byGoogle map[string]string // google_id → id
	byInvite map[string]string // invite_token → id
	byViewer map[string]string // viewer_token → brand_contact.id

	brand_contacts map[string]store.BrandContact // keyed by id
	talents        map[string]store.Talent        // keyed by user_id
}

func newMock() *mockStore {
	return &mockStore{
		users:          make(map[string]store.User),
		sessions:       make(map[string]store.Session),
		byEmail:        make(map[string]string),
		byGoogle:       make(map[string]string),
		byInvite:       make(map[string]string),
		byViewer:       make(map[string]string),
		brand_contacts: make(map[string]store.BrandContact),
		talents:        make(map[string]store.Talent),
	}
}

func (m *mockStore) addUser(u store.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[u.ID] = u
	m.byEmail[u.Email] = u.ID
	if u.Google_id != "" {
		m.byGoogle[u.Google_id] = u.ID
	}
	if u.Invite_token != "" {
		m.byInvite[u.Invite_token] = u.ID
	}
}

func (m *mockStore) addTalent(t store.Talent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.talents[t.User_id] = t
}

func (m *mockStore) addBrandContact(c store.BrandContact) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.brand_contacts[c.ID] = c
	m.byViewer[c.Viewer_token] = c.ID
}

// ─── Store interface implementation ──────────────────────────────────────────

func (m *mockStore) CreateUser(ctx context.Context, u store.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u.ID == "" {
		u.ID = "u_" + u.Email
	}
	m.users[u.ID] = u
	m.byEmail[u.Email] = u.ID
	if u.Invite_token != "" {
		m.byInvite[u.Invite_token] = u.ID
	}
	return nil
}

func (m *mockStore) GetUserByID(ctx context.Context, id string) (store.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return store.User{}, errors.New("not_found")
	}
	return u, nil
}

func (m *mockStore) GetUserByEmail(ctx context.Context, email string) (store.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byEmail[email]
	if !ok {
		return store.User{}, errors.New("not_found")
	}
	return m.users[id], nil
}

func (m *mockStore) GetUserByGoogleID(ctx context.Context, google_id string) (store.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byGoogle[google_id]
	if !ok {
		return store.User{}, errors.New("not_found")
	}
	return m.users[id], nil
}

func (m *mockStore) GetUserByInviteToken(ctx context.Context, token string) (store.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byInvite[token]
	if !ok {
		return store.User{}, errors.New("not_found")
	}
	return m.users[id], nil
}

func (m *mockStore) UpdateUser(ctx context.Context, id string, patch store.UserPatch) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return errors.New("not_found")
	}
	if patch.Full_name != nil {
		u.Full_name = *patch.Full_name
	}
	if patch.Avatar_url != nil {
		u.Avatar_url = *patch.Avatar_url
	}
	if patch.Password_hash != nil {
		u.Password_hash = *patch.Password_hash
	}
	if patch.Totp_secret != nil {
		u.Totp_secret = *patch.Totp_secret
	}
	if patch.Totp_enabled != nil {
		u.Totp_enabled = *patch.Totp_enabled
	}
	if patch.Totp_verified != nil {
		u.Totp_verified = *patch.Totp_verified
	}
	if patch.Totp_last_verified_at != nil {
		u.Totp_last_verified_at = *patch.Totp_last_verified_at
	}
	if patch.Invite_token != nil {
		old_token := u.Invite_token
		u.Invite_token = *patch.Invite_token
		delete(m.byInvite, old_token)
		if u.Invite_token != "" {
			m.byInvite[u.Invite_token] = id
		}
	}
	if patch.Invite_expires_at != nil {
		u.Invite_expires_at = *patch.Invite_expires_at
	}
	if patch.Active != nil {
		u.Active = *patch.Active
	}
	m.users[id] = u
	return nil
}

func (m *mockStore) ListUsers(ctx context.Context, filter store.UserFilter) ([]store.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []store.User
	for _, u := range m.users {
		if filter.Role != "" && string(u.Role) != filter.Role {
			continue
		}
		if filter.Active != nil && u.Active != *filter.Active {
			continue
		}
		out = append(out, u)
	}
	return out, nil
}

func (m *mockStore) CreateSession(ctx context.Context, s store.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.Token] = s
	return nil
}

func (m *mockStore) GetSession(ctx context.Context, token string) (store.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[token]
	if !ok {
		return store.Session{}, errors.New("not_found")
	}
	return s, nil
}

func (m *mockStore) TouchSession(ctx context.Context, token string, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[token]
	if !ok {
		return errors.New("not_found")
	}
	s.Last_active_at = now
	m.sessions[token] = s
	return nil
}

func (m *mockStore) InvalidateSession(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[token]
	if !ok {
		return errors.New("not_found")
	}
	s.Invalidated = true
	m.sessions[token] = s
	return nil
}

func (m *mockStore) InvalidateAllUserSessions(_ context.Context, _ string) error {
	return errors.New("not_implemented")
}

func (m *mockStore) GetTalentByUserID(ctx context.Context, user_id string) (store.Talent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.talents[user_id]
	if !ok {
		return store.Talent{}, errors.New("not_found")
	}
	return t, nil
}

func (m *mockStore) GetBrandContactByViewerToken(ctx context.Context, token string) (store.BrandContact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byViewer[token]
	if !ok {
		return store.BrandContact{}, errors.New("not_found")
	}
	return m.brand_contacts[id], nil
}

func (m *mockStore) GetViewerByToken(ctx context.Context, token string) (store.CampaignViewer, error) {
	return store.CampaignViewer{}, errors.New("not_implemented")
}

func (m *mockStore) ValidateViewerPassword(ctx context.Context, viewer_id, password string) (bool, error) {
	return false, errors.New("not_implemented")
}

func (m *mockStore) Ping(_ context.Context) error { return nil }

// ─── Remaining Store methods (not needed by auth service — return not_implemented) ─

func (m *mockStore) CreateBrand(_ context.Context, _ store.Brand) error {
	return errors.New("not_implemented")
}
func (m *mockStore) GetBrandByID(_ context.Context, _ string) (store.Brand, error) {
	return store.Brand{}, errors.New("not_implemented")
}
func (m *mockStore) GetBrandByShortcode(_ context.Context, _ string) (store.Brand, error) {
	return store.Brand{}, errors.New("not_implemented")
}
func (m *mockStore) ListBrands(_ context.Context, _ store.BrandFilter) ([]store.Brand, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) UpdateBrand(_ context.Context, _ string, _ store.BrandPatch) error {
	return errors.New("not_implemented")
}
func (m *mockStore) ShortcodeExists(_ context.Context, _ string) (bool, error) {
	return false, errors.New("not_implemented")
}
func (m *mockStore) CreateBrandContact(_ context.Context, _ store.BrandContact) error {
	return errors.New("not_implemented")
}
func (m *mockStore) GetBrandContactByID(_ context.Context, _ string) (store.BrandContact, error) {
	return store.BrandContact{}, errors.New("not_implemented")
}
func (m *mockStore) ListBrandContacts(_ context.Context, _ string) ([]store.BrandContact, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) UpdateBrandContact(_ context.Context, _ string, _ store.BrandContactPatch) error {
	return errors.New("not_implemented")
}
func (m *mockStore) DeactivateBrandContact(_ context.Context, _ string) error {
	return errors.New("not_implemented")
}
func (m *mockStore) RegenerateBrandContactPassword(_ context.Context, _ string, _ string) error {
	return errors.New("not_implemented")
}
func (m *mockStore) ValidateBrandContactAccess(_ context.Context, _, _ string) (store.BrandContact, error) {
	return store.BrandContact{}, errors.New("not_implemented")
}
func (m *mockStore) CreateTalent(_ context.Context, _ store.Talent) error {
	return errors.New("not_implemented")
}
func (m *mockStore) GetTalentByID(_ context.Context, _ string) (store.Talent, error) {
	return store.Talent{}, errors.New("not_implemented")
}
func (m *mockStore) ListTalents(_ context.Context, _ store.TalentFilter) ([]store.Talent, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) UpdateTalent(_ context.Context, _ string, _ store.TalentPatch) error {
	return errors.New("not_implemented")
}
func (m *mockStore) ListAllTalents(_ context.Context) ([]store.Talent, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) CreateCampaign(_ context.Context, _ store.Campaign) error {
	return errors.New("not_implemented")
}
func (m *mockStore) GetCampaignByID(_ context.Context, _ string) (store.Campaign, error) {
	return store.Campaign{}, errors.New("not_implemented")
}
func (m *mockStore) GetCampaignByHumanID(_ context.Context, _ string) (store.Campaign, error) {
	return store.Campaign{}, errors.New("not_implemented")
}
func (m *mockStore) ListCampaigns(_ context.Context, _ store.CampaignFilter) ([]store.Campaign, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) ListActiveCampaigns(_ context.Context) ([]store.Campaign, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) UpdateCampaign(_ context.Context, _ string, _ store.CampaignPatch) error {
	return errors.New("not_implemented")
}
func (m *mockStore) DecrementRemainingBudget(_ context.Context, _ string, _ float64) error {
	return errors.New("not_implemented")
}
func (m *mockStore) NextCampaignHumanID(_ context.Context, _ string) (string, error) {
	return "", errors.New("not_implemented")
}
func (m *mockStore) GetCampaignsByManagerID(_ context.Context, _ string) ([]store.Campaign, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) AssignManagerToCampaign(_ context.Context, _, _, _ string) error {
	return errors.New("not_implemented")
}
func (m *mockStore) UnassignManagerFromCampaign(_ context.Context, _, _ string) error {
	return errors.New("not_implemented")
}
func (m *mockStore) CreateCycle(_ context.Context, _ store.Cycle) error {
	return errors.New("not_implemented")
}
func (m *mockStore) GetCycleByID(_ context.Context, _ string) (store.Cycle, error) {
	return store.Cycle{}, errors.New("not_implemented")
}
func (m *mockStore) ListCyclesByCampaign(_ context.Context, _ string) ([]store.Cycle, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) GetActiveCycles(_ context.Context) ([]store.Cycle, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) UpdateCycle(_ context.Context, _ string, _ store.CyclePatch) error {
	return errors.New("not_implemented")
}
func (m *mockStore) CloseCycle(_ context.Context, _ string, _ float64) error {
	return errors.New("not_implemented")
}
func (m *mockStore) CreateBudgetSlots(_ context.Context, _ []store.BudgetSlot) error {
	return errors.New("not_implemented")
}
func (m *mockStore) ListSlotsByCycle(_ context.Context, _ string) ([]store.BudgetSlot, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) AssignSlot(_ context.Context, _, _ string) error {
	return errors.New("not_implemented")
}
func (m *mockStore) CreateAssignment(_ context.Context, _ store.TalentAssignment) error {
	return errors.New("not_implemented")
}
func (m *mockStore) GetAssignment(_ context.Context, _, _ string) (store.TalentAssignment, error) {
	return store.TalentAssignment{}, errors.New("not_implemented")
}
func (m *mockStore) ListAssignedTalents(_ context.Context, _ string) ([]store.TalentAssignment, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) UpdateAssignment(_ context.Context, _, _ string, _ store.AssignmentPatch) error {
	return errors.New("not_implemented")
}
func (m *mockStore) CreateTrackingLink(_ context.Context, _ store.TrackingLink) error {
	return errors.New("not_implemented")
}
func (m *mockStore) GetTrackingLinkByToken(_ context.Context, _ string) (store.TrackingLink, error) {
	return store.TrackingLink{}, errors.New("not_implemented")
}
func (m *mockStore) ListTrackingLinksByCycle(_ context.Context, _ string) ([]store.TrackingLink, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) LogConversionEvent(_ context.Context, _ store.ConversionEvent) error {
	return errors.New("not_implemented")
}
func (m *mockStore) GetDailyConversionCount(_ context.Context, _, _ string, _ time.Time) (float64, error) {
	return 0, errors.New("not_implemented")
}
func (m *mockStore) GetTotalConversions(_ context.Context, _ string) (float64, error) {
	return 0, errors.New("not_implemented")
}
func (m *mockStore) GetTalentConversions(_ context.Context, _, _ string) (float64, error) {
	return 0, errors.New("not_implemented")
}
func (m *mockStore) FlagFallbackConversions(_ context.Context, _ string, _ time.Time) error {
	return errors.New("not_implemented")
}
func (m *mockStore) LockFallbackConversions(_ context.Context, _ string) error {
	return errors.New("not_implemented")
}
func (m *mockStore) GetCycleState(_ context.Context, _, _ string) (store.CycleState, error) {
	return store.CycleState{}, errors.New("not_implemented")
}
func (m *mockStore) UpsertCycleState(_ context.Context, _ store.CycleState) error {
	return errors.New("not_implemented")
}
func (m *mockStore) GetDailyOutputs(_ context.Context, _, _ string) ([]float64, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) GetTalentBaseline(_ context.Context, _ string) (store.TalentBaseline, error) {
	return store.TalentBaseline{}, errors.New("not_implemented")
}
func (m *mockStore) UpsertTalentBaseline(_ context.Context, _ store.TalentBaseline) error {
	return errors.New("not_implemented")
}
func (m *mockStore) GetCategoryBaseline(_ context.Context, _ string) (algo.CategoryBaseline, error) {
	return algo.CategoryBaseline{}, errors.New("not_implemented")
}
func (m *mockStore) GetTalentTodayOutput(_ context.Context, _ string, _ time.Time) (float64, error) {
	return 0, errors.New("not_implemented")
}
func (m *mockStore) GetTalentOutputWindow(_ context.Context, _ string, _ int) ([]float64, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) CreatePayoutRecord(_ context.Context, _ store.PayoutRecord) error {
	return errors.New("not_implemented")
}
func (m *mockStore) GetPayoutRecord(_ context.Context, _, _ string) (store.PayoutRecord, error) {
	return store.PayoutRecord{}, errors.New("not_implemented")
}
func (m *mockStore) ListPayoutsByCycle(_ context.Context, _ string) ([]store.PayoutRecord, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) UpdatePayoutRecord(_ context.Context, _ string, _ store.PayoutPatch) error {
	return errors.New("not_implemented")
}
func (m *mockStore) CreateViewer(_ context.Context, _ store.CampaignViewer) error {
	return errors.New("not_implemented")
}
func (m *mockStore) ListViewersByCampaign(_ context.Context, _ string) ([]store.CampaignViewer, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) UpdateViewer(_ context.Context, _ string, _ store.ViewerPatch) error {
	return errors.New("not_implemented")
}
func (m *mockStore) AddViewerPassword(_ context.Context, _ store.ViewerPassword) error {
	return errors.New("not_implemented")
}
func (m *mockStore) ListViewerPasswords(_ context.Context, _ string) ([]store.ViewerPassword, error) {
	return nil, errors.New("not_implemented")
}
func (m *mockStore) DeactivateViewerPassword(_ context.Context, _ string) error {
	return errors.New("not_implemented")
}
func (m *mockStore) WriteAuditLog(_ context.Context, _ store.AuditLog) error {
	return errors.New("not_implemented")
}
func (m *mockStore) ListAuditLog(_ context.Context, _, _ string) ([]store.AuditLog, error) {
	return nil, errors.New("not_implemented")
}

// ─── Test helpers ─────────────────────────────────────────────────────────────

func hashPassword(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	return string(h)
}

func newSvc(m *mockStore) *auth.AuthService {
	return auth.NewAuthService(m, "Scaloo")
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestLogin_SuperadminRequiresTOTP_AfterEnrollment(t *testing.T) {
	m := newMock()
	m.addUser(store.User{
		ID:            "sa1",
		Email:         "sa@scaloo.com",
		Password_hash: hashPassword(t, "pass"),
		Role:          store.Role_superadmin,
		Active:        true,
		Totp_enabled:  true,
		Totp_verified: true,
		Totp_secret:   generateTOTPSecret(t),
	})
	svc := newSvc(m)
	_, err := svc.Login(context.Background(), "sa@scaloo.com", "pass", "", "127.0.0.1", "test")
	if err == nil || err.Error() != "totp_required" {
		t.Fatalf("expected totp_required, got %v", err)
	}
}

func TestLogin_AdminRequiresTOTP_AfterEnrollment(t *testing.T) {
	m := newMock()
	m.addUser(store.User{
		ID:            "a1",
		Email:         "admin@scaloo.com",
		Password_hash: hashPassword(t, "pass"),
		Role:          store.Role_admin,
		Active:        true,
		Totp_enabled:  true,
		Totp_verified: true,
		Totp_secret:   generateTOTPSecret(t),
	})
	svc := newSvc(m)
	_, err := svc.Login(context.Background(), "admin@scaloo.com", "pass", "", "127.0.0.1", "test")
	if err == nil || err.Error() != "totp_required" {
		t.Fatalf("expected totp_required, got %v", err)
	}
}

func TestLogin_CampaignManagerRequiresTOTP_AfterEnrollment(t *testing.T) {
	m := newMock()
	secret := generateTOTPSecret(t)
	m.addUser(store.User{
		ID:            "cm1",
		Email:         "cm@scaloo.com",
		Password_hash: hashPassword(t, "pass"),
		Role:          store.Role_campaign_manager,
		Active:        true,
		Totp_enabled:  true,
		Totp_verified: true,
		Totp_secret:   secret,
	})
	svc := newSvc(m)
	_, err := svc.Login(context.Background(), "cm@scaloo.com", "pass", "", "127.0.0.1", "test")
	if err == nil || err.Error() != "totp_required" {
		t.Fatalf("expected totp_required, got %v", err)
	}
}

func TestLogin_AdminAllowed_BeforeEnrollment(t *testing.T) {
	m := newMock()
	m.addUser(store.User{
		ID:            "a2",
		Email:         "admin2@scaloo.com",
		Password_hash: hashPassword(t, "pass"),
		Role:          store.Role_admin,
		Active:        true,
		Totp_enabled:  false,
		Totp_verified: false,
	})
	svc := newSvc(m)
	result, err := svc.Login(context.Background(), "admin2@scaloo.com", "pass", "", "", "")
	if err != nil {
		t.Fatalf("expected no error before TOTP enrollment, got %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected session token to be set")
	}
}

func TestLogin_TalentNoTOTP_ReturnsSetupFlag(t *testing.T) {
	m := newMock()
	m.addUser(store.User{
		ID:            "t1",
		Email:         "talent@example.com",
		Password_hash: hashPassword(t, "pass"),
		Role:          store.Role_talent,
		Active:        true,
		Totp_verified: false,
	})
	m.addTalent(store.Talent{User_id: "t1", Status: store.Status_active})
	svc := newSvc(m)
	result, err := svc.Login(context.Background(), "talent@example.com", "pass", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Require_totp_setup {
		t.Fatal("expected require_totp_setup=true for talent without TOTP")
	}
}

func TestLogin_Talent72hRecheckDue_RequiresTOTP(t *testing.T) {
	m := newMock()
	secret := generateTOTPSecret(t)
	m.addUser(store.User{
		ID:                    "t2",
		Email:                 "talent2@example.com",
		Password_hash:         hashPassword(t, "pass"),
		Role:                  store.Role_talent,
		Active:                true,
		Totp_verified:         true,
		Totp_secret:           secret,
		Totp_last_verified_at: time.Now().Add(-73 * time.Hour),
	})
	m.addTalent(store.Talent{User_id: "t2", Status: store.Status_active})
	svc := newSvc(m)
	_, err := svc.Login(context.Background(), "talent2@example.com", "pass", "", "", "")
	if err == nil || err.Error() != "totp_required" {
		t.Fatalf("expected totp_required after 73h, got %v", err)
	}
}

func TestLogin_Talent72hRecheckDue_WithCode_Passes(t *testing.T) {
	m := newMock()
	secret := generateTOTPSecret(t)
	m.addUser(store.User{
		ID:                    "t3",
		Email:                 "talent3@example.com",
		Password_hash:         hashPassword(t, "pass"),
		Role:                  store.Role_talent,
		Active:                true,
		Totp_verified:         true,
		Totp_secret:           secret,
		Totp_last_verified_at: time.Now().Add(-73 * time.Hour),
	})
	m.addTalent(store.Talent{User_id: "t3", Status: store.Status_active})
	svc := newSvc(m)
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	result, err := svc.Login(context.Background(), "talent3@example.com", "pass", code, "", "")
	if err != nil {
		t.Fatalf("expected success with valid TOTP code: %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected session token")
	}
}

func TestLogin_Talent72hNotElapsed_NoCodeNeeded(t *testing.T) {
	m := newMock()
	secret := generateTOTPSecret(t)
	m.addUser(store.User{
		ID:                    "t4",
		Email:                 "talent4@example.com",
		Password_hash:         hashPassword(t, "pass"),
		Role:                  store.Role_talent,
		Active:                true,
		Totp_verified:         true,
		Totp_secret:           secret,
		Totp_last_verified_at: time.Now().Add(-1 * time.Hour),
	})
	m.addTalent(store.Talent{User_id: "t4", Status: store.Status_active})
	svc := newSvc(m)
	result, err := svc.Login(context.Background(), "talent4@example.com", "pass", "", "", "")
	if err != nil {
		t.Fatalf("expected success within 72h window: %v", err)
	}
	if result.Token == "" {
		t.Fatal("expected session token")
	}
}

func TestLogin_SuperadminSession_8hTTL(t *testing.T) {
	m := newMock()
	m.addUser(store.User{
		ID:            "sa2",
		Email:         "sa2@scaloo.com",
		Password_hash: hashPassword(t, "pass"),
		Role:          store.Role_superadmin,
		Active:        true,
		Totp_enabled:  false,
		Totp_verified: false,
	})
	svc := newSvc(m)
	before := time.Now()
	result, err := svc.Login(context.Background(), "sa2@scaloo.com", "pass", "", "", "")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	sess, err := m.GetSession(context.Background(), result.Token)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	want := before.Add(8 * time.Hour)
	diff := sess.Expires_at.Sub(want)
	if diff > 5*time.Second || diff < -5*time.Second {
		t.Fatalf("expected ~8h TTL, got %v", sess.Expires_at.Sub(before))
	}
}

func TestLogin_AdminSession_8hTTL(t *testing.T) {
	m := newMock()
	m.addUser(store.User{
		ID:            "a3",
		Email:         "admin3@scaloo.com",
		Password_hash: hashPassword(t, "pass"),
		Role:          store.Role_admin,
		Active:        true,
		Totp_enabled:  false,
		Totp_verified: false,
	})
	svc := newSvc(m)
	before := time.Now()
	result, err := svc.Login(context.Background(), "admin3@scaloo.com", "pass", "", "", "")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	sess, _ := m.GetSession(context.Background(), result.Token)
	diff := sess.Expires_at.Sub(before.Add(8 * time.Hour))
	if diff > 5*time.Second || diff < -5*time.Second {
		t.Fatalf("expected ~8h TTL, diff=%v", diff)
	}
}

func TestLogin_CampaignManagerSession_8hTTL(t *testing.T) {
	m := newMock()
	m.addUser(store.User{
		ID:            "cm2",
		Email:         "cm2@scaloo.com",
		Password_hash: hashPassword(t, "pass"),
		Role:          store.Role_campaign_manager,
		Active:        true,
		Totp_enabled:  false,
		Totp_verified: false,
	})
	svc := newSvc(m)
	before := time.Now()
	result, err := svc.Login(context.Background(), "cm2@scaloo.com", "pass", "", "", "")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	sess, _ := m.GetSession(context.Background(), result.Token)
	diff := sess.Expires_at.Sub(before.Add(8 * time.Hour))
	if diff > 5*time.Second || diff < -5*time.Second {
		t.Fatalf("expected ~8h TTL, diff=%v", diff)
	}
}

func TestLogin_TalentSession_7dTTL(t *testing.T) {
	m := newMock()
	m.addUser(store.User{
		ID:            "t5",
		Email:         "talent5@example.com",
		Password_hash: hashPassword(t, "pass"),
		Role:          store.Role_talent,
		Active:        true,
		Totp_verified: false,
	})
	m.addTalent(store.Talent{User_id: "t5", Status: store.Status_active})
	svc := newSvc(m)
	before := time.Now()
	result, err := svc.Login(context.Background(), "talent5@example.com", "pass", "", "", "")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	sess, _ := m.GetSession(context.Background(), result.Token)
	diff := sess.Expires_at.Sub(before.Add(7 * 24 * time.Hour))
	if diff > 5*time.Second || diff < -5*time.Second {
		t.Fatalf("expected ~7d TTL, diff=%v", diff)
	}
}

func TestVerifyInvite_ExpiredToken(t *testing.T) {
	m := newMock()
	m.addUser(store.User{
		ID:                "inv1",
		Email:             "invited@scaloo.com",
		Role:              store.Role_admin,
		Active:            false,
		Invite_token:      "expired-tok",
		Invite_expires_at: time.Now().Add(-1 * time.Hour), // past
	})
	svc := newSvc(m)
	_, err := svc.VerifyInvite(context.Background(), "expired-tok", "newpass")
	if err == nil || err.Error() != "invalid_or_expired_invite" {
		t.Fatalf("expected invalid_or_expired_invite, got %v", err)
	}
}

func TestValidateBrandContactAccess_Valid(t *testing.T) {
	m := newMock()
	plain := "secret123pass"
	hash, _ := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	m.addBrandContact(store.BrandContact{
		ID:                   "bc1",
		Viewer_token:         "tok123",
		Access_password_hash: string(hash),
		Token_active:         true,
	})
	svc := newSvc(m)
	contact, err := svc.ValidateBrandContactAccess(context.Background(), "tok123", plain)
	if err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if contact.ID != "bc1" {
		t.Fatalf("expected bc1, got %s", contact.ID)
	}
}

func TestValidateBrandContactAccess_WrongPassword(t *testing.T) {
	m := newMock()
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	m.addBrandContact(store.BrandContact{
		ID:                   "bc2",
		Viewer_token:         "tok456",
		Access_password_hash: string(hash),
		Token_active:         true,
	})
	svc := newSvc(m)
	_, err := svc.ValidateBrandContactAccess(context.Background(), "tok456", "wrong")
	if err == nil || err.Error() != "invalid_viewer_credentials" {
		t.Fatalf("expected invalid_viewer_credentials, got %v", err)
	}
}

func TestValidateBrandContactAccess_InactiveToken(t *testing.T) {
	m := newMock()
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.MinCost)
	m.addBrandContact(store.BrandContact{
		ID:                   "bc3",
		Viewer_token:         "tok789",
		Access_password_hash: string(hash),
		Token_active:         false, // deactivated
	})
	svc := newSvc(m)
	_, err := svc.ValidateBrandContactAccess(context.Background(), "tok789", "pass")
	if err == nil || err.Error() != "invalid_viewer_credentials" {
		t.Fatalf("expected invalid_viewer_credentials for inactive token, got %v", err)
	}
}

// ─── Helper ───────────────────────────────────────────────────────────────────

func generateTOTPSecret(t *testing.T) string {
	t.Helper()
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "test", AccountName: "test@test.com"})
	if err != nil {
		t.Fatalf("generate totp key: %v", err)
	}
	return key.Secret()
}

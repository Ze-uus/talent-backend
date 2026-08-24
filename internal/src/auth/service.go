package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/Ze-uus/talent-backend/internal/mail"
	"github.com/Ze-uus/talent-backend/internal/store"
)

const (
	admin_session_duration  = 8 * time.Hour // superadmin + admin + campaign_manager
	talent_session_duration = 7 * 24 * time.Hour
	viewer_cookie_ttl       = 4 * time.Hour
	inactivity_timeout      = 30 * time.Minute
	totp_recheck_interval   = 72 * time.Hour
	bcrypt_cost             = 12
	invite_ttl              = 48 * time.Hour
)

var (
	err_invalid_credentials = errors.New("invalid_credentials")
	err_totp_required       = errors.New("totp_required")
	err_invalid_totp        = errors.New("invalid_totp_code")
	err_session_expired     = errors.New("session_expired")
	err_session_inactive    = errors.New("session_inactive")
	err_account_pending     = errors.New("account_pending_approval")
	err_account_suspended   = errors.New("account_suspended")
	err_account_banned      = errors.New("account_banned")
	err_account_deleted     = errors.New("account_deleted")
	err_account_rejected    = errors.New("account_rejected")
	err_account_invited     = errors.New("account_invited")
	err_invalid_invite      = errors.New("invalid_or_expired_invite")
	err_viewer_invalid      = errors.New("invalid_viewer_credentials")
	err_bootstrap_exists    = errors.New("superadmin_already_exists")
)

type AuthService struct {
	store       store.Store
	issuer      string // "Scaloo"
	google_conf *oauth2.Config
	mail        *mail.Service
}

func NewAuthService(s store.Store, issuer string, mailSvc *mail.Service) *AuthService {
	return &AuthService{store: s, issuer: issuer, mail: mailSvc}
}

// WithGoogleOAuth adds Google OAuth support to the service.
// Call this at startup if GOOGLE_CLIENT_ID/SECRET are set.
func (a *AuthService) WithGoogleOAuth(client_id, client_secret, redirect_url string) {
	a.google_conf = &oauth2.Config{
		ClientID:     client_id,
		ClientSecret: client_secret,
		RedirectURL:  redirect_url,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

// ─── Bootstrap ────────────────────────────────────────────────────────────────

// BootstrapSuperAdmin creates the first superadmin from env vars if none exists yet.
// Reads SCALOO_BOOTSTRAP_EMAIL and SCALOO_BOOTSTRAP_PASSWORD.
// Returns err_bootstrap_exists if a superadmin already exists — safe to call on every startup.
func (a *AuthService) BootstrapSuperAdmin(ctx context.Context) error {
	email := os.Getenv("SCALOO_BOOTSTRAP_EMAIL")
	password := os.Getenv("SCALOO_BOOTSTRAP_PASSWORD")
	if email == "" || password == "" {
		return nil // bootstrap vars not set; skip
	}
	users, err := a.store.ListUsers(ctx, store.UserFilter{Role: string(store.Role_superadmin)})
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return err_bootstrap_exists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt_cost)
	if err != nil {
		return err
	}
	return a.store.CreateUser(ctx, store.User{
		Email:         email,
		Password_hash: string(hash),
		Full_name:     "Superadmin",
		Role:          store.Role_superadmin,
		Provider:      store.Provider_local,
		Active:        true,
		Status:        store.User_status_active,
	})
}

// ─── Invite flows ─────────────────────────────────────────────────────────────

func (a *AuthService) InviteSuperAdmin(ctx context.Context, email, full_name string) (string, error) {
	return a.createInvite(ctx, email, full_name, store.Role_superadmin)
}

func (a *AuthService) InviteAdmin(ctx context.Context, email, full_name string) (string, error) {
	return a.createInvite(ctx, email, full_name, store.Role_admin)
}

func (a *AuthService) InviteCampaignManager(ctx context.Context, email, full_name string) (string, error) {
	return a.createInvite(ctx, email, full_name, store.Role_campaign_manager)
}

func (a *AuthService) createInvite(ctx context.Context, email, full_name string, role store.User_role) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	expires := time.Now().UTC().Add(invite_ttl)
	u := store.User{
		Email:             email,
		Full_name:         full_name,
		Role:              role,
		Provider:          store.Provider_local,
		Invite_token:      token,
		Invite_expires_at: expires,
		Active:            false,
		Status:            store.User_status_invited,
	}
	if err := a.store.CreateUser(ctx, u); err != nil {
		return "", err
	}
	if a.mail != nil {
		a.mail.NotifyStaffInvite(email, full_name, string(role), token)
	}
	return token, nil
}

// VerifyInvite sets the password from an invite token, activating the account.
// Unified for all staff roles (superadmin, admin, campaign_manager).
func (a *AuthService) VerifyInvite(ctx context.Context, invite_token, password string) (store.User, error) {
	u, err := a.store.GetUserByInviteToken(ctx, invite_token)
	if err != nil || time.Now().After(u.Invite_expires_at) {
		return store.User{}, err_invalid_invite
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt_cost)
	if err != nil {
		return store.User{}, err
	}
	empty := ""
	active := true
	status := string(store.User_status_active)
	_ = a.store.UpdateUser(ctx, u.ID, store.UserPatch{
		Password_hash: strPtr(string(hash)),
		Invite_token:  &empty,
		Active:        &active,
		Status:        &status,
	})
	return a.store.GetUserByID(ctx, u.ID)
}

// ─── Talent registration ──────────────────────────────────────────────────────

func (a *AuthService) RegisterTalent(ctx context.Context, email, password, full_name, phone_number string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt_cost)
	if err != nil {
		return err
	}
	if err := a.store.CreateUser(ctx, store.User{
		Email:         email,
		Password_hash: string(hash),
		Full_name:     full_name,
		Phone_number:  phone_number,
		Role:          store.Role_talent,
		Provider:      store.Provider_local,
		Active:        true,
		Status:        store.User_status_pending,
	}); err != nil {
		return err
	}
	user, err := a.store.GetUserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if err := a.createPendingTalent(ctx, user.ID); err != nil {
		return err
	}
	if a.mail != nil {
		a.mail.NotifyTalentRegistered(email, full_name)
	}
	return nil
}

func (a *AuthService) RegisterTalentGoogle(ctx context.Context, google_id, email, full_name, avatar_url string) (store.User, error) {
	u, err := a.store.GetUserByGoogleID(ctx, google_id)
	if err == nil {
		return u, nil
	}
	if err := a.store.CreateUser(ctx, store.User{
		Email:      email,
		Full_name:  full_name,
		Avatar_url: avatar_url,
		Role:       store.Role_talent,
		Provider:   store.Provider_google,
		Google_id:  google_id,
		Active:     true,
		Status:     store.User_status_pending,
	}); err != nil {
		return store.User{}, err
	}
	user, err := a.store.GetUserByEmail(ctx, email)
	if err != nil {
		return store.User{}, err
	}
	if err := a.createPendingTalent(ctx, user.ID); err != nil {
		return store.User{}, err
	}
	if a.mail != nil {
		a.mail.NotifyTalentRegistered(email, full_name)
	}
	return user, nil
}

// createPendingTalent inserts the talents row linked to a new talent user.
// Category is provisional (student) until an admin approves with the real category.
func (a *AuthService) createPendingTalent(ctx context.Context, user_id string) error {
	return a.store.CreateTalent(ctx, store.Talent{
		ID:                uuid.NewString(),
		User_id:           user_id,
		Category:          store.Category_student,
		Status:            store.Status_pending,
		Skills:            []string{},
		Report_compliance: 1.0,
	})
}

// ─── Login ────────────────────────────────────────────────────────────────────

// LoginResult carries the session token plus routing fields the frontend needs.
type LoginResult struct {
	Token              string
	User_id            string
	Role               store.User_role
	Status             store.User_status
	Active             bool
	Require_totp_setup bool // true when talent has not yet enrolled TOTP
	Totp_recheck_due   bool // true when 72h interval has passed and TOTP is needed
}

func statusGateError(status store.User_status) error {
	switch status {
	case store.User_status_deleted:
		return err_account_deleted
	case store.User_status_banned:
		return err_account_banned
	case store.User_status_suspended:
		return err_account_suspended
	case store.User_status_pending:
		return err_account_pending
	case store.User_status_rejected:
		return err_account_rejected
	case store.User_status_invited:
		return err_account_invited
	default:
		return nil
	}
}

func sessionBlocked(status store.User_status) bool {
	switch status {
	case store.User_status_suspended, store.User_status_banned,
		store.User_status_deleted, store.User_status_rejected,
		store.User_status_invited:
		return true
	default:
		return false
	}
}

// Login handles login for all roles.
// superadmin/admin/campaign_manager: TOTP always required after enrollment (no grace).
// talent: TOTP required only if 72h elapsed since last verification.
func (a *AuthService) Login(ctx context.Context, email, password, totp_code, ip, ua string) (LoginResult, error) {
	u, err := a.store.GetUserByEmail(ctx, email)
	if err != nil {
		return LoginResult{}, err_invalid_credentials
	}

	if err := statusGateError(u.Status); err != nil {
		return LoginResult{}, err
	}
	if !u.Active {
		return LoginResult{}, err_invalid_credentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.Password_hash), []byte(password)); err != nil {
		return LoginResult{}, err_invalid_credentials
	}

	result := LoginResult{
		User_id: u.ID,
		Role:    u.Role,
		Status:  u.Status,
		Active:  u.Active,
	}

	switch u.Role {
	case store.Role_superadmin, store.Role_admin, store.Role_campaign_manager:
		if err := a.checkStaffTOTP(u, totp_code); err != nil {
			return LoginResult{}, err
		}

	case store.Role_talent:
		totp_recheck := u.Totp_verified && time.Since(u.Totp_last_verified_at) > totp_recheck_interval
		result.Require_totp_setup = !u.Totp_verified
		result.Totp_recheck_due = totp_recheck

		if totp_recheck {
			if totp_code == "" {
				return LoginResult{}, err_totp_required
			}
			if !totp.Validate(totp_code, u.Totp_secret) {
				return LoginResult{}, err_invalid_totp
			}
			now := time.Now().UTC()
			_ = a.store.UpdateUser(ctx, u.ID, store.UserPatch{Totp_last_verified_at: &now})
		}
	}

	issued, err := a.issueSession(ctx, u, ip, ua)
	if err != nil {
		return LoginResult{}, err
	}
	result.Token = issued.Token
	result.Require_totp_setup = issued.Require_totp_setup || result.Require_totp_setup
	return result, nil
}

// issueSession creates a new session for an already-authenticated user.
// Used by Login and the Google OAuth callback.
func (a *AuthService) issueSession(ctx context.Context, u store.User, ip, ua string) (LoginResult, error) {
	if err := statusGateError(u.Status); err != nil {
		return LoginResult{}, err
	}
	if !u.Active {
		return LoginResult{}, err_invalid_credentials
	}
	session_duration := talent_session_duration
	switch u.Role {
	case store.Role_superadmin, store.Role_admin, store.Role_campaign_manager:
		session_duration = admin_session_duration
	}
	token, err := generateToken()
	if err != nil {
		return LoginResult{}, err
	}
	if err := a.store.CreateSession(ctx, store.Session{
		User_id:        u.ID,
		Token:          token,
		IP_address:     ip,
		User_agent:     ua,
		Last_active_at: time.Now().UTC(),
		Expires_at:     time.Now().UTC().Add(session_duration),
	}); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		Token:              token,
		User_id:            u.ID,
		Role:               u.Role,
		Status:             u.Status,
		Active:             u.Active,
		Require_totp_setup: !u.Totp_verified,
	}, nil
}

// checkStaffTOTP enforces TOTP for superadmin/admin/campaign_manager after enrollment.
// Before enrollment (totp_verified=false): allowed — first login after invite verification.
func (a *AuthService) checkStaffTOTP(u store.User, code string) error {
	if !u.Totp_enabled || !u.Totp_verified {
		return nil
	}
	if code == "" {
		return err_totp_required
	}
	if !totp.Validate(code, u.Totp_secret) {
		return err_invalid_totp
	}
	return nil
}

// ─── TOTP enrollment ──────────────────────────────────────────────────────────

func (a *AuthService) EnrollTOTP(ctx context.Context, user_id string) (secret, qr_uri string, err error) {
	u, err := a.store.GetUserByID(ctx, user_id)
	if err != nil {
		return "", "", err
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      a.issuer,
		AccountName: u.Email,
	})
	if err != nil {
		return "", "", err
	}
	_ = a.store.UpdateUser(ctx, user_id, store.UserPatch{Totp_secret: strPtr(key.Secret())})
	return key.Secret(), key.URL(), nil
}

func (a *AuthService) VerifyAndEnableTOTP(ctx context.Context, user_id, code string) error {
	u, err := a.store.GetUserByID(ctx, user_id)
	if err != nil {
		return err
	}
	if !totp.Validate(code, u.Totp_secret) {
		return err_invalid_totp
	}
	t := true
	now := time.Now().UTC()
	return a.store.UpdateUser(ctx, user_id, store.UserPatch{
		Totp_enabled:          &t,
		Totp_verified:         &t,
		Totp_last_verified_at: &now,
	})
}

// ─── Session validation ───────────────────────────────────────────────────────

func (a *AuthService) ValidateSession(ctx context.Context, token string) (store.User, error) {
	sess, err := a.store.GetSession(ctx, token)
	if err != nil || sess.Invalidated {
		return store.User{}, err_session_expired
	}
	if time.Now().After(sess.Expires_at) {
		return store.User{}, err_session_expired
	}
	if time.Since(sess.Last_active_at) > inactivity_timeout {
		_ = a.store.InvalidateSession(ctx, token)
		return store.User{}, err_session_inactive
	}
	_ = a.store.TouchSession(ctx, token, time.Now().UTC())
	u, err := a.store.GetUserByID(ctx, sess.User_id)
	if err != nil {
		return store.User{}, err_session_expired
	}
	if sessionBlocked(u.Status) {
		_ = a.store.InvalidateSession(ctx, token)
		if err := statusGateError(u.Status); err != nil {
			return store.User{}, err
		}
		return store.User{}, err_account_suspended
	}
	return u, nil
}

func (a *AuthService) Logout(ctx context.Context, token string) error {
	return a.store.InvalidateSession(ctx, token)
}

// ─── Generic campaign viewer (non-brand-contact) ──────────────────────────────

func (a *AuthService) ValidateViewerAccess(ctx context.Context, token, password string) (store.CampaignViewer, error) {
	viewer, err := a.store.GetViewerByToken(ctx, token)
	if err != nil || !viewer.Active {
		return store.CampaignViewer{}, err_viewer_invalid
	}
	ok, err := a.store.ValidateViewerPassword(ctx, viewer.ID, password)
	if err != nil || !ok {
		return store.CampaignViewer{}, err_viewer_invalid
	}
	return viewer, nil
}

// ─── Brand contact access ─────────────────────────────────────────────────────

// ValidateBrandContactAccess validates a brand contact's token + password.
// Returns the contact record. Caller is responsible for setting the cookie.
func (a *AuthService) ValidateBrandContactAccess(ctx context.Context, token, password string) (store.BrandContact, error) {
	contact, err := a.store.GetBrandContactByViewerToken(ctx, token)
	if err != nil || !contact.Token_active {
		return store.BrandContact{}, err_viewer_invalid
	}
	if err := bcrypt.CompareHashAndPassword([]byte(contact.Access_password_hash), []byte(password)); err != nil {
		return store.BrandContact{}, err_viewer_invalid
	}
	return contact, nil
}

// GetActiveBrandContact checks that a brand contact token exists and is active.
// Used by middleware to re-validate a cookie without re-checking the password.
func (a *AuthService) GetActiveBrandContact(ctx context.Context, token string) (store.BrandContact, error) {
	contact, err := a.store.GetBrandContactByViewerToken(ctx, token)
	if err != nil || !contact.Token_active {
		return store.BrandContact{}, err_viewer_invalid
	}
	return contact, nil
}

// ─── Google OAuth ─────────────────────────────────────────────────────────────

// GoogleAuthURL returns the OAuth consent URL and a random state token for CSRF protection.
// Returns empty strings if Google OAuth is not configured.
func (a *AuthService) GoogleAuthURL() (auth_url, state string, err error) {
	if a.google_conf == nil {
		return "", "", errors.New("google_oauth_not_configured")
	}
	state, err = generateToken()
	if err != nil {
		return "", "", err
	}
	return a.google_conf.AuthCodeURL(state, oauth2.AccessTypeOnline), state, nil
}

// ExchangeGoogleCode exchanges an authorization code for a user record,
// creating a new talent account if the Google ID is not yet registered.
func (a *AuthService) ExchangeGoogleCode(ctx context.Context, code string) (store.User, error) {
	if a.google_conf == nil {
		return store.User{}, errors.New("google_oauth_not_configured")
	}
	tok, err := a.google_conf.Exchange(ctx, code)
	if err != nil {
		return store.User{}, err_invalid_credentials
	}
	info, err := fetchGoogleUserInfo(ctx, a.google_conf, tok)
	if err != nil {
		return store.User{}, err
	}
	return a.RegisterTalentGoogle(ctx, info.id, info.email, info.name, info.picture)
}

type google_user_info struct {
	id      string
	email   string
	name    string
	picture string
}

func fetchGoogleUserInfo(ctx context.Context, conf *oauth2.Config, tok *oauth2.Token) (google_user_info, error) {
	client := conf.Client(ctx, tok)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return google_user_info{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	var raw struct {
		Sub     string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := jsonDecode(resp.Body, &raw); err != nil {
		return google_user_info{}, err
	}
	return google_user_info{id: raw.Sub, email: raw.Email, name: raw.Name, picture: raw.Picture}, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GeneratePlainPassword generates a human-readable plain password for brand contacts.
// Returns both the plain text (sent to admin once) and the bcrypt hash (stored in DB).
func GeneratePlainPassword() (plain, hash string, err error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	plain = base64.URLEncoding.EncodeToString(b)[:16] // 16-char url-safe string
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt_cost)
	if err != nil {
		return "", "", err
	}
	return plain, string(h), nil
}

func strPtr(s string) *string { return &s }

func jsonDecode(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

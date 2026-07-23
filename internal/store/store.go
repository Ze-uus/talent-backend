package store

import (
	"context"
	"time"

	"github.com/Ze-uus/talent-backend/internal/algo"
)

type Store interface {
	// --- Users ---
	CreateUser(ctx context.Context, u User) error
	GetUserByID(ctx context.Context, id string) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByGoogleID(ctx context.Context, google_id string) (User, error)
	GetUserByInviteToken(ctx context.Context, token string) (User, error)
	UpdateUser(ctx context.Context, id string, patch UserPatch) error
	ListUsers(ctx context.Context, filter UserFilter) ([]User, error)
	Ping(ctx context.Context) error

	// --- Sessions ---
	CreateSession(ctx context.Context, s Session) error
	GetSession(ctx context.Context, token string) (Session, error)
	TouchSession(ctx context.Context, token string, now time.Time) error
	InvalidateSession(ctx context.Context, token string) error
	InvalidateAllUserSessions(ctx context.Context, user_id string) error
	ListSessionsByUser(ctx context.Context, user_id string) ([]Session, error)

	// --- Brands ---
	CreateBrand(ctx context.Context, b Brand) error
	GetBrandByID(ctx context.Context, id string) (Brand, error)
	GetBrandByShortcode(ctx context.Context, shortcode string) (Brand, error)
	ListBrands(ctx context.Context, filter BrandFilter) ([]Brand, error)
	UpdateBrand(ctx context.Context, id string, patch BrandPatch) error
	ShortcodeExists(ctx context.Context, shortcode string) (bool, error)

	// --- Brand Contacts ---
	// Contacts are mutable — add, update, remove without affecting brand record.
	// On creation: system generates viewer_token and access_password_hash.
	// On removal: token is deactivated, not deleted (preserves audit trail).
	CreateBrandContact(ctx context.Context, c BrandContact) error
	GetBrandContactByID(ctx context.Context, id string) (BrandContact, error)
	GetBrandContactByViewerToken(ctx context.Context, token string) (BrandContact, error)
	ListBrandContacts(ctx context.Context, brand_id string) ([]BrandContact, error)
	UpdateBrandContact(ctx context.Context, id string, patch BrandContactPatch) error
	DeactivateBrandContact(ctx context.Context, id string) error
	RegenerateBrandContactPassword(ctx context.Context, id string, new_hash string) error
	ValidateBrandContactAccess(ctx context.Context, token, password string) (BrandContact, error)

	// --- Talents ---
	CreateTalent(ctx context.Context, t Talent) error
	GetTalentByID(ctx context.Context, id string) (Talent, error)
	GetTalentByUserID(ctx context.Context, user_id string) (Talent, error)
	ListTalents(ctx context.Context, filter TalentFilter) ([]Talent, error)
	UpdateTalent(ctx context.Context, id string, patch TalentPatch) error
	ListAllTalents(ctx context.Context) ([]Talent, error)

	// --- Campaigns ---
	CreateCampaign(ctx context.Context, c Campaign) error
	GetCampaignByID(ctx context.Context, id string) (Campaign, error)
	GetCampaignByHumanID(ctx context.Context, human_id string) (Campaign, error)
	ListCampaigns(ctx context.Context, filter CampaignFilter) ([]Campaign, error)
	ListActiveCampaigns(ctx context.Context) ([]Campaign, error)
	UpdateCampaign(ctx context.Context, id string, patch CampaignPatch) error
	DecrementRemainingBudget(ctx context.Context, campaign_id string, amount float64) error
	// NextCampaignHumanID returns the next available human_id for a brand in the current month.
	NextCampaignHumanID(ctx context.Context, brand_shortcode string) (string, error)

	// --- Campaign manager assignments ---
	GetCampaignsByManagerID(ctx context.Context, manager_id string) ([]Campaign, error)
	ListManagersByCampaignID(ctx context.Context, campaign_id string) ([]User, error)
	AssignManagerToCampaign(ctx context.Context, manager_id, campaign_id, assigned_by string) error
	UnassignManagerFromCampaign(ctx context.Context, manager_id, campaign_id string) error

	// --- Email dispatches (idempotent reminder/digest sends) ---
	// TryRecordEmailDispatch inserts a dispatch row; returns true if this was the first send.
	TryRecordEmailDispatch(ctx context.Context, entity_type, entity_id, template_key, recipient string) (bool, error)
	EmailDispatchExists(ctx context.Context, entity_type, entity_id, template_key, recipient string) (bool, error)

	// --- Cycles ---
	CreateCycle(ctx context.Context, c Cycle) error
	GetCycleByID(ctx context.Context, id string) (Cycle, error)
	ListCyclesByCampaign(ctx context.Context, campaign_id string) ([]Cycle, error)
	GetActiveCycles(ctx context.Context) ([]Cycle, error)
	UpdateCycle(ctx context.Context, id string, patch CyclePatch) error
	CloseCycle(ctx context.Context, id string, spend float64) error

	// --- Budget Slots (cycle-scoped) ---
	CreateBudgetSlots(ctx context.Context, slots []BudgetSlot) error
	ListSlotsByCycle(ctx context.Context, cycle_id string) ([]BudgetSlot, error)
	AssignSlot(ctx context.Context, slot_id, talent_id string) error

	// --- Assignments (cycle-scoped) ---
	CreateAssignment(ctx context.Context, a TalentAssignment) error
	GetAssignment(ctx context.Context, talent_id, cycle_id string) (TalentAssignment, error)
	ListAssignedTalents(ctx context.Context, cycle_id string) ([]TalentAssignment, error)
	ListAssignmentsByTalent(ctx context.Context, talent_id string) ([]TalentAssignment, error)
	UpdateAssignment(ctx context.Context, talent_id, cycle_id string, patch AssignmentPatch) error

	// --- Tracking links ---
	CreateTrackingLink(ctx context.Context, l TrackingLink) error
	GetTrackingLinkByToken(ctx context.Context, token string) (TrackingLink, error)
	ListTrackingLinksByCycle(ctx context.Context, cycle_id string) ([]TrackingLink, error)

	// --- Conversion events ---
	LogConversionEvent(ctx context.Context, e ConversionEvent) error
	GetDailyConversionCount(ctx context.Context, talent_id, cycle_id string, date time.Time) (float64, error)
	GetTotalConversions(ctx context.Context, cycle_id string) (float64, error)
	GetTalentConversions(ctx context.Context, talent_id, cycle_id string) (float64, error)
	// FlagFallbackConversions sets fallback_flagged=true on all unflagged events for the cycle/day.
	FlagFallbackConversions(ctx context.Context, cycle_id string, day time.Time) error
	// LockFallbackConversions freezes all fallback-flagged records (Story 24 — Finalise Payouts).
	LockFallbackConversions(ctx context.Context, cycle_id string) error

	// --- Algo state ---
	GetCycleState(ctx context.Context, talent_id, cycle_id string) (CycleState, error)
	UpsertCycleState(ctx context.Context, s CycleState) error
	GetDailyOutputs(ctx context.Context, talent_id, cycle_id string) ([]float64, error)

	// --- Talent baselines ---
	GetTalentBaseline(ctx context.Context, talent_id string) (TalentBaseline, error)
	UpsertTalentBaseline(ctx context.Context, b TalentBaseline) error
	GetCategoryBaseline(ctx context.Context, category string) (algo.CategoryBaseline, error)
	GetTalentTodayOutput(ctx context.Context, talent_id string, date time.Time) (float64, error)
	GetTalentOutputWindow(ctx context.Context, talent_id string, days int) ([]float64, error)

	// --- Payout records ---
	CreatePayoutRecord(ctx context.Context, p PayoutRecord) error
	GetPayoutRecord(ctx context.Context, talent_id, cycle_id string) (PayoutRecord, error)
	ListPayoutsByCycle(ctx context.Context, cycle_id string) ([]PayoutRecord, error)
	ListPayoutsByTalent(ctx context.Context, talent_id string) ([]PayoutRecord, error)
	UpdatePayoutRecord(ctx context.Context, id string, patch PayoutPatch) error

	// --- Campaign viewers (generic) ---
	CreateViewer(ctx context.Context, v CampaignViewer) error
	GetViewerByToken(ctx context.Context, token string) (CampaignViewer, error)
	ListViewersByCampaign(ctx context.Context, campaign_id string) ([]CampaignViewer, error)
	UpdateViewer(ctx context.Context, id string, patch ViewerPatch) error
	AddViewerPassword(ctx context.Context, p ViewerPassword) error
	ListViewerPasswords(ctx context.Context, viewer_id string) ([]ViewerPassword, error)
	DeactivateViewerPassword(ctx context.Context, id string) error
	ValidateViewerPassword(ctx context.Context, viewer_id, password string) (bool, error)

	// --- Audit log ---
	WriteAuditLog(ctx context.Context, entry AuditLog) error
	ListAuditLog(ctx context.Context, entity_type, entity_id string) ([]AuditLog, error)
}

// ─── Patch types ──────────────────────────────────────────────────────────────

type UserPatch struct {
	Full_name             *string
	Avatar_url            *string
	Password_hash         *string
	Totp_secret           *string
	Totp_enabled          *bool
	Totp_verified         *bool
	Totp_last_verified_at *time.Time
	Invite_token          *string
	Invite_expires_at     *time.Time
	Active                *bool
}

type BrandPatch struct {
	Name        *string
	Industry    *string
	Description *string
	Website     *string
	Logo_url    *string
	Status      *string
}

type BrandContactPatch struct {
	First_name      *string
	Last_name       *string
	Role            *string
	Email           *string
	Whatsapp_number *string
	Token_active    *bool
}

type TalentPatch struct {
	Status            *string
	Category          *string
	Rate_per_day      *float64
	Max_tier          *int
	Skills            []string
	Bio               *string
	Portfolio_url     *string
	Report_compliance *float64
}

type CampaignPatch struct {
	Name             *string
	Status           *string
	Urgency_level    *string
	Cycle_length     *int
	End_date         *time.Time
	Creators_allowed *bool
}

type CyclePatch struct {
	Status          *string
	Cycle_budget    *float64
	Cycle_objective *string
	Z_factor        *float64
	End_date        *time.Time
}

type AssignmentPatch struct {
	Status        *string
	Breakout_flag *bool
	Match_score   *float64
}

type PayoutPatch struct {
	Status           *Payout_status
	Admin_override   *bool
	Override_reason  *string
	Approved_by      *string
	Approved_at      *time.Time
	Paid_at          *time.Time
	Failure_reason   *string
	Retry_count      *int
	Report_submitted *bool
}

type ViewerPatch struct {
	Name   *string
	Active *bool
}

// ─── Filter types ─────────────────────────────────────────────────────────────

type UserFilter struct {
	Role   string
	Active *bool
}

type BrandFilter struct {
	Status string
	Limit  int
	Offset int
}

type TalentFilter struct {
	Status   string
	Category string
	Min_pdc  float64
	Limit    int
	Offset   int
}

type CampaignFilter struct {
	Status   string
	Brand_id string
	Limit    int
	Offset   int
}

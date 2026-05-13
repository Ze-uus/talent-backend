package store

import "time"

// ─── Constants ────────────────────────────────────────────────────────────────

type user_role string

const (
	role_admin  user_role = "admin"
	role_talent user_role = "talent"
	role_viewer user_role = "viewer"
)

type auth_provider string

const (
	provider_local  auth_provider = "local"
	provider_google auth_provider = "google"
)

type talent_status string

const (
	status_pending   talent_status = "pending"
	status_active    talent_status = "active"
	status_suspended talent_status = "suspended"
	status_rejected  talent_status = "rejected"
)

type campaign_status string

const (
	campaign_draft    campaign_status = "draft"
	campaign_active   campaign_status = "active"
	campaign_paused   campaign_status = "paused"
	campaign_closed   campaign_status = "closed"
	campaign_archived campaign_status = "archived"
)

type cycle_status string

const (
	cycle_pending cycle_status = "pending"
	cycle_active  cycle_status = "active"
	cycle_closed  cycle_status = "closed"
)

type campaign_type string

const (
	type_direct_traffic  campaign_type = "direct_traffic"
	type_lead_validation campaign_type = "lead_validation"
)

// Urgency thresholds per Story 5b.
// Low: <100 conv/day, Normal: 100–300/day, High: >300/day
type urgency_level string

const (
	urgency_low    urgency_level = "low"
	urgency_normal urgency_level = "normal"
	urgency_high   urgency_level = "high"
)

type talent_category string

const (
	category_student   talent_category = "student"   // Tier C — median PDC: 2
	category_micro     talent_category = "micro"     // Tier B — median PDC: 5
	category_community talent_category = "community" // Tier A — median PDC: 15
)

type pdc_mode string

const (
	pdc_cold_start pdc_mode = "cold_start"
	pdc_stec       pdc_mode = "stec"
	pdc_ltec       pdc_mode = "ltec"
)

type assignment_source string

const (
	source_algorithm assignment_source = "algorithm"
	source_pinned    assignment_source = "pinned"
	source_manual    assignment_source = "manual"
)

type payout_status string

const (
	payout_report_pending payout_status = "report_pending"
	payout_pending        payout_status = "pending"
	payout_approved       payout_status = "approved"
	payout_paid           payout_status = "paid"
	payout_forfeited      payout_status = "forfeited"
	payout_failed         payout_status = "failed"
)

// ─── User + Session ───────────────────────────────────────────────────────────

type User struct {
	ID                    string
	Email                 string
	Password_hash         string
	Role                  user_role
	Provider              auth_provider
	Google_id             string
	Full_name             string
	Avatar_url            string
	Totp_secret           string
	Totp_enabled          bool
	Totp_verified         bool
	Totp_last_verified_at time.Time
	Invite_token          string
	Invite_expires_at     time.Time
	Active                bool
	Created_at            time.Time
	Updated_at            time.Time
}

type Session struct {
	ID             string
	User_id        string
	Token          string
	IP_address     string
	User_agent     string
	Last_active_at time.Time
	Expires_at     time.Time
	Invalidated    bool
}

// ─── Brand + Brand Contacts ───────────────────────────────────────────────────

// Brand — client entity. Shortcode is auto-generated from brand name
// and used as the prefix for all Campaign IDs and Cycle IDs.
// e.g. "CreditDirect" → shortcode "CRD" → Campaign ID "CRD-26-01"
type Brand struct {
	ID          string
	Name        string
	Shortcode   string // 3-letter auto-generated, e.g. "CRD"
	Industry    string
	Description string
	Website     string
	Status      string // "active" | "suspended"
	Created_at  time.Time
	Updated_at  time.Time
}

// BrandContact — a person associated with a brand who is given access
// to campaign viewer dashboards. Up to 3 per brand (cap enforced at service layer).
//
// Access model:
//   - Contact is linked to the brand, NOT permanently fixed.
//   - On creation, a unique viewer_token and hashed access_password are generated.
//   - The contact uses token + password to access their campaign view.
//   - If the contact is removed, their token is deactivated (soft-delete).
//   - Admin can regenerate the password at any time.
//   - Future_user_account_id is reserved for when brand contacts get full portal access.
type BrandContact struct {
	ID                     string
	Brand_id               string
	First_name             string
	Last_name              string
	Role                   string // e.g. "CMO", "Marketing Manager" — free text
	Email                  string
	Whatsapp_number        string
	Viewer_token           string // unique URL token for campaign access
	Access_password_hash   string // hashed; system-generated on contact creation
	Token_active           bool   // set false when contact is removed
	Future_user_account_id string // null until brand contact portal is built
	Created_at             time.Time
	Updated_at             time.Time
}

// ─── Talent ───────────────────────────────────────────────────────────────────

type Talent struct {
	ID                string
	User_id           string
	Category          talent_category
	Status            talent_status
	Skills            []string
	Rate_per_day      float64
	Max_tier          int
	Bio               string
	Portfolio_url     string
	Report_compliance float64
	Created_at        time.Time
	Updated_at        time.Time
}

// ─── Campaign ─────────────────────────────────────────────────────────────────

// Campaign — top-level engagement record.
// Human ID format: "CRD-26-01" (shortcode-YY-sequential).
type Campaign struct {
	ID               string
	Human_id         string          // e.g. "CRD-26-01" — display identifier
	Brand_id         string          // FK → brands.id
	Name             string
	Status           campaign_status
	Campaign_type    campaign_type
	Total_budget     float64
	Remaining_budget float64 // system-managed, decremented as cycles close
	Market_cap       string  // "M" = unlimited, or numeric string e.g. "300"
	Audience         string
	Target_cpa       float64
	Max_cpa          float64
	Urgency_level    urgency_level
	Cycle_length     int // 5, 7, or 10 days
	Creators_allowed bool
	Start_date       time.Time
	End_date         time.Time
	Created_at       time.Time
	Updated_at       time.Time
}

// ─── Cycle ────────────────────────────────────────────────────────────────────

// Cycle — atomic execution unit.
// Human ID format: "CRD-26-01-C2" (campaign human_id + "-C" + cycle_number).
type Cycle struct {
	ID               string
	Human_id         string        // e.g. "CRD-26-01-C2"
	Campaign_id      string
	Cycle_number     int
	Status           cycle_status
	Cycle_budget     float64
	Remaining_budget float64
	Cycle_objective  string
	Campaign_type    campaign_type
	KPB_config       []KPBDefinition // lead_validation only
	Z_factor         float64         // derived from campaign urgency_level at creation
	Start_date       time.Time
	End_date         time.Time
	Created_at       time.Time
	Updated_at       time.Time
}

type KPBDefinition struct {
	Label    string  `json:"label"`
	Cost     float64 `json:"cost"`
	Quantity int     `json:"quantity"`
}

// ─── Budget Slot ──────────────────────────────────────────────────────────────

type BudgetSlot struct {
	ID         string
	Cycle_id   string
	Tier_value int
	Slot_index int
	Allocated  bool
	Talent_id  string
}

// ─── Tracking Link ────────────────────────────────────────────────────────────

type TrackingLink struct {
	ID          string
	Campaign_id string
	Cycle_id    string
	Talent_id   string
	Token       string
	Active      bool
	Created_at  time.Time
}

// ─── Conversion Event ─────────────────────────────────────────────────────────

// ConversionEvent — fired via tracking link.
// Pipeline_type and Valid_lead are critical for Story 17 payout branching.
// Fallback_flagged set by cycle-end process when downstream data unavailable (Story 24).
type ConversionEvent struct {
	ID               string
	Link_token       string
	Talent_id        string
	Campaign_id      string
	Cycle_id         string
	Pipeline_type    campaign_type // direct_traffic | lead_validation
	Event_type       string        // "click" | "signup" | "purchase" | kpb_label
	KPB_type         string        // populated for lead_validation KPB events
	Valid_lead       bool          // set by decision evaluation (Story 5e(v)); Lead Pipeline only
	Fallback_flagged bool          // true when fallback conversion rule applied (Story 24)
	Idempotency_key  string
	Occurred_at      time.Time
}

// ─── Talent Assignment ────────────────────────────────────────────────────────

// TalentAssignment — per talent per cycle.
// Stores full allocation audit data per Story 17 P.10 and Story 18 A.11.
type TalentAssignment struct {
	Talent_id         string
	Campaign_id       string
	Cycle_id          string
	Slot_id           string
	Role_label        string
	Status            string            // "active" | "completed" | "removed_forfeit" | "removed_payout"
	Assignment_source assignment_source // algorithm | pinned | manual
	PDC_mode          pdc_mode          // cold_start | stec | ltec
	PDC_value         float64           // PDC_t at time of allocation
	Match_score       float64           // MS_t composite (0.0–1.0)
	Match_dm          float64           // demographic match component
	Match_gp          float64           // geographic proximity component
	Match_oh          float64           // objective history component
	Pinned_tier       int               // 0 if not pinned
	Effective_tier    int               // actual assigned tier value
	Breakout_flag     bool              // set at cycle end by breakout detection
	Assigned_at       time.Time
}

// ─── Algo State ───────────────────────────────────────────────────────────────

type CycleState struct {
	Talent_id     string
	Cycle_id      string
	Cycle_number  int
	PDC_allocated float64
	PDC_next      float64
	Pattern       string
	Delta_st      float64
	Updated_at    time.Time
}

type TalentBaseline struct {
	Talent_id  string
	Alpha_lt   float64
	Beta_lt    float64
	Lambda_lt  float64
	Sigma_hist float64
	Delta_lt   float64
	Updated_at time.Time
}

// ─── Payout Records ───────────────────────────────────────────────────────────

// PayoutRecord — per talent per cycle audit record (Story 17 P.10).
// Created at cycle close by the payout service.
type PayoutRecord struct {
	ID            string
	Talent_id     string
	Cycle_id      string
	Campaign_id   string
	Pipeline_type campaign_type
	Status        payout_status

	// Pre-commission fields
	Allocated_budget float64
	Gross_base       float64
	KPB_total        float64
	Gross_total      float64

	// Cap evaluation
	Cost_per_unit  float64 // cost per conversion (Traffic) or per lead (Lead)
	Cap_applied    float64
	Cap_exceeded   bool
	Excess_forfeited float64

	// Commission + net
	Commission_rate   float64
	Commission_amount float64
	E_net             float64

	// Pool scaling
	Scale_factor float64
	Final_payout float64

	// Audit
	KPB_pool_source  string // "allocated_budget" | "separate_pool" | "n/a"
	Fallback_flagged bool
	Report_submitted bool
	Admin_override   bool
	Override_reason  string
	Approved_by      string
	Approved_at      time.Time
	Paid_at          time.Time
	Failure_reason   string
	Retry_count      int
	Created_at       time.Time
	Updated_at       time.Time
}

// ─── Campaign Viewers ─────────────────────────────────────────────────────────

// CampaignViewer — generic token+password access for a campaign dashboard.
// Used for ad-hoc viewer links created by admin (not tied to a brand contact).
type CampaignViewer struct {
	ID          string
	Campaign_id string
	Token       string
	Name        string
	Active      bool
	Created_at  time.Time
}

type ViewerPassword struct {
	ID            string
	Viewer_id     string
	Password_hash string
	Label         string
	Active        bool
	Created_at    time.Time
}

// ─── Reporting ────────────────────────────────────────────────────────────────

type TalentReport struct {
	ID                string
	Talent_id         string
	Cycle_id          string
	Submitted_at      time.Time
	Deadline_at       time.Time
	Audience_reached  string
	Offer_promoted    string
	Friction_score    float64
	Audience_reaction string
	Blocker_category  string
	Status            string // "pending" | "submitted" | "late" | "flagged"
	Created_at        time.Time
}

type CycleReport struct {
	ID                              string
	Cycle_id                        string
	Offer_decision_label            string
	Max_spend_decision_label        string
	Predicted_daily_conversions_next float64
	Generated_at                    time.Time
}

// ─── Audit Log ────────────────────────────────────────────────────────────────

type AuditLog struct {
	ID           string
	Actor_id     string
	Action_type  string
	Entity_type  string
	Entity_id    string
	Before_state []byte
	After_state  []byte
	Created_at   time.Time
}

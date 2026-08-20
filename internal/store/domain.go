package store

import "time"

// ─── Constants ────────────────────────────────────────────────────────────────

type User_role string

const (
	Role_superadmin       User_role = "superadmin"
	Role_admin            User_role = "admin"
	Role_campaign_manager User_role = "campaign_manager"
	Role_talent           User_role = "talent"
	Role_viewer           User_role = "viewer"
)

type Auth_provider string

const (
	Provider_local  Auth_provider = "local"
	Provider_google Auth_provider = "google"
)

type Talent_status string

const (
	Status_pending   Talent_status = "pending"
	Status_active    Talent_status = "active"
	Status_suspended Talent_status = "suspended"
	Status_rejected  Talent_status = "rejected"
)

// User_status is the shared account lifecycle for all roles.
type User_status string

const (
	User_status_active    User_status = "active"
	User_status_suspended User_status = "suspended"
	User_status_banned    User_status = "banned"
	User_status_deleted   User_status = "deleted"
	User_status_pending   User_status = "pending"
	User_status_rejected  User_status = "rejected"
	User_status_invited   User_status = "invited"
)

type Campaign_status string

const (
	Campaign_draft    Campaign_status = "draft"
	Campaign_active   Campaign_status = "active"
	Campaign_paused   Campaign_status = "paused"
	Campaign_closed   Campaign_status = "closed"
	Campaign_archived Campaign_status = "archived"
)

type Cycle_status string

const (
	Cycle_pending Cycle_status = "pending"
	Cycle_active  Cycle_status = "active"
	Cycle_paused  Cycle_status = "paused"
	Cycle_closed  Cycle_status = "closed"
)

type Campaign_type string

const (
	Type_direct_traffic  Campaign_type = "direct_traffic"
	Type_lead_validation Campaign_type = "lead_validation"
)

// Urgency thresholds per Story 5b.
// Low: <100 conv/day, Normal: 100–300/day, High: >300/day
type Urgency_level string

const (
	Urgency_low    Urgency_level = "low"
	Urgency_normal Urgency_level = "normal"
	Urgency_high   Urgency_level = "high"
)

type Talent_category string

const (
	Category_student   Talent_category = "student"   // Tier C — median PDC: 2
	Category_micro     Talent_category = "micro"     // Tier B — median PDC: 5
	Category_community Talent_category = "community" // Tier A — median PDC: 15
)

type Pdc_mode string

const (
	Pdc_cold_start Pdc_mode = "cold_start"
	Pdc_stec       Pdc_mode = "stec"
	Pdc_ltec       Pdc_mode = "ltec"
)

type Assignment_source string

const (
	Source_algorithm Assignment_source = "algorithm"
	Source_pinned    Assignment_source = "pinned"
	Source_manual    Assignment_source = "manual"
)

type Payout_status string

const (
	Payout_report_pending Payout_status = "report_pending"
	Payout_pending        Payout_status = "pending"
	Payout_approved       Payout_status = "approved"
	Payout_paid           Payout_status = "paid"
	Payout_forfeited      Payout_status = "forfeited"
	Payout_failed         Payout_status = "failed"
)

// ─── User + Session ───────────────────────────────────────────────────────────

type User struct {
	ID                    string
	Email                 string
	Password_hash         string `json:"-"` // never serialize
	Role                  User_role
	Provider              Auth_provider
	Google_id             string `json:"-"` // never serialize
	Full_name             string
	Phone_number          string
	Avatar_url            string
	Totp_secret           string `json:"-"` // never serialize — only returned explicitly from enroll
	Totp_enabled          bool
	Totp_verified         bool
	Totp_last_verified_at time.Time
	Invite_token          string    `json:"-"` // never serialize
	Invite_expires_at     time.Time `json:"-"` // never serialize
	Active                bool
	Status                User_status
	Deleted_at            *time.Time `json:",omitempty"`
	Created_at            time.Time
	Updated_at            time.Time
}

type Session struct {
	ID             string
	User_id        string
	Token          string `json:"-"` // never serialize — session secret
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
	Logo_url    string
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
	Access_password_hash   string `json:"-"` // never serialize
	Token_active           bool   // set false when contact is removed
	Future_user_account_id string // null until brand contact portal is built
	Created_at             time.Time
	Updated_at             time.Time
}

// ─── Talent ───────────────────────────────────────────────────────────────────

type Talent struct {
	ID                string
	User_id           string
	Category          Talent_category
	Status            Talent_status
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
	Human_id         string // e.g. "CRD-26-01" — display identifier
	Brand_id         string // FK → brands.id
	Name             string
	Status           Campaign_status
	Campaign_type    Campaign_type
	Total_budget     float64
	Remaining_budget float64 // system-managed, decremented as cycles close
	Market_cap       string  // "M" = unlimited, or numeric string e.g. "300"
	Audience         string
	Target_cpa       float64
	Max_cpa          float64
	Urgency_level    Urgency_level
	Cycle_length     int // 5, 7, or 10 days
	Creators_allowed bool
	Content          []ContentItem
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
	Human_id         string // e.g. "CRD-26-01-C2"
	Campaign_id      string
	Cycle_number     int
	Status           Cycle_status
	Cycle_budget     float64
	Remaining_budget float64
	Cycle_objective  string
	Campaign_type    Campaign_type
	KPB_config       []KPBDefinition // lead_validation only
	Content_override *[]ContentItem  // nil inherits campaign content
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

type ContentItem struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Images      []string      `json:"images"`
	Links       []ContentLink `json:"links"`
}

type ContentLink struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	URL   string `json:"url"`
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
	Pipeline_type    Campaign_type // direct_traffic | lead_validation
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
	Assignment_source Assignment_source // algorithm | pinned | manual
	PDC_mode          Pdc_mode          // cold_start | stec | ltec
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
	Pipeline_type Campaign_type
	Status        Payout_status

	// Pre-commission fields
	Allocated_budget float64
	Gross_base       float64
	KPB_total        float64
	Gross_total      float64

	// Cap evaluation
	Cost_per_unit    float64 // cost per conversion (Traffic) or per lead (Lead)
	Cap_applied      float64
	Cap_exceeded     bool
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
	Password_hash string `json:"-"` // never serialize
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
	ID                               string
	Cycle_id                         string
	Offer_decision_label             string
	Max_spend_decision_label         string
	Predicted_daily_conversions_next float64
	Generated_at                     time.Time
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
	Request_id   string
	Seq          int64
	Prev_hash    string
	Entry_hash   string
	Signature    string
	Archive_uri  string
	IP_address   string
	User_agent   string
	Created_at   time.Time
}

// AuditFilter scopes list queries for admin audit APIs.
type AuditFilter struct {
	Entity_type string
	Entity_id   string
	Actor_id    string
	Action_type string
	From        *time.Time
	To          *time.Time
	After_seq   int64 // cursor: return rows with seq < After_seq (newer-first)
	Limit       int
}

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/store"
)

// Store implements store.Store backed by a pgxpool connection pool.
type Store struct {
	pool *pgxpool.Pool
	q    *Queries
}

// NewStore opens a connection pool and returns a store.Store.
func NewStore(database_url string) (store.Store, error) {
	pool, err := pgxpool.New(context.Background(), database_url)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	return &Store{pool: pool, q: New(pool)}, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func ptext(s pgtype.Text) string {
	if s.Valid {
		return s.String
	}
	return ""
}

func pts(t pgtype.Timestamptz) time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Time{}
}

func ptsToPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid || t.Time.IsZero() {
		return nil
	}
	tt := t.Time
	return &tt
}

func ptsPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil || t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// ─── Type mappers ─────────────────────────────────────────────────────────────

func toStoreUser(u User) store.User {
	return store.User{
		ID:                    u.ID,
		Email:                 u.Email,
		Password_hash:         ptext(u.PasswordHash),
		Role:                  store.User_role(u.Role),
		Provider:              store.Auth_provider(u.Provider),
		Google_id:             ptext(u.GoogleID),
		Full_name:             u.FullName,
		Avatar_url:            u.AvatarUrl,
		Totp_secret:           u.TotpSecret,
		Totp_enabled:          u.TotpEnabled,
		Totp_verified:         u.TotpVerified,
		Totp_last_verified_at: pts(u.TotpLastVerifiedAt),
		Invite_token:          ptext(u.InviteToken),
		Invite_expires_at:     pts(u.InviteExpiresAt),
		Active:                u.Active,
		Status:                store.User_status(u.Status),
		Deleted_at:            ptsToPtr(u.DeletedAt),
		Created_at:            pts(u.CreatedAt),
		Updated_at:            pts(u.UpdatedAt),
	}
}

func toStoreSession(s Session) store.Session {
	return store.Session{
		ID:             s.ID,
		User_id:        s.UserID,
		Token:          s.Token,
		IP_address:     s.IpAddress,
		User_agent:     s.UserAgent,
		Last_active_at: pts(s.LastActiveAt),
		Expires_at:     pts(s.ExpiresAt),
		Invalidated:    s.Invalidated,
	}
}

func toStoreBrand(b Brand) store.Brand {
	return store.Brand{
		ID:          b.ID,
		Name:        b.Name,
		Shortcode:   b.Shortcode,
		Industry:    b.Industry,
		Description: b.Description,
		Website:     b.Website,
		Logo_url:    b.LogoUrl,
		Status:      b.Status,
		Created_at:  pts(b.CreatedAt),
		Updated_at:  pts(b.UpdatedAt),
	}
}

func toStoreBrandContact(c BrandContact) store.BrandContact {
	return store.BrandContact{
		ID:                     c.ID,
		Brand_id:               c.BrandID,
		First_name:             c.FirstName,
		Last_name:              c.LastName,
		Role:                   c.Role,
		Email:                  c.Email,
		Whatsapp_number:        c.WhatsappNumber,
		Viewer_token:           c.ViewerToken,
		Access_password_hash:   c.AccessPasswordHash,
		Token_active:           c.TokenActive,
		Future_user_account_id: ptext(c.FutureUserAccountID),
		Created_at:             pts(c.CreatedAt),
		Updated_at:             pts(c.UpdatedAt),
	}
}

func toStoreTalent(t Talent) store.Talent {
	return store.Talent{
		ID:                t.ID,
		User_id:           t.UserID,
		Category:          store.Talent_category(t.Category),
		Status:            store.Talent_status(t.Status),
		Skills:            t.Skills,
		Rate_per_day:      t.RatePerDay,
		Max_tier:          int(t.MaxTier),
		Bio:               t.Bio,
		Portfolio_url:     t.PortfolioUrl,
		Report_compliance: t.ReportCompliance,
		Created_at:        pts(t.CreatedAt),
		Updated_at:        pts(t.UpdatedAt),
	}
}

func toStoreCampaign(c Campaign) store.Campaign {
	return store.Campaign{
		ID:               c.ID,
		Human_id:         c.HumanID,
		Brand_id:         c.BrandID,
		Name:             c.Name,
		Status:           store.Campaign_status(c.Status),
		Campaign_type:    store.Campaign_type(c.CampaignType),
		Total_budget:     c.TotalBudget,
		Remaining_budget: c.RemainingBudget,
		Market_cap:       c.MarketCap,
		Audience:         c.Audience,
		Target_cpa:       c.TargetCpa,
		Max_cpa:          c.MaxCpa,
		Urgency_level:    store.Urgency_level(c.UrgencyLevel),
		Cycle_length:     int(c.CycleLength),
		Creators_allowed: c.CreatorsAllowed,
		Start_date:       pts(c.StartDate),
		End_date:         pts(c.EndDate),
		Created_at:       pts(c.CreatedAt),
		Updated_at:       pts(c.UpdatedAt),
	}
}

func toStoreCycle(c Cycle) store.Cycle {
	var kpb []store.KPBDefinition
	if len(c.KpbConfig) > 0 {
		_ = json.Unmarshal(c.KpbConfig, &kpb)
	}
	return store.Cycle{
		ID:               c.ID,
		Human_id:         c.HumanID,
		Campaign_id:      c.CampaignID,
		Cycle_number:     int(c.CycleNumber),
		Status:           store.Cycle_status(c.Status),
		Cycle_budget:     c.CycleBudget,
		Remaining_budget: c.RemainingBudget,
		Cycle_objective:  c.CycleObjective,
		Campaign_type:    store.Campaign_type(c.CampaignType),
		KPB_config:       kpb,
		Z_factor:         c.ZFactor,
		Start_date:       pts(c.StartDate),
		End_date:         pts(c.EndDate),
		Created_at:       pts(c.CreatedAt),
		Updated_at:       pts(c.UpdatedAt),
	}
}

func toStoreBudgetSlot(b BudgetSlot) store.BudgetSlot {
	return store.BudgetSlot{
		ID:         b.ID,
		Cycle_id:   b.CycleID,
		Tier_value: int(b.TierValue),
		Slot_index: int(b.SlotIndex),
		Allocated:  b.Allocated,
		Talent_id:  ptext(b.TalentID),
	}
}

func toStoreTrackingLink(l TrackingLink) store.TrackingLink {
	return store.TrackingLink{
		ID:          l.ID,
		Campaign_id: l.CampaignID,
		Cycle_id:    l.CycleID,
		Talent_id:   l.TalentID,
		Token:       l.Token,
		Active:      l.Active,
		Created_at:  pts(l.CreatedAt),
	}
}

func toStoreTalentAssignment(a TalentAssignment) store.TalentAssignment {
	return store.TalentAssignment{
		Talent_id:         a.TalentID,
		Campaign_id:       a.CampaignID,
		Cycle_id:          a.CycleID,
		Slot_id:           a.SlotID,
		Role_label:        a.RoleLabel,
		Status:            a.Status,
		Assignment_source: store.Assignment_source(a.AssignmentSource),
		PDC_mode:          store.Pdc_mode(a.PdcMode),
		PDC_value:         a.PdcValue,
		Match_score:       a.MatchScore,
		Match_dm:          a.MatchDm,
		Match_gp:          a.MatchGp,
		Match_oh:          a.MatchOh,
		Pinned_tier:       int(a.PinnedTier),
		Effective_tier:    int(a.EffectiveTier),
		Breakout_flag:     a.BreakoutFlag,
		Assigned_at:       pts(a.AssignedAt),
	}
}

func toStoreCycleState(s TalentCycleState) store.CycleState {
	return store.CycleState{
		Talent_id:     s.TalentID,
		Cycle_id:      s.CycleID,
		Cycle_number:  int(s.CycleNumber),
		PDC_allocated: s.PdcAllocated,
		PDC_next:      s.PdcNext,
		Pattern:       s.Pattern,
		Delta_st:      s.DeltaSt,
		Updated_at:    pts(s.UpdatedAt),
	}
}

func toStoreTalentBaseline(b TalentBaseline) store.TalentBaseline {
	return store.TalentBaseline{
		Talent_id:  b.TalentID,
		Alpha_lt:   b.AlphaLt,
		Beta_lt:    b.BetaLt,
		Lambda_lt:  b.LambdaLt,
		Sigma_hist: b.SigmaHist,
		Delta_lt:   b.DeltaLt,
		Updated_at: pts(b.UpdatedAt),
	}
}

func toStorePayoutRecord(p PayoutRecord) store.PayoutRecord {
	return store.PayoutRecord{
		ID:               p.ID,
		Talent_id:        p.TalentID,
		Cycle_id:         p.CycleID,
		Campaign_id:      p.CampaignID,
		Pipeline_type:    store.Campaign_type(p.PipelineType),
		Status:           store.Payout_status(p.Status),
		Allocated_budget: p.AllocatedBudget,
		Gross_base:       p.GrossBase,
		KPB_total:        p.KpbTotal,
		Gross_total:      p.GrossTotal,
		Cost_per_unit:    p.CostPerUnit,
		Cap_applied:      p.CapApplied,
		Cap_exceeded:     p.CapExceeded,
		Excess_forfeited: p.ExcessForfeited,
		Commission_rate:  p.CommissionRate,
		Commission_amount: p.CommissionAmount,
		E_net:            p.ENet,
		Scale_factor:     p.ScaleFactor,
		Final_payout:     p.FinalPayout,
		KPB_pool_source:  p.KpbPoolSource,
		Fallback_flagged: p.FallbackFlagged,
		Report_submitted: p.ReportSubmitted,
		Admin_override:   p.AdminOverride,
		Override_reason:  p.OverrideReason,
		Approved_by:      ptext(p.ApprovedBy),
		Approved_at:      pts(p.ApprovedAt),
		Paid_at:          pts(p.PaidAt),
		Failure_reason:   p.FailureReason,
		Retry_count:      int(p.RetryCount),
		Created_at:       pts(p.CreatedAt),
		Updated_at:       pts(p.UpdatedAt),
	}
}

func toStoreCampaignViewer(v CampaignViewer) store.CampaignViewer {
	return store.CampaignViewer{
		ID:          v.ID,
		Campaign_id: v.CampaignID,
		Token:       v.Token,
		Name:        v.Name,
		Active:      v.Active,
		Created_at:  pts(v.CreatedAt),
	}
}

func toStoreViewerPassword(p ViewerPassword) store.ViewerPassword {
	return store.ViewerPassword{
		ID:            p.ID,
		Viewer_id:     p.ViewerID,
		Password_hash: p.PasswordHash,
		Label:         p.Label,
		Active:        p.Active,
		Created_at:    pts(p.CreatedAt),
	}
}

func toStoreAuditLog(a AuditLog) store.AuditLog {
	seq := int64(0)
	if a.Seq.Valid {
		seq = a.Seq.Int64
	}
	return store.AuditLog{
		ID:           a.ID,
		Actor_id:     ptext(a.ActorID),
		Action_type:  a.ActionType,
		Entity_type:  a.EntityType,
		Entity_id:    a.EntityID,
		Before_state: a.BeforeState,
		After_state:  a.AfterState,
		Request_id:   a.RequestID,
		Seq:          seq,
		Prev_hash:    a.PrevHash,
		Entry_hash:   a.EntryHash,
		Signature:    a.Signature,
		Archive_uri:  a.ArchiveUri,
		IP_address:   a.IpAddress,
		User_agent:   a.UserAgent,
		Created_at:   pts(a.CreatedAt),
	}
}

// ─── Ping ─────────────────────────────────────────────────────────────────────

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// ─── Users ────────────────────────────────────────────────────────────────────

func (s *Store) CreateUser(ctx context.Context, u store.User) error {
	status := string(u.Status)
	if status == "" {
		status = string(store.User_status_active)
	}
	return s.q.CreateUser(ctx, CreateUserParams{
		Email:              u.Email,
		PasswordHash:       pgtype.Text{String: u.Password_hash, Valid: u.Password_hash != ""},
		Role:               string(u.Role),
		Provider:           string(u.Provider),
		GoogleID:           pgtype.Text{String: u.Google_id, Valid: u.Google_id != ""},
		FullName:           u.Full_name,
		AvatarUrl:          u.Avatar_url,
		TotpSecret:         u.Totp_secret,
		TotpEnabled:        u.Totp_enabled,
		TotpVerified:       u.Totp_verified,
		TotpLastVerifiedAt: ptsPtr(&u.Totp_last_verified_at),
		InviteToken:        pgtype.Text{String: u.Invite_token, Valid: u.Invite_token != ""},
		InviteExpiresAt:    ptsPtr(&u.Invite_expires_at),
		Active:             u.Active,
		Status:             status,
		DeletedAt:          ptsPtr(u.Deleted_at),
	})
}

func (s *Store) GetUserByID(ctx context.Context, id string) (store.User, error) {
	u, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return store.User{}, err
	}
	return toStoreUser(u), nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (store.User, error) {
	u, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return store.User{}, err
	}
	return toStoreUser(u), nil
}

func (s *Store) GetUserByGoogleID(ctx context.Context, google_id string) (store.User, error) {
	u, err := s.q.GetUserByGoogleID(ctx, pgtype.Text{String: google_id, Valid: google_id != ""})
	if err != nil {
		return store.User{}, err
	}
	return toStoreUser(u), nil
}

func (s *Store) GetUserByInviteToken(ctx context.Context, token string) (store.User, error) {
	u, err := s.q.GetUserByInviteToken(ctx, pgtype.Text{String: token, Valid: token != ""})
	if err != nil {
		return store.User{}, err
	}
	return toStoreUser(u), nil
}

func (s *Store) UpdateUser(ctx context.Context, id string, p store.UserPatch) error {
	args := pgx.NamedArgs{"id": id}
	sets := []string{"updated_at = NOW()"}
	if p.Full_name != nil {
		sets = append(sets, "full_name = @full_name")
		args["full_name"] = *p.Full_name
	}
	if p.Avatar_url != nil {
		sets = append(sets, "avatar_url = @avatar_url")
		args["avatar_url"] = *p.Avatar_url
	}
	if p.Password_hash != nil {
		sets = append(sets, "password_hash = @password_hash")
		args["password_hash"] = *p.Password_hash
	}
	if p.Totp_secret != nil {
		sets = append(sets, "totp_secret = @totp_secret")
		args["totp_secret"] = *p.Totp_secret
	}
	if p.Totp_enabled != nil {
		sets = append(sets, "totp_enabled = @totp_enabled")
		args["totp_enabled"] = *p.Totp_enabled
	}
	if p.Totp_verified != nil {
		sets = append(sets, "totp_verified = @totp_verified")
		args["totp_verified"] = *p.Totp_verified
	}
	if p.Totp_last_verified_at != nil {
		sets = append(sets, "totp_last_verified_at = @totp_last_verified_at")
		args["totp_last_verified_at"] = *p.Totp_last_verified_at
	}
	if p.Invite_token != nil {
		sets = append(sets, "invite_token = @invite_token")
		args["invite_token"] = *p.Invite_token
	}
	if p.Invite_expires_at != nil {
		sets = append(sets, "invite_expires_at = @invite_expires_at")
		args["invite_expires_at"] = *p.Invite_expires_at
	}
	if p.Active != nil {
		sets = append(sets, "active = @active")
		args["active"] = *p.Active
	}
	if p.Status != nil {
		sets = append(sets, "status = @status")
		args["status"] = *p.Status
	}
	if p.Deleted_at != nil {
		sets = append(sets, "deleted_at = @deleted_at")
		args["deleted_at"] = *p.Deleted_at
	}
	_, err := s.pool.Exec(ctx,
		"UPDATE users SET "+strings.Join(sets, ", ")+" WHERE id = @id", args)
	return err
}

func (s *Store) ListUsers(ctx context.Context, f store.UserFilter) ([]store.User, error) {
	args := pgx.NamedArgs{"role": f.Role, "status": f.Status}
	q := `SELECT id, email, password_hash, role, provider, google_id, full_name, avatar_url,
		totp_secret, totp_enabled, totp_verified, totp_last_verified_at, invite_token,
		invite_expires_at, active, created_at, updated_at, status, deleted_at
		FROM users WHERE (@role::text = '' OR role = @role)
		AND (@status::text = '' OR status = @status)`
	if f.Active != nil {
		q += " AND active = @active"
		args["active"] = *f.Active
	}
	q += " ORDER BY created_at DESC"
	pgRows, err := s.pool.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}
	defer pgRows.Close()
	var out []store.User
	for pgRows.Next() {
		var u User
		if err := pgRows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Provider, &u.GoogleID,
			&u.FullName, &u.AvatarUrl, &u.TotpSecret, &u.TotpEnabled, &u.TotpVerified,
			&u.TotpLastVerifiedAt, &u.InviteToken, &u.InviteExpiresAt, &u.Active,
			&u.CreatedAt, &u.UpdatedAt, &u.Status, &u.DeletedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, toStoreUser(u))
	}
	return out, pgRows.Err()
}

// ─── Sessions ─────────────────────────────────────────────────────────────────

func (s *Store) CreateSession(ctx context.Context, sess store.Session) error {
	// id is omitted — Postgres DEFAULT gen_random_uuid() assigns it.
	return s.q.CreateSession(ctx, CreateSessionParams{
		UserID:    sess.User_id,
		Token:     sess.Token,
		IpAddress: sess.IP_address,
		UserAgent: sess.User_agent,
		ExpiresAt: pgtype.Timestamptz{Time: sess.Expires_at, Valid: true},
	})
}

func (s *Store) GetSession(ctx context.Context, token string) (store.Session, error) {
	sess, err := s.q.GetSession(ctx, token)
	if err != nil {
		return store.Session{}, err
	}
	return toStoreSession(sess), nil
}

func (s *Store) TouchSession(ctx context.Context, token string, now time.Time) error {
	return s.q.TouchSession(ctx, TouchSessionParams{
		Token:        token,
		LastActiveAt: pgtype.Timestamptz{Time: now, Valid: true},
	})
}

func (s *Store) InvalidateSession(ctx context.Context, token string) error {
	return s.q.InvalidateSession(ctx, token)
}

func (s *Store) InvalidateAllUserSessions(ctx context.Context, user_id string) error {
	return s.q.InvalidateAllUserSessions(ctx, user_id)
}

func (s *Store) ListSessionsByUser(ctx context.Context, user_id string) ([]store.Session, error) {
	rows, err := s.q.ListSessionsByUser(ctx, user_id)
	if err != nil {
		return nil, err
	}
	out := make([]store.Session, len(rows))
	for i, r := range rows {
		out[i] = toStoreSession(r)
	}
	return out, nil
}

// ─── Brands ───────────────────────────────────────────────────────────────────

func (s *Store) CreateBrand(ctx context.Context, b store.Brand) error {
	return s.q.CreateBrand(ctx, CreateBrandParams{
		ID:          b.ID,
		Name:        b.Name,
		Shortcode:   b.Shortcode,
		Industry:    b.Industry,
		Description: b.Description,
		Website:     b.Website,
		Status:      b.Status,
		LogoUrl:     b.Logo_url,
	})
}

func (s *Store) GetBrandByID(ctx context.Context, id string) (store.Brand, error) {
	b, err := s.q.GetBrandByID(ctx, id)
	if err != nil {
		return store.Brand{}, err
	}
	return toStoreBrand(b), nil
}

func (s *Store) GetBrandByShortcode(ctx context.Context, shortcode string) (store.Brand, error) {
	b, err := s.q.GetBrandByShortcode(ctx, shortcode)
	if err != nil {
		return store.Brand{}, err
	}
	return toStoreBrand(b), nil
}

func (s *Store) ListBrands(ctx context.Context, f store.BrandFilter) ([]store.Brand, error) {
	rows, err := s.q.ListBrands(ctx, ListBrandsParams{
		Column1: f.Status,
		Column2: int32(f.Limit),
		Column3: int32(f.Offset),
	})
	if err != nil {
		return nil, err
	}
	out := make([]store.Brand, len(rows))
	for i, r := range rows {
		out[i] = toStoreBrand(r)
	}
	return out, nil
}

func (s *Store) UpdateBrand(ctx context.Context, id string, p store.BrandPatch) error {
	args := pgx.NamedArgs{"id": id}
	sets := []string{"updated_at = NOW()"}
	if p.Name != nil {
		sets = append(sets, "name = @name")
		args["name"] = *p.Name
	}
	if p.Industry != nil {
		sets = append(sets, "industry = @industry")
		args["industry"] = *p.Industry
	}
	if p.Description != nil {
		sets = append(sets, "description = @description")
		args["description"] = *p.Description
	}
	if p.Website != nil {
		sets = append(sets, "website = @website")
		args["website"] = *p.Website
	}
	if p.Logo_url != nil {
		sets = append(sets, "logo_url = @logo_url")
		args["logo_url"] = *p.Logo_url
	}
	if p.Status != nil {
		sets = append(sets, "status = @status")
		args["status"] = *p.Status
	}
	_, err := s.pool.Exec(ctx,
		"UPDATE brands SET "+strings.Join(sets, ", ")+" WHERE id = @id", args)
	return err
}

func (s *Store) ShortcodeExists(ctx context.Context, shortcode string) (bool, error) {
	return s.q.ShortcodeExists(ctx, shortcode)
}

// ─── Brand Contacts ───────────────────────────────────────────────────────────

func (s *Store) CreateBrandContact(ctx context.Context, c store.BrandContact) error {
	return s.q.CreateBrandContact(ctx, CreateBrandContactParams{
		ID:                 c.ID,
		BrandID:            c.Brand_id,
		FirstName:          c.First_name,
		LastName:           c.Last_name,
		Role:               c.Role,
		Email:              c.Email,
		WhatsappNumber:     c.Whatsapp_number,
		ViewerToken:        c.Viewer_token,
		AccessPasswordHash: c.Access_password_hash,
		TokenActive:        c.Token_active,
	})
}

func (s *Store) GetBrandContactByID(ctx context.Context, id string) (store.BrandContact, error) {
	c, err := s.q.GetBrandContactByID(ctx, id)
	if err != nil {
		return store.BrandContact{}, err
	}
	return toStoreBrandContact(c), nil
}

func (s *Store) GetBrandContactByViewerToken(ctx context.Context, token string) (store.BrandContact, error) {
	c, err := s.q.GetBrandContactByViewerToken(ctx, token)
	if err != nil {
		return store.BrandContact{}, err
	}
	return toStoreBrandContact(c), nil
}

func (s *Store) ListBrandContacts(ctx context.Context, brand_id string) ([]store.BrandContact, error) {
	rows, err := s.q.ListBrandContacts(ctx, brand_id)
	if err != nil {
		return nil, err
	}
	out := make([]store.BrandContact, len(rows))
	for i, r := range rows {
		out[i] = toStoreBrandContact(r)
	}
	return out, nil
}

func (s *Store) UpdateBrandContact(ctx context.Context, id string, p store.BrandContactPatch) error {
	args := pgx.NamedArgs{"id": id}
	sets := []string{"updated_at = NOW()"}
	if p.First_name != nil {
		sets = append(sets, "first_name = @first_name")
		args["first_name"] = *p.First_name
	}
	if p.Last_name != nil {
		sets = append(sets, "last_name = @last_name")
		args["last_name"] = *p.Last_name
	}
	if p.Role != nil {
		sets = append(sets, "role = @role")
		args["role"] = *p.Role
	}
	if p.Email != nil {
		sets = append(sets, "email = @email")
		args["email"] = *p.Email
	}
	if p.Whatsapp_number != nil {
		sets = append(sets, "whatsapp_number = @whatsapp_number")
		args["whatsapp_number"] = *p.Whatsapp_number
	}
	if p.Token_active != nil {
		sets = append(sets, "token_active = @token_active")
		args["token_active"] = *p.Token_active
	}
	_, err := s.pool.Exec(ctx,
		"UPDATE brand_contacts SET "+strings.Join(sets, ", ")+" WHERE id = @id", args)
	return err
}

func (s *Store) DeactivateBrandContact(ctx context.Context, id string) error {
	return s.q.DeactivateBrandContact(ctx, id)
}

func (s *Store) RegenerateBrandContactPassword(ctx context.Context, id string, new_hash string) error {
	return s.q.RegenerateBrandContactPassword(ctx, RegenerateBrandContactPasswordParams{
		ID:                 id,
		AccessPasswordHash: new_hash,
	})
}

func (s *Store) ValidateBrandContactAccess(ctx context.Context, token, password string) (store.BrandContact, error) {
	c, err := s.q.GetBrandContactForAuth(ctx, token)
	if err != nil {
		return store.BrandContact{}, errors.New("not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(c.AccessPasswordHash), []byte(password)); err != nil {
		return store.BrandContact{}, errors.New("invalid credentials")
	}
	return toStoreBrandContact(c), nil
}

// ─── Talents ──────────────────────────────────────────────────────────────────

func (s *Store) CreateTalent(ctx context.Context, t store.Talent) error {
	return s.q.CreateTalent(ctx, CreateTalentParams{
		ID:               t.ID,
		UserID:           t.User_id,
		Category:         string(t.Category),
		Status:           string(t.Status),
		Skills:           t.Skills,
		RatePerDay:       t.Rate_per_day,
		MaxTier:          int32(t.Max_tier),
		Bio:              t.Bio,
		PortfolioUrl:     t.Portfolio_url,
		ReportCompliance: t.Report_compliance,
	})
}

func (s *Store) GetTalentByID(ctx context.Context, id string) (store.Talent, error) {
	t, err := s.q.GetTalentByID(ctx, id)
	if err != nil {
		return store.Talent{}, err
	}
	return toStoreTalent(t), nil
}

func (s *Store) GetTalentByUserID(ctx context.Context, user_id string) (store.Talent, error) {
	t, err := s.q.GetTalentByUserID(ctx, user_id)
	if err != nil {
		return store.Talent{}, err
	}
	return toStoreTalent(t), nil
}

func (s *Store) ListTalents(ctx context.Context, f store.TalentFilter) ([]store.Talent, error) {
	rows, err := s.q.ListTalents(ctx, ListTalentsParams{
		Column1: f.Status,
		Column2: f.Category,
		Column3: int32(f.Limit),
		Column4: int32(f.Offset),
	})
	if err != nil {
		return nil, err
	}
	out := make([]store.Talent, len(rows))
	for i, r := range rows {
		out[i] = toStoreTalent(r)
	}
	return out, nil
}

func (s *Store) UpdateTalent(ctx context.Context, id string, p store.TalentPatch) error {
	args := pgx.NamedArgs{"id": id}
	sets := []string{"updated_at = NOW()"}
	if p.Status != nil {
		sets = append(sets, "status = @status")
		args["status"] = *p.Status
	}
	if p.Category != nil {
		sets = append(sets, "category = @category")
		args["category"] = *p.Category
	}
	if p.Rate_per_day != nil {
		sets = append(sets, "rate_per_day = @rate_per_day")
		args["rate_per_day"] = *p.Rate_per_day
	}
	if p.Max_tier != nil {
		sets = append(sets, "max_tier = @max_tier")
		args["max_tier"] = *p.Max_tier
	}
	if p.Skills != nil {
		sets = append(sets, "skills = @skills")
		args["skills"] = p.Skills
	}
	if p.Bio != nil {
		sets = append(sets, "bio = @bio")
		args["bio"] = *p.Bio
	}
	if p.Portfolio_url != nil {
		sets = append(sets, "portfolio_url = @portfolio_url")
		args["portfolio_url"] = *p.Portfolio_url
	}
	if p.Report_compliance != nil {
		sets = append(sets, "report_compliance = @report_compliance")
		args["report_compliance"] = *p.Report_compliance
	}
	tag, err := s.pool.Exec(ctx,
		"UPDATE talents SET "+strings.Join(sets, ", ")+" WHERE id = @id", args)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("not_found")
	}
	return nil
}

func (s *Store) ListAllTalents(ctx context.Context) ([]store.Talent, error) {
	rows, err := s.q.ListAllTalents(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]store.Talent, len(rows))
	for i, r := range rows {
		out[i] = toStoreTalent(r)
	}
	return out, nil
}

// ─── Campaigns ────────────────────────────────────────────────────────────────

func (s *Store) CreateCampaign(ctx context.Context, c store.Campaign) error {
	return s.q.CreateCampaign(ctx, CreateCampaignParams{
		ID:               c.ID,
		HumanID:          c.Human_id,
		BrandID:          c.Brand_id,
		Name:             c.Name,
		Status:           string(c.Status),
		CampaignType:     string(c.Campaign_type),
		TotalBudget:      c.Total_budget,
		RemainingBudget:  c.Remaining_budget,
		MarketCap:        c.Market_cap,
		Audience:         c.Audience,
		TargetCpa:        c.Target_cpa,
		MaxCpa:           c.Max_cpa,
		UrgencyLevel:     string(c.Urgency_level),
		CycleLength:      int32(c.Cycle_length),
		CreatorsAllowed:  c.Creators_allowed,
		StartDate:        pgtype.Timestamptz{Time: c.Start_date, Valid: true},
		EndDate:          pgtype.Timestamptz{Time: c.End_date, Valid: true},
	})
}

func (s *Store) GetCampaignByID(ctx context.Context, id string) (store.Campaign, error) {
	c, err := s.q.GetCampaignByID(ctx, id)
	if err != nil {
		return store.Campaign{}, err
	}
	return toStoreCampaign(c), nil
}

func (s *Store) GetCampaignByHumanID(ctx context.Context, human_id string) (store.Campaign, error) {
	c, err := s.q.GetCampaignByHumanID(ctx, human_id)
	if err != nil {
		return store.Campaign{}, err
	}
	return toStoreCampaign(c), nil
}

func (s *Store) ListCampaigns(ctx context.Context, f store.CampaignFilter) ([]store.Campaign, error) {
	rows, err := s.q.ListCampaigns(ctx, ListCampaignsParams{
		Column1: f.Status,
		Column2: f.Brand_id,
		Column3: int32(f.Limit),
		Column4: int32(f.Offset),
	})
	if err != nil {
		return nil, err
	}
	out := make([]store.Campaign, len(rows))
	for i, r := range rows {
		out[i] = toStoreCampaign(r)
	}
	return out, nil
}

func (s *Store) ListActiveCampaigns(ctx context.Context) ([]store.Campaign, error) {
	rows, err := s.q.ListActiveCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]store.Campaign, len(rows))
	for i, r := range rows {
		out[i] = toStoreCampaign(r)
	}
	return out, nil
}

func (s *Store) UpdateCampaign(ctx context.Context, id string, p store.CampaignPatch) error {
	args := pgx.NamedArgs{"id": id}
	sets := []string{"updated_at = NOW()"}
	if p.Name != nil {
		sets = append(sets, "name = @name")
		args["name"] = *p.Name
	}
	if p.Status != nil {
		sets = append(sets, "status = @status")
		args["status"] = *p.Status
	}
	if p.Urgency_level != nil {
		sets = append(sets, "urgency_level = @urgency_level")
		args["urgency_level"] = *p.Urgency_level
	}
	if p.Cycle_length != nil {
		sets = append(sets, "cycle_length = @cycle_length")
		args["cycle_length"] = *p.Cycle_length
	}
	if p.End_date != nil {
		sets = append(sets, "end_date = @end_date")
		args["end_date"] = *p.End_date
	}
	if p.Creators_allowed != nil {
		sets = append(sets, "creators_allowed = @creators_allowed")
		args["creators_allowed"] = *p.Creators_allowed
	}
	_, err := s.pool.Exec(ctx,
		"UPDATE campaigns SET "+strings.Join(sets, ", ")+" WHERE id = @id", args)
	return err
}

func (s *Store) DecrementRemainingBudget(ctx context.Context, campaign_id string, amount float64) error {
	return s.q.DecrementRemainingBudget(ctx, DecrementRemainingBudgetParams{
		ID:              campaign_id,
		RemainingBudget: amount,
	})
}

func (s *Store) NextCampaignHumanID(ctx context.Context, brand_id string) (string, error) {
	brand, err := s.q.GetBrandByID(ctx, brand_id)
	if err != nil {
		return "", err
	}
	count, err := s.q.CountCampaignsByBrandMonth(ctx, brand.ID)
	if err != nil {
		return "", err
	}
	yy := time.Now().UTC().Year() % 100
	return fmt.Sprintf("%s-%02d-%02d", brand.Shortcode, yy, count+1), nil
}

func (s *Store) GetCampaignsByManagerID(ctx context.Context, manager_id string) ([]store.Campaign, error) {
	rows, err := s.q.GetCampaignsByManagerID(ctx, manager_id)
	if err != nil {
		return nil, err
	}
	out := make([]store.Campaign, len(rows))
	for i, r := range rows {
		out[i] = toStoreCampaign(r)
	}
	return out, nil
}

func (s *Store) ListManagersByCampaignID(ctx context.Context, campaign_id string) ([]store.User, error) {
	rows, err := s.q.ListManagersByCampaignID(ctx, campaign_id)
	if err != nil {
		return nil, err
	}
	out := make([]store.User, len(rows))
	for i, r := range rows {
		out[i] = toStoreUser(r)
	}
	return out, nil
}

func (s *Store) AssignManagerToCampaign(ctx context.Context, manager_id, campaign_id, assigned_by string) error {
	return s.q.AssignManagerToCampaign(ctx, AssignManagerToCampaignParams{
		ManagerID:  manager_id,
		CampaignID: campaign_id,
		AssignedBy: assigned_by,
	})
}

func (s *Store) UnassignManagerFromCampaign(ctx context.Context, manager_id, campaign_id string) error {
	return s.q.UnassignManagerFromCampaign(ctx, UnassignManagerFromCampaignParams{
		ManagerID:  manager_id,
		CampaignID: campaign_id,
	})
}

func (s *Store) TryRecordEmailDispatch(ctx context.Context, entity_type, entity_id, template_key, recipient string) (bool, error) {
	n, err := s.q.InsertEmailDispatch(ctx, InsertEmailDispatchParams{
		EntityType:  entity_type,
		EntityID:    entity_id,
		TemplateKey: template_key,
		Recipient:   recipient,
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) EmailDispatchExists(ctx context.Context, entity_type, entity_id, template_key, recipient string) (bool, error) {
	return s.q.EmailDispatchExists(ctx, EmailDispatchExistsParams{
		EntityType:  entity_type,
		EntityID:    entity_id,
		TemplateKey: template_key,
		Recipient:   recipient,
	})
}

// ─── Cycles ───────────────────────────────────────────────────────────────────

func (s *Store) CreateCycle(ctx context.Context, c store.Cycle) error {
	kpb, _ := json.Marshal(c.KPB_config)
	return s.q.CreateCycle(ctx, CreateCycleParams{
		ID:              c.ID,
		HumanID:         c.Human_id,
		CampaignID:      c.Campaign_id,
		CycleNumber:     int32(c.Cycle_number),
		Status:          string(c.Status),
		CycleBudget:     c.Cycle_budget,
		RemainingBudget: c.Remaining_budget,
		CycleObjective:  c.Cycle_objective,
		CampaignType:    string(c.Campaign_type),
		KpbConfig:       kpb,
		ZFactor:         c.Z_factor,
		StartDate:       pgtype.Timestamptz{Time: c.Start_date, Valid: true},
		EndDate:         pgtype.Timestamptz{Time: c.End_date, Valid: true},
	})
}

func (s *Store) GetCycleByID(ctx context.Context, id string) (store.Cycle, error) {
	c, err := s.q.GetCycleByID(ctx, id)
	if err != nil {
		return store.Cycle{}, err
	}
	return toStoreCycle(c), nil
}

func (s *Store) ListCyclesByCampaign(ctx context.Context, campaign_id string) ([]store.Cycle, error) {
	rows, err := s.q.ListCyclesByCampaign(ctx, campaign_id)
	if err != nil {
		return nil, err
	}
	out := make([]store.Cycle, len(rows))
	for i, r := range rows {
		out[i] = toStoreCycle(r)
	}
	return out, nil
}

func (s *Store) GetActiveCycles(ctx context.Context) ([]store.Cycle, error) {
	rows, err := s.q.GetActiveCycles(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]store.Cycle, len(rows))
	for i, r := range rows {
		out[i] = toStoreCycle(r)
	}
	return out, nil
}

func (s *Store) UpdateCycle(ctx context.Context, id string, p store.CyclePatch) error {
	args := pgx.NamedArgs{"id": id}
	sets := []string{"updated_at = NOW()"}
	if p.Status != nil {
		sets = append(sets, "status = @status")
		args["status"] = *p.Status
	}
	if p.Cycle_budget != nil {
		sets = append(sets, "cycle_budget = @cycle_budget")
		args["cycle_budget"] = *p.Cycle_budget
	}
	if p.Cycle_objective != nil {
		sets = append(sets, "cycle_objective = @cycle_objective")
		args["cycle_objective"] = *p.Cycle_objective
	}
	if p.Z_factor != nil {
		sets = append(sets, "z_factor = @z_factor")
		args["z_factor"] = *p.Z_factor
	}
	if p.End_date != nil {
		sets = append(sets, "end_date = @end_date")
		args["end_date"] = *p.End_date
	}
	_, err := s.pool.Exec(ctx,
		"UPDATE cycles SET "+strings.Join(sets, ", ")+" WHERE id = @id", args)
	return err
}

func (s *Store) CloseCycle(ctx context.Context, id string, spend float64) error {
	return s.q.CloseCycle(ctx, CloseCycleParams{ID: id, RemainingBudget: spend})
}

// ─── Budget Slots ─────────────────────────────────────────────────────────────

func (s *Store) CreateBudgetSlots(ctx context.Context, slots []store.BudgetSlot) error {
	for _, slot := range slots {
		var tid pgtype.Text
		if slot.Talent_id != "" {
			tid = pgtype.Text{String: slot.Talent_id, Valid: true}
		}
		_, err := s.pool.Exec(ctx,
			`INSERT INTO budget_slots (id, cycle_id, tier_value, slot_index, allocated, talent_id)
			 VALUES ($1,$2,$3,$4,$5,$6)`,
			slot.ID, slot.Cycle_id, slot.Tier_value, slot.Slot_index, slot.Allocated, tid,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListSlotsByCycle(ctx context.Context, cycle_id string) ([]store.BudgetSlot, error) {
	rows, err := s.q.ListSlotsByCycle(ctx, cycle_id)
	if err != nil {
		return nil, err
	}
	out := make([]store.BudgetSlot, len(rows))
	for i, r := range rows {
		out[i] = toStoreBudgetSlot(r)
	}
	return out, nil
}

func (s *Store) AssignSlot(ctx context.Context, slot_id, talent_id string) error {
	return s.q.AssignSlot(ctx, AssignSlotParams{
		ID:       slot_id,
		TalentID: pgtype.Text{String: talent_id, Valid: true},
	})
}

// ─── Assignments ──────────────────────────────────────────────────────────────

func (s *Store) CreateAssignment(ctx context.Context, a store.TalentAssignment) error {
	return s.q.CreateAssignment(ctx, CreateAssignmentParams{
		TalentID:         a.Talent_id,
		CampaignID:       a.Campaign_id,
		CycleID:          a.Cycle_id,
		SlotID:           a.Slot_id,
		RoleLabel:        a.Role_label,
		Status:           a.Status,
		AssignmentSource: string(a.Assignment_source),
		PdcMode:          string(a.PDC_mode),
		PdcValue:         a.PDC_value,
		MatchScore:       a.Match_score,
		MatchDm:          a.Match_dm,
		MatchGp:          a.Match_gp,
		MatchOh:          a.Match_oh,
		PinnedTier:       int32(a.Pinned_tier),
		EffectiveTier:    int32(a.Effective_tier),
		BreakoutFlag:     a.Breakout_flag,
	})
}

func (s *Store) GetAssignment(ctx context.Context, talent_id, cycle_id string) (store.TalentAssignment, error) {
	a, err := s.q.GetAssignment(ctx, GetAssignmentParams{TalentID: talent_id, CycleID: cycle_id})
	if err != nil {
		return store.TalentAssignment{}, err
	}
	return toStoreTalentAssignment(a), nil
}

func (s *Store) ListAssignedTalents(ctx context.Context, cycle_id string) ([]store.TalentAssignment, error) {
	rows, err := s.q.ListAssignedTalents(ctx, cycle_id)
	if err != nil {
		return nil, err
	}
	out := make([]store.TalentAssignment, len(rows))
	for i, r := range rows {
		out[i] = toStoreTalentAssignment(r)
	}
	return out, nil
}

func (s *Store) ListAssignmentsByTalent(ctx context.Context, talent_id string) ([]store.TalentAssignment, error) {
	rows, err := s.q.ListAssignmentsByTalent(ctx, talent_id)
	if err != nil {
		return nil, err
	}
	out := make([]store.TalentAssignment, len(rows))
	for i, r := range rows {
		out[i] = toStoreTalentAssignment(r)
	}
	return out, nil
}

func (s *Store) UpdateAssignment(ctx context.Context, talent_id, cycle_id string, p store.AssignmentPatch) error {
	args := pgx.NamedArgs{"talent_id": talent_id, "cycle_id": cycle_id}
	sets := []string{}
	if p.Status != nil {
		sets = append(sets, "status = @status")
		args["status"] = *p.Status
	}
	if p.Breakout_flag != nil {
		sets = append(sets, "breakout_flag = @breakout_flag")
		args["breakout_flag"] = *p.Breakout_flag
	}
	if p.Match_score != nil {
		sets = append(sets, "match_score = @match_score")
		args["match_score"] = *p.Match_score
	}
	if len(sets) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		"UPDATE talent_assignments SET "+strings.Join(sets, ", ")+
			" WHERE talent_id = @talent_id AND cycle_id = @cycle_id", args)
	return err
}

// ─── Tracking Links ───────────────────────────────────────────────────────────

func (s *Store) CreateTrackingLink(ctx context.Context, l store.TrackingLink) error {
	return s.q.CreateTrackingLink(ctx, CreateTrackingLinkParams{
		ID:         l.ID,
		CampaignID: l.Campaign_id,
		CycleID:    l.Cycle_id,
		TalentID:   l.Talent_id,
		Token:      l.Token,
		Active:     l.Active,
	})
}

func (s *Store) GetTrackingLinkByToken(ctx context.Context, token string) (store.TrackingLink, error) {
	l, err := s.q.GetTrackingLinkByToken(ctx, token)
	if err != nil {
		return store.TrackingLink{}, err
	}
	return toStoreTrackingLink(l), nil
}

func (s *Store) ListTrackingLinksByCycle(ctx context.Context, cycle_id string) ([]store.TrackingLink, error) {
	rows, err := s.q.ListTrackingLinksByCycle(ctx, cycle_id)
	if err != nil {
		return nil, err
	}
	out := make([]store.TrackingLink, len(rows))
	for i, r := range rows {
		out[i] = toStoreTrackingLink(r)
	}
	return out, nil
}

// ─── Conversion Events ────────────────────────────────────────────────────────

func (s *Store) LogConversionEvent(ctx context.Context, e store.ConversionEvent) error {
	return s.q.LogConversionEvent(ctx, LogConversionEventParams{
		ID:              e.ID,
		LinkToken:       e.Link_token,
		TalentID:        e.Talent_id,
		CampaignID:      e.Campaign_id,
		CycleID:         e.Cycle_id,
		PipelineType:    string(e.Pipeline_type),
		EventType:       e.Event_type,
		KpbType:         pgtype.Text{String: e.KPB_type, Valid: e.KPB_type != ""},
		ValidLead:       e.Valid_lead,
		IdempotencyKey:  e.Idempotency_key,
		OccurredAt:      pgtype.Timestamptz{Time: e.Occurred_at, Valid: true},
	})
}

func (s *Store) GetDailyConversionCount(ctx context.Context, talent_id, cycle_id string, date time.Time) (float64, error) {
	return s.q.GetDailyConversionCount(ctx, GetDailyConversionCountParams{
		TalentID: talent_id,
		CycleID:  cycle_id,
		Timezone: date,
	})
}

func (s *Store) GetTotalConversions(ctx context.Context, cycle_id string) (float64, error) {
	return s.q.GetTotalConversions(ctx, cycle_id)
}

func (s *Store) GetTalentConversions(ctx context.Context, talent_id, cycle_id string) (float64, error) {
	return s.q.GetTalentConversions(ctx, GetTalentConversionsParams{
		TalentID: talent_id,
		CycleID:  cycle_id,
	})
}

func (s *Store) FlagFallbackConversions(ctx context.Context, cycle_id string, day time.Time) error {
	return s.q.FlagFallbackConversions(ctx, FlagFallbackConversionsParams{
		CycleID:  cycle_id,
		Timezone: day,
	})
}

func (s *Store) LockFallbackConversions(ctx context.Context, cycle_id string) error {
	return s.q.LockFallbackConversions(ctx, cycle_id)
}

// ─── Algo State ───────────────────────────────────────────────────────────────

func (s *Store) GetCycleState(ctx context.Context, talent_id, cycle_id string) (store.CycleState, error) {
	st, err := s.q.GetCycleState(ctx, GetCycleStateParams{TalentID: talent_id, CycleID: cycle_id})
	if err != nil {
		return store.CycleState{}, err
	}
	return toStoreCycleState(st), nil
}

func (s *Store) UpsertCycleState(ctx context.Context, cs store.CycleState) error {
	return s.q.UpsertCycleState(ctx, UpsertCycleStateParams{
		TalentID:     cs.Talent_id,
		CycleID:      cs.Cycle_id,
		CycleNumber:  int32(cs.Cycle_number),
		PdcAllocated: cs.PDC_allocated,
		PdcNext:      cs.PDC_next,
		Pattern:      cs.Pattern,
		DeltaSt:      cs.Delta_st,
		UpdatedAt:    pgtype.Timestamptz{Time: cs.Updated_at, Valid: true},
	})
}

func (s *Store) GetDailyOutputs(ctx context.Context, talent_id, cycle_id string) ([]float64, error) {
	return s.q.GetDailyOutputs(ctx, GetDailyOutputsParams{
		TalentID: talent_id,
		CycleID:  cycle_id,
	})
}

func (s *Store) GetTalentBaseline(ctx context.Context, talent_id string) (store.TalentBaseline, error) {
	b, err := s.q.GetTalentBaseline(ctx, talent_id)
	if err != nil {
		return store.TalentBaseline{}, err
	}
	return toStoreTalentBaseline(b), nil
}

func (s *Store) UpsertTalentBaseline(ctx context.Context, b store.TalentBaseline) error {
	return s.q.UpsertTalentBaseline(ctx, UpsertTalentBaselineParams{
		TalentID:  b.Talent_id,
		AlphaLt:   b.Alpha_lt,
		BetaLt:    b.Beta_lt,
		LambdaLt:  b.Lambda_lt,
		SigmaHist: b.Sigma_hist,
		DeltaLt:   b.Delta_lt,
	})
}

func (s *Store) GetCategoryBaseline(ctx context.Context, category string) (algo.CategoryBaseline, error) {
	row, err := s.q.GetCategoryBaseline(ctx, category)
	if err != nil {
		return algo.CategoryBaseline{}, err
	}
	return algo.CategoryBaseline{Category: row.Category, Median: row.Median}, nil
}

func (s *Store) GetTalentTodayOutput(ctx context.Context, talent_id string, date time.Time) (float64, error) {
	return s.q.GetTalentTodayOutput(ctx, GetTalentTodayOutputParams{
		TalentID: talent_id,
		Timezone: date,
	})
}

func (s *Store) GetTalentOutputWindow(ctx context.Context, talent_id string, days int) ([]float64, error) {
	return s.q.GetTalentOutputWindow(ctx, GetTalentOutputWindowParams{
		TalentID: talent_id,
		Column2:  int32(days),
	})
}

// ─── Payout Records ───────────────────────────────────────────────────────────

func (s *Store) CreatePayoutRecord(ctx context.Context, p store.PayoutRecord) error {
	return s.q.CreatePayoutRecord(ctx, CreatePayoutRecordParams{
		ID:               p.ID,
		TalentID:         p.Talent_id,
		CycleID:          p.Cycle_id,
		CampaignID:       p.Campaign_id,
		PipelineType:     string(p.Pipeline_type),
		Status:           string(p.Status),
		AllocatedBudget:  p.Allocated_budget,
		GrossBase:        p.Gross_base,
		KpbTotal:         p.KPB_total,
		GrossTotal:       p.Gross_total,
		CostPerUnit:      p.Cost_per_unit,
		CapApplied:       p.Cap_applied,
		CapExceeded:      p.Cap_exceeded,
		ExcessForfeited:  p.Excess_forfeited,
		CommissionRate:   p.Commission_rate,
		CommissionAmount: p.Commission_amount,
		ENet:             p.E_net,
		ScaleFactor:      p.Scale_factor,
		FinalPayout:      p.Final_payout,
		KpbPoolSource:    p.KPB_pool_source,
		FallbackFlagged:  p.Fallback_flagged,
		ReportSubmitted:  p.Report_submitted,
		AdminOverride:    p.Admin_override,
		OverrideReason:   p.Override_reason,
		ApprovedBy:       pgtype.Text{String: p.Approved_by, Valid: p.Approved_by != ""},
		ApprovedAt:       ptsPtr(&p.Approved_at),
		PaidAt:           ptsPtr(&p.Paid_at),
		FailureReason:    p.Failure_reason,
		RetryCount:       int32(p.Retry_count),
	})
}

func (s *Store) GetPayoutRecord(ctx context.Context, talent_id, cycle_id string) (store.PayoutRecord, error) {
	p, err := s.q.GetPayoutRecord(ctx, GetPayoutRecordParams{TalentID: talent_id, CycleID: cycle_id})
	if err != nil {
		return store.PayoutRecord{}, err
	}
	return toStorePayoutRecord(p), nil
}

func (s *Store) ListPayoutsByCycle(ctx context.Context, cycle_id string) ([]store.PayoutRecord, error) {
	rows, err := s.q.ListPayoutsByCycle(ctx, cycle_id)
	if err != nil {
		return nil, err
	}
	out := make([]store.PayoutRecord, len(rows))
	for i, r := range rows {
		out[i] = toStorePayoutRecord(r)
	}
	return out, nil
}

func (s *Store) ListPayoutsByTalent(ctx context.Context, talent_id string) ([]store.PayoutRecord, error) {
	rows, err := s.q.ListPayoutsByTalent(ctx, talent_id)
	if err != nil {
		return nil, err
	}
	out := make([]store.PayoutRecord, len(rows))
	for i, r := range rows {
		out[i] = toStorePayoutRecord(r)
	}
	return out, nil
}

func (s *Store) UpdatePayoutRecord(ctx context.Context, id string, p store.PayoutPatch) error {
	args := pgx.NamedArgs{"id": id}
	sets := []string{"updated_at = NOW()"}
	if p.Status != nil {
		sets = append(sets, "status = @status")
		args["status"] = string(*p.Status)
	}
	if p.Admin_override != nil {
		sets = append(sets, "admin_override = @admin_override")
		args["admin_override"] = *p.Admin_override
	}
	if p.Override_reason != nil {
		sets = append(sets, "override_reason = @override_reason")
		args["override_reason"] = *p.Override_reason
	}
	if p.Approved_by != nil {
		sets = append(sets, "approved_by = @approved_by")
		args["approved_by"] = *p.Approved_by
	}
	if p.Approved_at != nil {
		sets = append(sets, "approved_at = @approved_at")
		args["approved_at"] = *p.Approved_at
	}
	if p.Paid_at != nil {
		sets = append(sets, "paid_at = @paid_at")
		args["paid_at"] = *p.Paid_at
	}
	if p.Failure_reason != nil {
		sets = append(sets, "failure_reason = @failure_reason")
		args["failure_reason"] = *p.Failure_reason
	}
	if p.Retry_count != nil {
		sets = append(sets, "retry_count = @retry_count")
		args["retry_count"] = *p.Retry_count
	}
	if p.Report_submitted != nil {
		sets = append(sets, "report_submitted = @report_submitted")
		args["report_submitted"] = *p.Report_submitted
	}
	_, err := s.pool.Exec(ctx,
		"UPDATE payout_records SET "+strings.Join(sets, ", ")+" WHERE id = @id", args)
	return err
}

// ─── Campaign Viewers ─────────────────────────────────────────────────────────

func (s *Store) CreateViewer(ctx context.Context, v store.CampaignViewer) error {
	return s.q.CreateViewer(ctx, CreateViewerParams{
		ID:         v.ID,
		CampaignID: v.Campaign_id,
		Token:      v.Token,
		Name:       v.Name,
		Active:     v.Active,
	})
}

func (s *Store) GetViewerByToken(ctx context.Context, token string) (store.CampaignViewer, error) {
	v, err := s.q.GetViewerByToken(ctx, token)
	if err != nil {
		return store.CampaignViewer{}, err
	}
	return toStoreCampaignViewer(v), nil
}

func (s *Store) ListViewersByCampaign(ctx context.Context, campaign_id string) ([]store.CampaignViewer, error) {
	rows, err := s.q.ListViewersByCampaign(ctx, campaign_id)
	if err != nil {
		return nil, err
	}
	out := make([]store.CampaignViewer, len(rows))
	for i, r := range rows {
		out[i] = toStoreCampaignViewer(r)
	}
	return out, nil
}

func (s *Store) UpdateViewer(ctx context.Context, id string, p store.ViewerPatch) error {
	args := pgx.NamedArgs{"id": id}
	sets := []string{}
	if p.Name != nil {
		sets = append(sets, "name = @name")
		args["name"] = *p.Name
	}
	if p.Active != nil {
		sets = append(sets, "active = @active")
		args["active"] = *p.Active
	}
	if len(sets) == 0 {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		"UPDATE campaign_viewers SET "+strings.Join(sets, ", ")+" WHERE id = @id", args)
	return err
}

func (s *Store) AddViewerPassword(ctx context.Context, p store.ViewerPassword) error {
	return s.q.AddViewerPassword(ctx, AddViewerPasswordParams{
		ID:           p.ID,
		ViewerID:     p.Viewer_id,
		PasswordHash: p.Password_hash,
		Label:        p.Label,
		Active:       p.Active,
	})
}

func (s *Store) ListViewerPasswords(ctx context.Context, viewer_id string) ([]store.ViewerPassword, error) {
	rows, err := s.q.ListViewerPasswords(ctx, viewer_id)
	if err != nil {
		return nil, err
	}
	out := make([]store.ViewerPassword, len(rows))
	for i, r := range rows {
		out[i] = toStoreViewerPassword(r)
	}
	return out, nil
}

func (s *Store) DeactivateViewerPassword(ctx context.Context, id string) error {
	return s.q.DeactivateViewerPassword(ctx, id)
}

func (s *Store) ValidateViewerPassword(ctx context.Context, viewer_id, password string) (bool, error) {
	rows, err := s.q.GetAllViewerPasswords(ctx, viewer_id)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(password)) == nil {
			return true, nil
		}
	}
	return false, nil
}

// ─── Audit Log ────────────────────────────────────────────────────────────────

func auditActorText(actor_id string) pgtype.Text {
	return pgtype.Text{String: actor_id, Valid: actor_id != ""}
}

func (s *Store) WriteAuditLog(ctx context.Context, entry store.AuditLog) error {
	return s.q.WriteAuditLog(ctx, WriteAuditLogParams{
		ID:          entry.ID,
		ActorID:     auditActorText(entry.Actor_id),
		ActionType:  entry.Action_type,
		EntityType:  entry.Entity_type,
		EntityID:    entry.Entity_id,
		BeforeState: entry.Before_state,
		AfterState:  entry.After_state,
		RequestID:   entry.Request_id,
		Seq:         pgtype.Int8{Int64: entry.Seq, Valid: entry.Seq != 0},
		PrevHash:    entry.Prev_hash,
		EntryHash:   entry.Entry_hash,
		Signature:   entry.Signature,
		ArchiveUri:  entry.Archive_uri,
		IpAddress:   entry.IP_address,
		UserAgent:   entry.User_agent,
	})
}

func (s *Store) ListAuditLog(ctx context.Context, entity_type, entity_id string) ([]store.AuditLog, error) {
	rows, err := s.q.ListAuditLog(ctx, ListAuditLogParams{
		EntityType: entity_type,
		EntityID:   entity_id,
	})
	if err != nil {
		return nil, err
	}
	out := make([]store.AuditLog, len(rows))
	for i, r := range rows {
		out[i] = toStoreAuditLog(r)
	}
	return out, nil
}

func (s *Store) GetAuditLogByID(ctx context.Context, id string) (store.AuditLog, error) {
	row, err := s.q.GetAuditLogByID(ctx, id)
	if err != nil {
		return store.AuditLog{}, err
	}
	return toStoreAuditLog(row), nil
}

func (s *Store) GetAuditChainTip(ctx context.Context) (string, int64, error) {
	tip, err := s.q.GetAuditChainTip(ctx)
	if err != nil {
		return "", 0, err
	}
	return tip.LastHash, tip.LastSeq, nil
}

// AppendAuditLog locks the chain tip, assigns seq from tip+1, inserts, and advances the tip.
// Caller must supply Prev_hash, Entry_hash, Signature, Seq matching tip+1.
func (s *Store) AppendAuditLog(ctx context.Context, entry store.AuditLog) (store.AuditLog, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return store.AuditLog{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.q.WithTx(tx)
	tip, err := qtx.GetAuditChainTipForUpdate(ctx)
	if err != nil {
		return store.AuditLog{}, err
	}
	if entry.Prev_hash == "" {
		entry.Prev_hash = tip.LastHash
	}
	if entry.Seq == 0 {
		entry.Seq = tip.LastSeq + 1
	}
	if entry.Prev_hash != tip.LastHash || entry.Seq != tip.LastSeq+1 {
		return store.AuditLog{}, fmt.Errorf("audit_chain_mismatch")
	}

	if err := qtx.WriteAuditLog(ctx, WriteAuditLogParams{
		ID:          entry.ID,
		ActorID:     auditActorText(entry.Actor_id),
		ActionType:  entry.Action_type,
		EntityType:  entry.Entity_type,
		EntityID:    entry.Entity_id,
		BeforeState: entry.Before_state,
		AfterState:  entry.After_state,
		RequestID:   entry.Request_id,
		Seq:         pgtype.Int8{Int64: entry.Seq, Valid: true},
		PrevHash:    entry.Prev_hash,
		EntryHash:   entry.Entry_hash,
		Signature:   entry.Signature,
		ArchiveUri:  entry.Archive_uri,
		IpAddress:   entry.IP_address,
		UserAgent:   entry.User_agent,
	}); err != nil {
		return store.AuditLog{}, err
	}
	if err := qtx.UpdateAuditChainTip(ctx, UpdateAuditChainTipParams{
		LastHash: entry.Entry_hash,
		LastSeq:  entry.Seq,
	}); err != nil {
		return store.AuditLog{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return store.AuditLog{}, err
	}
	entry.Created_at = time.Now().UTC()
	return entry, nil
}

func (s *Store) ListAuditLogFiltered(ctx context.Context, f store.AuditFilter) ([]store.AuditLog, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args := pgx.NamedArgs{
		"entity_type": f.Entity_type,
		"entity_id":   f.Entity_id,
		"actor_id":    f.Actor_id,
		"action_type": f.Action_type,
		"after_seq":   f.After_seq,
		"limit":       limit,
	}
	q := `SELECT id, actor_id, action_type, entity_type, entity_id, before_state, after_state,
		created_at, request_id, seq, prev_hash, entry_hash, signature, archive_uri, ip_address, user_agent
		FROM audit_log WHERE 1=1`
	if f.Entity_type != "" {
		q += " AND entity_type = @entity_type"
	}
	if f.Entity_id != "" {
		q += " AND entity_id = @entity_id"
	}
	if f.Actor_id != "" {
		q += " AND actor_id = @actor_id"
	}
	if f.Action_type != "" {
		q += " AND action_type = @action_type"
	}
	if f.From != nil {
		q += " AND created_at >= @from_ts"
		args["from_ts"] = *f.From
	}
	if f.To != nil {
		q += " AND created_at <= @to_ts"
		args["to_ts"] = *f.To
	}
	if f.After_seq > 0 {
		q += " AND seq < @after_seq"
	}
	q += " ORDER BY seq DESC LIMIT @limit"

	pgRows, err := s.pool.Query(ctx, q, args)
	if err != nil {
		return nil, err
	}
	defer pgRows.Close()
	var out []store.AuditLog
	for pgRows.Next() {
		var a AuditLog
		if err := pgRows.Scan(
			&a.ID, &a.ActorID, &a.ActionType, &a.EntityType, &a.EntityID,
			&a.BeforeState, &a.AfterState, &a.CreatedAt, &a.RequestID, &a.Seq,
			&a.PrevHash, &a.EntryHash, &a.Signature, &a.ArchiveUri, &a.IpAddress, &a.UserAgent,
		); err != nil {
			return nil, err
		}
		out = append(out, toStoreAuditLog(a))
	}
	return out, pgRows.Err()
}

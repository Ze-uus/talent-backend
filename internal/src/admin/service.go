package admin

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Ze-uus/talent-backend/internal/audit"
	"github.com/Ze-uus/talent-backend/internal/baseline"
	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/mail"
	"github.com/Ze-uus/talent-backend/internal/store"
	ws "github.com/Ze-uus/talent-backend/internal/websocket"
)

var (
	err_invalid_active_transition = errors.New("invalid_active_transition")
	err_talent_not_found          = errors.New("not_found")
)

type AdminService struct {
	st               store.Store
	talent_update_ch chan<- domain.TalentUpdateEvent
	log              *slog.Logger
	mail             *mail.Service
	auditor          *audit.Recorder
	delta_lt         float64
}

func New(s store.Store, talent_update_ch chan<- domain.TalentUpdateEvent, log *slog.Logger, mailSvc *mail.Service, auditor *audit.Recorder, delta_lt float64) *AdminService {
	if delta_lt <= 0 || delta_lt >= 1 {
		delta_lt = 0.97
	}
	return &AdminService{st: s, talent_update_ch: talent_update_ch, log: log, mail: mailSvc, auditor: auditor, delta_lt: delta_lt}
}

func (s *AdminService) emitTalentUpdate(talent_id, update_type string, shard domain.TalentUpdateShard) {
	ws.TryPush(s.talent_update_ch, domain.TalentUpdateEvent{
		Talent_id:   talent_id,
		Cycle_id:    shard.Cycle_id,
		Update_type: update_type,
		Shard:       shard,
	}, s.log, "talent_update_ch full, event dropped", "talent_id", talent_id, "update_type", update_type)
}

func (s *AdminService) notifyTalentUser(ctx context.Context, talent_id string, fn func(email, name string)) {
	if s.mail == nil || fn == nil {
		return
	}
	talent, err := s.st.GetTalentByID(ctx, talent_id)
	if err != nil {
		return
	}
	user, err := s.st.GetUserByID(ctx, talent.User_id)
	if err != nil {
		return
	}
	fn(user.Email, user.Full_name)
}

func (s *AdminService) record(ctx context.Context, action, entityType, entityID string, before, after any) {
	if s.auditor == nil {
		return
	}
	_ = s.auditor.Record(ctx, audit.Entry{
		Action:      action,
		Entity_type: entityType,
		Entity_id:   entityID,
		Before:      before,
		After:       after,
	})
}

// ─── User management ──────────────────────────────────────────────────────────

func (s *AdminService) ListUsers(ctx context.Context, filter store.UserFilter) ([]store.User, error) {
	return s.st.ListUsers(ctx, filter)
}

func (s *AdminService) GetUser(ctx context.Context, id string) (store.User, error) {
	return s.st.GetUserByID(ctx, id)
}

func (s *AdminService) PatchUser(ctx context.Context, id string, patch store.UserPatch) error {
	before, err := s.st.GetUserByID(ctx, id)
	if err != nil {
		return err
	}

	// Sync Status when Active is patched without an explicit Status.
	if patch.Active != nil && patch.Status == nil {
		wantActive := *patch.Active
		if wantActive {
			switch before.Status {
			case store.User_status_suspended, store.User_status_banned:
				st := string(store.User_status_active)
				patch.Status = &st
			case store.User_status_active:
				// already active
			default:
				return err_invalid_active_transition
			}
		} else {
			switch before.Status {
			case store.User_status_invited, store.User_status_deleted,
				store.User_status_pending, store.User_status_rejected:
				return err_invalid_active_transition
			case store.User_status_banned:
				// Keep banned; Active already false in patch.
			case store.User_status_active, store.User_status_suspended:
				st := string(store.User_status_suspended)
				patch.Status = &st
			default:
				st := string(store.User_status_suspended)
				patch.Status = &st
			}
		}
	}

	if err := s.st.UpdateUser(ctx, id, patch); err != nil {
		return err
	}
	after, _ := s.st.GetUserByID(ctx, id)
	if after.Status != store.User_status_active || (patch.Active != nil && !*patch.Active) {
		_ = s.st.InvalidateAllUserSessions(ctx, id)
	}
	s.record(ctx, "user_patched", "user", id, before, after)
	return nil
}

func (s *AdminService) setUserStatus(ctx context.Context, id string, status store.User_status, action string) error {
	before, err := s.st.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	active := status == store.User_status_active
	st := string(status)
	patch := store.UserPatch{Status: &st, Active: &active}
	if status == store.User_status_deleted {
		now := time.Now().UTC()
		patch.Deleted_at = &now
		active = false
		patch.Active = &active
	}
	if err := s.st.UpdateUser(ctx, id, patch); err != nil {
		return err
	}
	if status != store.User_status_active {
		_ = s.st.InvalidateAllUserSessions(ctx, id)
	}
	// Keep talents.status in sync for talent accounts.
	if before.Role == store.Role_talent {
		if talent, err := s.st.GetTalentByUserID(ctx, id); err == nil {
			switch status {
			case store.User_status_active, store.User_status_suspended,
				store.User_status_pending, store.User_status_rejected:
				ts := string(status)
				_ = s.st.UpdateTalent(ctx, talent.ID, store.TalentPatch{Status: &ts})
			case store.User_status_banned, store.User_status_deleted:
				ts := string(store.Status_suspended)
				_ = s.st.UpdateTalent(ctx, talent.ID, store.TalentPatch{Status: &ts})
			}
		}
	}
	after, _ := s.st.GetUserByID(ctx, id)
	s.record(ctx, action, "user", id, before, after)
	return nil
}

func (s *AdminService) SuspendUser(ctx context.Context, id string) error {
	return s.setUserStatus(ctx, id, store.User_status_suspended, "user_suspended")
}

func (s *AdminService) BanUser(ctx context.Context, id string) error {
	return s.setUserStatus(ctx, id, store.User_status_banned, "user_banned")
}

func (s *AdminService) ReinstateUser(ctx context.Context, id string) error {
	return s.setUserStatus(ctx, id, store.User_status_active, "user_reinstated")
}

func (s *AdminService) SoftDeleteUser(ctx context.Context, id string) error {
	user, _ := s.st.GetUserByID(ctx, id)
	if err := s.setUserStatus(ctx, id, store.User_status_deleted, "user_deleted"); err != nil {
		return err
	}
	if s.mail != nil && user.Email != "" {
		s.mail.NotifyAccountDeactivated(user.Email, user.Full_name)
	}
	return nil
}

// DeactivateUser soft-deletes the account (legacy alias).
func (s *AdminService) DeactivateUser(ctx context.Context, id string) error {
	return s.SoftDeleteUser(ctx, id)
}

// ─── Talent management ────────────────────────────────────────────────────────

func (s *AdminService) syncUserStatusFromTalent(ctx context.Context, talent store.Talent, status string) {
	st := status
	active := status == string(store.Status_active)
	_ = s.st.UpdateUser(ctx, talent.User_id, store.UserPatch{Status: &st, Active: &active})
	if !active {
		_ = s.st.InvalidateAllUserSessions(ctx, talent.User_id)
	}
}

// resolveTalent accepts either a talent id or a user id (Users UI approve path).
func (s *AdminService) resolveTalent(ctx context.Context, id string) (store.Talent, error) {
	t, err := s.st.GetTalentByID(ctx, id)
	if err == nil {
		return t, nil
	}
	t, err = s.st.GetTalentByUserID(ctx, id)
	if err == nil {
		return t, nil
	}
	return store.Talent{}, err_talent_not_found
}

func (s *AdminService) ApproveTalent(ctx context.Context, id string, category store.Talent_category) error {
	before, err := s.resolveTalent(ctx, id)
	if err != nil {
		return err
	}
	status := string(store.Status_active)
	cat := string(category)
	if err := s.st.UpdateTalent(ctx, before.ID, store.TalentPatch{Status: &status, Category: &cat}); err != nil {
		return err
	}
	talent, err := s.st.GetTalentByID(ctx, before.ID)
	if err != nil {
		return err
	}
	if err := baseline.EnsureColdStart(ctx, s.st, talent, s.delta_lt); err != nil {
		return err
	}
	s.syncUserStatusFromTalent(ctx, talent, status)
	s.emitTalentUpdate(talent.ID, "approved", domain.TalentUpdateShard{Status: status, Category: cat})
	s.record(ctx, "talent_approved", "talent", talent.ID, before, talent)
	if s.mail != nil {
		s.notifyTalentUser(ctx, talent.ID, s.mail.NotifyTalentApproved)
	}
	return nil
}

func (s *AdminService) RejectTalent(ctx context.Context, id string) error {
	before, err := s.resolveTalent(ctx, id)
	if err != nil {
		return err
	}
	status := string(store.Status_rejected)
	if err := s.st.UpdateTalent(ctx, before.ID, store.TalentPatch{Status: &status}); err != nil {
		return err
	}
	talent, err := s.st.GetTalentByID(ctx, before.ID)
	if err != nil {
		return err
	}
	s.syncUserStatusFromTalent(ctx, talent, status)
	s.emitTalentUpdate(talent.ID, "rejected", domain.TalentUpdateShard{Status: status})
	s.record(ctx, "talent_rejected", "talent", talent.ID, before, talent)
	if s.mail != nil {
		s.notifyTalentUser(ctx, talent.ID, s.mail.NotifyTalentRejected)
	}
	return nil
}

func (s *AdminService) SuspendTalent(ctx context.Context, id string) error {
	before, err := s.resolveTalent(ctx, id)
	if err != nil {
		return err
	}
	status := string(store.Status_suspended)
	if err := s.st.UpdateTalent(ctx, before.ID, store.TalentPatch{Status: &status}); err != nil {
		return err
	}
	talent, err := s.st.GetTalentByID(ctx, before.ID)
	if err != nil {
		return err
	}
	s.syncUserStatusFromTalent(ctx, talent, status)
	s.emitTalentUpdate(talent.ID, "suspended", domain.TalentUpdateShard{Status: status})
	s.record(ctx, "talent_suspended", "talent", talent.ID, before, talent)
	if s.mail != nil {
		s.notifyTalentUser(ctx, talent.ID, s.mail.NotifyTalentSuspended)
	}
	return nil
}

func (s *AdminService) ReinstateTalent(ctx context.Context, id string) error {
	before, err := s.resolveTalent(ctx, id)
	if err != nil {
		return err
	}
	status := string(store.Status_active)
	if err := s.st.UpdateTalent(ctx, before.ID, store.TalentPatch{Status: &status}); err != nil {
		return err
	}
	talent, err := s.st.GetTalentByID(ctx, before.ID)
	if err != nil {
		return err
	}
	if err := baseline.EnsureColdStart(ctx, s.st, talent, s.delta_lt); err != nil {
		return err
	}
	s.syncUserStatusFromTalent(ctx, talent, status)
	s.emitTalentUpdate(talent.ID, "reinstated", domain.TalentUpdateShard{Status: status})
	s.record(ctx, "talent_reinstated", "talent", talent.ID, before, talent)
	if s.mail != nil {
		s.notifyTalentUser(ctx, talent.ID, s.mail.NotifyTalentReinstated)
	}
	return nil
}

func (s *AdminService) PatchTalent(ctx context.Context, id string, patch store.TalentPatch) error {
	before, err := s.resolveTalent(ctx, id)
	if err != nil {
		return err
	}
	if err := s.st.UpdateTalent(ctx, before.ID, patch); err != nil {
		return err
	}
	talent, err := s.st.GetTalentByID(ctx, before.ID)
	if err != nil {
		return err
	}
	if patch.Status != nil {
		s.syncUserStatusFromTalent(ctx, talent, *patch.Status)
	}
	s.emitTalentUpdate(talent.ID, "patched", domain.TalentUpdateShard{
		Status:   string(talent.Status),
		Category: string(talent.Category),
	})
	s.record(ctx, "talent_patched", "talent", talent.ID, before, talent)
	return nil
}

// ListTalents returns talents enriched with user display fields plus aggregate stats.
func (s *AdminService) ListTalents(ctx context.Context, filter store.TalentFilter) (TalentListResult, error) {
	talents, err := s.st.ListTalents(ctx, filter)
	if err != nil {
		return TalentListResult{}, err
	}
	items := make([]TalentListItem, 0, len(talents))
	var (
		activeCount int
		complianceSum float64
		earnings    float64
	)
	for _, t := range talents {
		item := TalentListItem{Talent: t}
		if u, err := s.st.GetUserByID(ctx, t.User_id); err == nil {
			item.Full_name = u.Full_name
			item.Email = u.Email
		}
		items = append(items, item)
		if t.Status == store.Status_active {
			activeCount++
		}
		complianceSum += t.Report_compliance
		if payouts, err := s.st.ListPayoutsByTalent(ctx, t.ID); err == nil {
			for _, p := range payouts {
				switch p.Status {
				case store.Payout_paid, store.Payout_approved:
					earnings += p.Final_payout
				}
			}
		}
	}
	avg := 0.0
	if len(talents) > 0 {
		avg = (complianceSum / float64(len(talents))) * 100
	}
	return TalentListResult{
		Stats: TalentStats{
			Human_earnings:  earnings,
			Humans:          len(talents),
			Active_humans:   activeCount,
			Avg_performance: avg,
		},
		Talents: items,
	}, nil
}

// ─── Viewer management ────────────────────────────────────────────────────────

func (s *AdminService) CreateViewer(ctx context.Context, v store.CampaignViewer) error {
	if err := s.st.CreateViewer(ctx, v); err != nil {
		return err
	}
	s.record(ctx, "viewer_created", "viewer", v.ID, nil, v)
	return nil
}

func (s *AdminService) ListViewers(ctx context.Context, campaign_id string) ([]store.CampaignViewer, error) {
	return s.st.ListViewersByCampaign(ctx, campaign_id)
}

func (s *AdminService) PatchViewer(ctx context.Context, id string, patch store.ViewerPatch) error {
	if err := s.st.UpdateViewer(ctx, id, patch); err != nil {
		return err
	}
	s.record(ctx, "viewer_patched", "viewer", id, nil, patch)
	return nil
}

func (s *AdminService) AddViewerPassword(ctx context.Context, p store.ViewerPassword) error {
	if err := s.st.AddViewerPassword(ctx, p); err != nil {
		return err
	}
	s.record(ctx, "viewer_password_added", "viewer", p.Viewer_id, nil, map[string]string{"label": p.Label})
	return nil
}

func (s *AdminService) ListViewerPasswords(ctx context.Context, viewer_id string) ([]store.ViewerPassword, error) {
	return s.st.ListViewerPasswords(ctx, viewer_id)
}

func (s *AdminService) DeactivateViewerPassword(ctx context.Context, id string) error {
	if err := s.st.DeactivateViewerPassword(ctx, id); err != nil {
		return err
	}
	s.record(ctx, "viewer_password_deactivated", "viewer_password", id, nil, nil)
	return nil
}

// ─── Audit log ────────────────────────────────────────────────────────────────

func (s *AdminService) ListAudit(ctx context.Context, entity_type, entity_id string) ([]store.AuditLog, error) {
	return s.st.ListAuditLog(ctx, entity_type, entity_id)
}

func (s *AdminService) ListAuditFiltered(ctx context.Context, f store.AuditFilter) ([]store.AuditLog, error) {
	return s.st.ListAuditLogFiltered(ctx, f)
}

func (s *AdminService) GetAudit(ctx context.Context, id string) (store.AuditLog, error) {
	return s.st.GetAuditLogByID(ctx, id)
}

func (s *AdminService) VerifyAudit(ctx context.Context, id string) (audit.VerifyResult, error) {
	if s.auditor == nil {
		return audit.VerifyResult{}, nil
	}
	return s.auditor.Verify(ctx, id)
}

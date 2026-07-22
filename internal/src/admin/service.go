package admin

import (
	"context"
	"log/slog"

	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/mail"
	"github.com/Ze-uus/talent-backend/internal/store"
	ws "github.com/Ze-uus/talent-backend/internal/websocket"
)

type AdminService struct {
	st               store.Store
	talent_update_ch chan<- domain.TalentUpdateEvent
	log              *slog.Logger
	mail             *mail.Service
}

func New(s store.Store, talent_update_ch chan<- domain.TalentUpdateEvent, log *slog.Logger, mailSvc *mail.Service) *AdminService {
	return &AdminService{st: s, talent_update_ch: talent_update_ch, log: log, mail: mailSvc}
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

// ─── User management ──────────────────────────────────────────────────────────

func (s *AdminService) ListUsers(ctx context.Context, filter store.UserFilter) ([]store.User, error) {
	return s.st.ListUsers(ctx, filter)
}

func (s *AdminService) GetUser(ctx context.Context, id string) (store.User, error) {
	return s.st.GetUserByID(ctx, id)
}

func (s *AdminService) PatchUser(ctx context.Context, id string, patch store.UserPatch) error {
	return s.st.UpdateUser(ctx, id, patch)
}

func (s *AdminService) DeactivateUser(ctx context.Context, id string) error {
	user, _ := s.st.GetUserByID(ctx, id)
	active := false
	if err := s.st.UpdateUser(ctx, id, store.UserPatch{Active: &active}); err != nil {
		return err
	}
	if s.mail != nil && user.Email != "" {
		s.mail.NotifyAccountDeactivated(user.Email, user.Full_name)
	}
	return nil
}

// ─── Talent management ────────────────────────────────────────────────────────

func (s *AdminService) ApproveTalent(ctx context.Context, id string, category store.Talent_category) error {
	status := string(store.Status_active)
	cat := string(category)
	if err := s.st.UpdateTalent(ctx, id, store.TalentPatch{Status: &status, Category: &cat}); err != nil {
		return err
	}
	s.emitTalentUpdate(id, "approved", domain.TalentUpdateShard{Status: status, Category: cat})
	if s.mail != nil {
		s.notifyTalentUser(ctx, id, s.mail.NotifyTalentApproved)
	}
	return nil
}

func (s *AdminService) RejectTalent(ctx context.Context, id string) error {
	status := string(store.Status_rejected)
	if err := s.st.UpdateTalent(ctx, id, store.TalentPatch{Status: &status}); err != nil {
		return err
	}
	s.emitTalentUpdate(id, "rejected", domain.TalentUpdateShard{Status: status})
	if s.mail != nil {
		s.notifyTalentUser(ctx, id, s.mail.NotifyTalentRejected)
	}
	return nil
}

func (s *AdminService) SuspendTalent(ctx context.Context, id string) error {
	status := string(store.Status_suspended)
	if err := s.st.UpdateTalent(ctx, id, store.TalentPatch{Status: &status}); err != nil {
		return err
	}
	s.emitTalentUpdate(id, "suspended", domain.TalentUpdateShard{Status: status})
	if s.mail != nil {
		s.notifyTalentUser(ctx, id, s.mail.NotifyTalentSuspended)
	}
	return nil
}

func (s *AdminService) ReinstateTalent(ctx context.Context, id string) error {
	status := string(store.Status_active)
	if err := s.st.UpdateTalent(ctx, id, store.TalentPatch{Status: &status}); err != nil {
		return err
	}
	s.emitTalentUpdate(id, "reinstated", domain.TalentUpdateShard{Status: status})
	if s.mail != nil {
		s.notifyTalentUser(ctx, id, s.mail.NotifyTalentReinstated)
	}
	return nil
}

func (s *AdminService) PatchTalent(ctx context.Context, id string, patch store.TalentPatch) error {
	if err := s.st.UpdateTalent(ctx, id, patch); err != nil {
		return err
	}
	talent, err := s.st.GetTalentByID(ctx, id)
	if err != nil {
		return err
	}
	s.emitTalentUpdate(id, "patched", domain.TalentUpdateShard{
		Status:   string(talent.Status),
		Category: string(talent.Category),
	})
	return nil
}

// ─── Viewer management ────────────────────────────────────────────────────────

func (s *AdminService) CreateViewer(ctx context.Context, v store.CampaignViewer) error {
	return s.st.CreateViewer(ctx, v)
}

func (s *AdminService) ListViewers(ctx context.Context, campaign_id string) ([]store.CampaignViewer, error) {
	return s.st.ListViewersByCampaign(ctx, campaign_id)
}

func (s *AdminService) PatchViewer(ctx context.Context, id string, patch store.ViewerPatch) error {
	return s.st.UpdateViewer(ctx, id, patch)
}

func (s *AdminService) AddViewerPassword(ctx context.Context, p store.ViewerPassword) error {
	return s.st.AddViewerPassword(ctx, p)
}

func (s *AdminService) ListViewerPasswords(ctx context.Context, viewer_id string) ([]store.ViewerPassword, error) {
	return s.st.ListViewerPasswords(ctx, viewer_id)
}

func (s *AdminService) DeactivateViewerPassword(ctx context.Context, id string) error {
	return s.st.DeactivateViewerPassword(ctx, id)
}

// ─── Audit log ────────────────────────────────────────────────────────────────

func (s *AdminService) ListAudit(ctx context.Context, entity_type, entity_id string) ([]store.AuditLog, error) {
	return s.st.ListAuditLog(ctx, entity_type, entity_id)
}

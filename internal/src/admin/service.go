package admin

import (
	"context"

	"github.com/Ze-uus/talent-backend/internal/store"
)

type AdminService struct {
	st store.Store
}

func New(s store.Store) *AdminService { return &AdminService{st: s} }

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
	active := false
	return s.st.UpdateUser(ctx, id, store.UserPatch{Active: &active})
}

// ─── Talent management ────────────────────────────────────────────────────────

func (s *AdminService) ApproveTalent(ctx context.Context, id string, category store.Talent_category) error {
	status := string(store.Status_active)
	cat := string(category)
	return s.st.UpdateTalent(ctx, id, store.TalentPatch{Status: &status, Category: &cat})
}

func (s *AdminService) RejectTalent(ctx context.Context, id string) error {
	status := string(store.Status_rejected)
	return s.st.UpdateTalent(ctx, id, store.TalentPatch{Status: &status})
}

func (s *AdminService) SuspendTalent(ctx context.Context, id string) error {
	status := string(store.Status_suspended)
	return s.st.UpdateTalent(ctx, id, store.TalentPatch{Status: &status})
}

func (s *AdminService) ReinstateTalent(ctx context.Context, id string) error {
	status := string(store.Status_active)
	return s.st.UpdateTalent(ctx, id, store.TalentPatch{Status: &status})
}

func (s *AdminService) PatchTalent(ctx context.Context, id string, patch store.TalentPatch) error {
	return s.st.UpdateTalent(ctx, id, patch)
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

package talent

import (
	"context"
	"errors"

	"github.com/Ze-uus/talent-backend/internal/audit"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type TalentService struct {
	st      store.Store
	auditor *audit.Recorder
}

func New(s store.Store, auditor *audit.Recorder) *TalentService {
	return &TalentService{st: s, auditor: auditor}
}

func (s *TalentService) GetProfile(ctx context.Context, user_id string) (store.Talent, error) {
	return s.st.GetTalentByUserID(ctx, user_id)
}

func (s *TalentService) PatchProfile(ctx context.Context, talent_id string, patch store.TalentPatch) error {
	before, _ := s.st.GetTalentByID(ctx, talent_id)
	if err := s.st.UpdateTalent(ctx, talent_id, patch); err != nil {
		return err
	}
	after, _ := s.st.GetTalentByID(ctx, talent_id)
	if s.auditor != nil {
		_ = s.auditor.Record(ctx, audit.Entry{
			Action:      "talent_profile_patched",
			Entity_type: "talent",
			Entity_id:   talent_id,
			Before:      before,
			After:       after,
		})
	}
	return nil
}

func (s *TalentService) ListMyCycles(ctx context.Context, talent_id string) ([]store.TalentAssignment, error) {
	return s.st.ListAssignmentsByTalent(ctx, talent_id)
}

// GetMyCycle returns the assignment and the talent's tracking link for that cycle.
func (s *TalentService) GetMyCycle(ctx context.Context, talent_id, cycle_id string) (store.TalentAssignment, store.TrackingLink, error) {
	assignment, err := s.st.GetAssignment(ctx, talent_id, cycle_id)
	if err != nil {
		return store.TalentAssignment{}, store.TrackingLink{}, err
	}
	links, err := s.st.ListTrackingLinksByCycle(ctx, cycle_id)
	if err != nil {
		return assignment, store.TrackingLink{}, err
	}
	for _, l := range links {
		if l.Talent_id == talent_id && l.Active {
			return assignment, l, nil
		}
	}
	return assignment, store.TrackingLink{}, nil
}

func (s *TalentService) GetCycleStats(ctx context.Context, talent_id, cycle_id string) (map[string]any, error) {
	my_conversions, err := s.st.GetTalentConversions(ctx, talent_id, cycle_id)
	if err != nil {
		return nil, err
	}
	total, err := s.st.GetTotalConversions(ctx, cycle_id)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"my_conversions": my_conversions,
		"cycle_total":    total,
	}, nil
}

func (s *TalentService) GetHistory(ctx context.Context, talent_id string) ([]store.PayoutRecord, error) {
	return s.st.ListPayoutsByTalent(ctx, talent_id)
}

// RequestExpansion writes an audit log entry for a budget expansion request.
// No budget mutation happens here — it is reviewed by admin.
func (s *TalentService) RequestExpansion(ctx context.Context, talent_id, cycle_id, actor_id string) error {
	assignment, err := s.st.GetAssignment(ctx, talent_id, cycle_id)
	if err != nil {
		return err
	}
	if assignment.Status != "active" {
		return errors.New("assignment_not_active")
	}
	if s.auditor != nil {
		return s.auditor.Record(ctx, audit.Entry{
			Action:      "budget_expansion_requested",
			Entity_type: "assignment",
			Entity_id:   talent_id + ":" + cycle_id,
		})
	}
	return s.st.WriteAuditLog(ctx, store.AuditLog{
		Entity_type: "assignment",
		Entity_id:   talent_id + ":" + cycle_id,
		Action_type: "budget_expansion_requested",
		Actor_id:    actor_id,
	})
}

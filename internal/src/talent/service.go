package talent

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/Ze-uus/talent-backend/internal/audit"
	"github.com/Ze-uus/talent-backend/internal/store"
)

var ErrCycleNotAssigned = errors.New("cycle_not_assigned")

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

func (s *TalentService) ListMyCycles(ctx context.Context, talent_id string) ([]CycleView, error) {
	assignments, err := s.st.ListAssignmentsByTalent(ctx, talent_id)
	if err != nil {
		return nil, err
	}
	views := make([]CycleView, 0, len(assignments))
	for _, assignment := range assignments {
		view, err := s.buildCycleView(ctx, assignment)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

// GetMyCycle returns a talent-safe enriched view and verifies assignment ownership.
func (s *TalentService) GetMyCycle(ctx context.Context, talent_id, cycle_id string) (CycleView, error) {
	assignment, err := s.getAssignment(ctx, talent_id, cycle_id)
	if err != nil {
		return CycleView{}, err
	}
	return s.buildCycleView(ctx, assignment)
}

func (s *TalentService) buildCycleView(ctx context.Context, assignment store.TalentAssignment) (CycleView, error) {
	cycle, err := s.st.GetCycleByID(ctx, assignment.Cycle_id)
	if err != nil {
		return CycleView{}, err
	}
	campaign, err := s.st.GetCampaignByID(ctx, assignment.Campaign_id)
	if err != nil {
		return CycleView{}, err
	}
	brand, err := s.st.GetBrandByID(ctx, campaign.Brand_id)
	if err != nil {
		return CycleView{}, err
	}
	managers, err := s.st.ListManagersByCampaignID(ctx, campaign.ID)
	if err != nil {
		return CycleView{}, err
	}
	links, err := s.st.ListTrackingLinksByCycle(ctx, cycle.ID)
	if err != nil {
		return CycleView{}, err
	}

	managerViews := make([]ManagerView, 0, len(managers))
	for _, manager := range managers {
		managerViews = append(managerViews, ManagerView{
			ID: manager.ID, FullName: manager.Full_name, Email: manager.Email,
		})
	}

	var trackingLink *TrackingLinkView
	for _, l := range links {
		if l.Talent_id == assignment.Talent_id && l.Active {
			trackingLink = &TrackingLinkView{
				Token: l.Token, Active: l.Active, EventEndpoint: "/t/" + l.Token,
			}
			break
		}
	}

	stats, err := s.loadCycleStats(ctx, assignment.Talent_id, cycle.ID)
	if err != nil {
		return CycleView{}, err
	}
	kpbConfig := cycle.KPB_config
	if kpbConfig == nil {
		kpbConfig = []store.KPBDefinition{}
	}

	return CycleView{
		Assignment: AssignmentView{
			TalentID: assignment.Talent_id, CampaignID: assignment.Campaign_id,
			CycleID: assignment.Cycle_id, SlotID: assignment.Slot_id,
			RoleLabel: assignment.Role_label, Status: assignment.Status,
			Source: assignment.Assignment_source, EffectiveTier: assignment.Effective_tier,
			AssignedAt: assignment.Assigned_at,
		},
		Campaign: CampaignView{
			ID: campaign.ID, HumanID: campaign.Human_id, Name: campaign.Name,
			Status: campaign.Status, CampaignType: campaign.Campaign_type,
			Audience: campaign.Audience, TargetCPA: campaign.Target_cpa,
			MaxCPA: campaign.Max_cpa, CreatorsAllowed: campaign.Creators_allowed,
		},
		Brand: BrandView{
			ID: brand.ID, Name: brand.Name, Industry: brand.Industry,
			Description: brand.Description, Website: brand.Website, LogoURL: brand.Logo_url,
		},
		Cycle: CycleSummary{
			ID: cycle.ID, HumanID: cycle.Human_id, CampaignID: cycle.Campaign_id,
			CycleNumber: cycle.Cycle_number, Status: cycle.Status,
			CycleObjective: cycle.Cycle_objective, CampaignType: cycle.Campaign_type,
			KPBConfig: kpbConfig, Content: store.EffectiveContent(campaign, cycle),
			StartDate: cycle.Start_date, EndDate: cycle.End_date,
		},
		CampaignManagers: managerViews,
		TrackingLink:     trackingLink,
		Stats:            stats,
	}, nil
}

func (s *TalentService) GetCycleStats(ctx context.Context, talent_id, cycle_id string) (CycleStatsView, error) {
	if _, err := s.getAssignment(ctx, talent_id, cycle_id); err != nil {
		return CycleStatsView{}, err
	}
	return s.loadCycleStats(ctx, talent_id, cycle_id)
}

func (s *TalentService) loadCycleStats(ctx context.Context, talent_id, cycle_id string) (CycleStatsView, error) {
	my_conversions, err := s.st.GetTalentConversions(ctx, talent_id, cycle_id)
	if err != nil {
		return CycleStatsView{}, err
	}
	total, err := s.st.GetTotalConversions(ctx, cycle_id)
	if err != nil {
		return CycleStatsView{}, err
	}
	return CycleStatsView{MyConversions: my_conversions, CycleTotal: total}, nil
}

func (s *TalentService) getAssignment(ctx context.Context, talent_id, cycle_id string) (store.TalentAssignment, error) {
	assignment, err := s.st.GetAssignment(ctx, talent_id, cycle_id)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.TalentAssignment{}, ErrCycleNotAssigned
	}
	return assignment, err
}

func (s *TalentService) GetHistory(ctx context.Context, talent_id string) ([]store.PayoutRecord, error) {
	return s.st.ListPayoutsByTalent(ctx, talent_id)
}

// RequestExpansion writes an audit log entry for a budget expansion request.
// No budget mutation happens here — it is reviewed by admin.
func (s *TalentService) RequestExpansion(ctx context.Context, talent_id, cycle_id, actor_id string) error {
	assignment, err := s.getAssignment(ctx, talent_id, cycle_id)
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

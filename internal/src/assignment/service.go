package assignment

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"time"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/domain"
	"github.com/Ze-uus/talent-backend/internal/mail"
	"github.com/Ze-uus/talent-backend/internal/store"
	ws "github.com/Ze-uus/talent-backend/internal/websocket"
)

type AssignmentService struct {
	st               store.Store
	talent_update_ch chan<- domain.TalentUpdateEvent
	log              *slog.Logger
	mail             *mail.Service
}

func New(s store.Store, talent_update_ch chan<- domain.TalentUpdateEvent, log *slog.Logger, mailSvc *mail.Service) *AssignmentService {
	return &AssignmentService{st: s, talent_update_ch: talent_update_ch, log: log, mail: mailSvc}
}

type SolverOutput struct {
	Cycle_id     string                         `json:"cycle_id"`
	Assignments  []algo.Assignment              `json:"assignments"`
	Unassigned   []string                       `json:"unassigned"`
	Total_cost   float64                        `json:"total_cost"`
	Match_scores map[string]algo.MatchScoreResult `json:"match_scores"`
	Computed_at  time.Time                      `json:"computed_at"`
}

type ConfirmedAssignment struct {
	Talent_id      string          `json:"talent_id"`
	Slot_id        string          `json:"slot_id"`
	Role_label     string          `json:"role_label"`
	PDC_mode       store.Pdc_mode  `json:"pdc_mode"`
	PDC_value      float64         `json:"pdc_value"`
	Effective_tier int             `json:"effective_tier"`
	Is_pinned      bool            `json:"is_pinned"`
}

// RunSolver runs qualifier + match score + Hungarian for a cycle.
func (s *AssignmentService) RunSolver(ctx context.Context, cycle_id string) (SolverOutput, error) {
	cycle, err := s.st.GetCycleByID(ctx, cycle_id)
	if err != nil {
		return SolverOutput{}, err
	}
	campaign, err := s.st.GetCampaignByID(ctx, cycle.Campaign_id)
	if err != nil {
		return SolverOutput{}, err
	}
	slots, err := s.st.ListSlotsByCycle(ctx, cycle_id)
	if err != nil {
		return SolverOutput{}, err
	}
	talents, err := s.st.ListAllTalents(ctx)
	if err != nil {
		return SolverOutput{}, err
	}

	algo_slots := make([]algo.BudgetSlot, len(slots))
	for i, sl := range slots {
		algo_slots[i] = algo.BudgetSlot{
			ID:          sl.ID,
			Tier_value:  sl.Tier_value,
			Target_cost: campaign.Target_cpa,
			Max_cost:    campaign.Max_cpa,
		}
	}

	all_tiers := algo.ValidTiers(algo.DefaultConfig())
	match_scores := make(map[string]algo.MatchScoreResult)
	talent_profiles := make([]algo.TalentProfile, 0, len(talents))

	for _, t := range talents {
		if t.Status != store.Status_active {
			continue
		}
		b, err := s.st.GetTalentBaseline(ctx, t.ID)
		if err != nil {
			continue
		}

		dm := computeDemographicMatch(t, campaign)
		gp := computeGeographicProximity(t, campaign)
		oh := computeObjectiveHistory(t, campaign.Campaign_type)

		ms := algo.ComputeMatchScore(algo.MatchDimensions{DM: dm, GP: gp, OH: oh})
		match_scores[t.ID] = ms

		pdc, err := algo.Cycle1PDC(algo.TalentBaseline{Lambda_lt: b.Lambda_lt}, cycle.Cycle_budget)
		if err != nil {
			continue
		}
		talent_profiles = append(talent_profiles, algo.TalentProfile{
			ID:           t.ID,
			PDC:          pdc,
			Max_tier:     t.Max_tier,
			Cycle_length: cycleDays(cycle),
		})
	}

	result, err := algo.Solve(algo.AssignmentInput{
		Talents:     talent_profiles,
		Slots:       algo_slots,
		All_tiers:   all_tiers,
		MatchScores: match_scores,
		Alpha:       algo.DefaultAlpha,
	})
	if err != nil {
		return SolverOutput{}, err
	}

	return SolverOutput{
		Cycle_id:     cycle_id,
		Assignments:  result.Assignments,
		Unassigned:   result.Unassigned,
		Total_cost:   result.Total_cost,
		Match_scores: match_scores,
		Computed_at:  time.Now().UTC(),
	}, nil
}

// ConfirmAssignments persists solver output with full allocation audit fields.
func (s *AssignmentService) ConfirmAssignments(
	ctx context.Context,
	cycle_id, campaign_id, actor_id string,
	confirmed []ConfirmedAssignment,
	output SolverOutput,
	is_override bool,
) error {
	for _, ca := range confirmed {
		ms := output.Match_scores[ca.Talent_id]
		source := store.Source_algorithm
		pinned_tier := 0
		if ca.Is_pinned {
			source = store.Source_pinned
			pinned_tier = ca.Effective_tier
		}
		if is_override {
			source = store.Source_manual
		}

		a := store.TalentAssignment{
			Talent_id:         ca.Talent_id,
			Campaign_id:       campaign_id,
			Cycle_id:          cycle_id,
			Slot_id:           ca.Slot_id,
			Role_label:        ca.Role_label,
			Status:            "active",
			Assignment_source: source,
			PDC_mode:          ca.PDC_mode,
			PDC_value:         ca.PDC_value,
			Match_score:       ms.MS_t,
			Match_dm:          ms.Dimensions.DM,
			Match_gp:          ms.Dimensions.GP,
			Match_oh:          ms.Dimensions.OH,
			Effective_tier:    ca.Effective_tier,
			Pinned_tier:       pinned_tier,
			Assigned_at:       time.Now().UTC(),
		}
		if err := s.st.CreateAssignment(ctx, a); err != nil {
			return err
		}
		if err := s.st.AssignSlot(ctx, ca.Slot_id, ca.Talent_id); err != nil {
			return err
		}
		link := store.TrackingLink{
			Campaign_id: campaign_id,
			Cycle_id:    cycle_id,
			Talent_id:   ca.Talent_id,
			Token:       generateLinkToken(),
			Active:      true,
			Created_at:  time.Now().UTC(),
		}
		if err := s.st.CreateTrackingLink(ctx, link); err != nil {
			return err
		}
		ws.TryPush(s.talent_update_ch, domain.TalentUpdateEvent{
			Talent_id:   ca.Talent_id,
			Cycle_id:    cycle_id,
			Update_type: "assigned",
			Shard: domain.TalentUpdateShard{
				Status:         "active",
				Cycle_id:       cycle_id,
				Effective_tier: ca.Effective_tier,
				Slot_id:        ca.Slot_id,
				Tracking_token: link.Token,
			},
		}, s.log, "talent_update_ch full, event dropped", "talent_id", ca.Talent_id, "cycle_id", cycle_id)

		if s.mail != nil {
			s.notifyTalentAssigned(ctx, ca.Talent_id, campaign_id, cycle_id, ca.Effective_tier)
		}
	}
	if is_override {
		_ = s.st.WriteAuditLog(ctx, store.AuditLog{
			Actor_id:    actor_id,
			Action_type: "assignment_override",
			Entity_type: "cycle",
			Entity_id:   cycle_id,
		})
	}
	return nil
}

func (s *AssignmentService) notifyTalentAssigned(ctx context.Context, talent_id, campaign_id, cycle_id string, tier int) {
	talent, err := s.st.GetTalentByID(ctx, talent_id)
	if err != nil {
		return
	}
	user, err := s.st.GetUserByID(ctx, talent.User_id)
	if err != nil || user.Email == "" {
		return
	}
	campaign, err := s.st.GetCampaignByID(ctx, campaign_id)
	if err != nil {
		return
	}
	cycle, err := s.st.GetCycleByID(ctx, cycle_id)
	if err != nil {
		return
	}
	s.mail.NotifyTalentAssigned(user.Email, user.Full_name, campaign.Name, cycle.Cycle_number, tier)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func computeDemographicMatch(t store.Talent, c store.Campaign) float64 {
	if t.Category != "" && c.Audience != "" {
		if string(t.Category) == c.Audience {
			return 1.0
		}
		return 0.3
	}
	return 0.5
}

func computeGeographicProximity(_ store.Talent, _ store.Campaign) float64 {
	return 0.5
}

func computeObjectiveHistory(_ store.Talent, _ store.Campaign_type) float64 {
	return 0.5
}

func campaignTypeToAlgoPipeline(t store.Campaign_type) algo.PipelineType {
	if t == store.Type_lead_validation {
		return algo.Pipeline_lead
	}
	return algo.Pipeline_traffic
}

func cycleDays(c store.Cycle) int {
	if c.End_date.IsZero() || c.Start_date.IsZero() {
		return 30
	}
	d := int(c.End_date.Sub(c.Start_date).Hours() / 24)
	if d <= 0 {
		return 30
	}
	return d
}

func generateLinkToken() string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:24]
}

// Ensure helper is used.
var _ = campaignTypeToAlgoPipeline

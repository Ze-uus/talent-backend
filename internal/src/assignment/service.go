package assignment

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log/slog"
	"time"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/allocation"
	"github.com/Ze-uus/talent-backend/internal/audit"
	"github.com/Ze-uus/talent-backend/internal/baseline"
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
	auditor          *audit.Recorder
	delta_lt         float64
}

func New(s store.Store, talent_update_ch chan<- domain.TalentUpdateEvent, log *slog.Logger, mailSvc *mail.Service, auditor *audit.Recorder, delta_lt float64) *AssignmentService {
	if delta_lt <= 0 || delta_lt >= 1 {
		delta_lt = 0.97
	}
	return &AssignmentService{st: s, talent_update_ch: talent_update_ch, log: log, mail: mailSvc, auditor: auditor, delta_lt: delta_lt}
}

type SolverOutput struct {
	Cycle_id           string                           `json:"cycle_id"`
	Assignments        []algo.Assignment                `json:"assignments"`
	Unassigned         []string                         `json:"unassigned"`
	Unassigned_details []UnassignedDetail               `json:"unassigned_details"`
	Total_cost         float64                          `json:"total_cost"`
	Match_scores       map[string]algo.MatchScoreResult `json:"match_scores"`
	Computed_at        time.Time                        `json:"computed_at"`
}

type UnassignedDetail struct {
	Talent_id string `json:"talent_id"`
	Reason    string `json:"reason"`
}

type ConfirmedAssignment struct {
	Talent_id      string         `json:"talent_id"`
	Slot_id        string         `json:"slot_id"`
	Role_label     string         `json:"role_label"`
	PDC_mode       store.Pdc_mode `json:"pdc_mode"`
	PDC_value      float64        `json:"pdc_value"`
	Effective_tier int            `json:"effective_tier"`
	Is_pinned      bool           `json:"is_pinned"`
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
	if !hasUsableSlot(slots, campaign.Target_cpa) {
		var regenerated []store.BudgetSlot
		regenerated, err = allocation.BuildBudgetSlots(cycle, campaign, talents)
		if err != nil {
			return SolverOutput{}, err
		}
		if len(regenerated) == 0 {
			return SolverOutput{}, errors.New("no eligible budget slots")
		}
		if err := s.st.CreateBudgetSlots(ctx, regenerated); err != nil {
			return SolverOutput{}, err
		}
		slots = append(slots, regenerated...)
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
			if bootErr := baseline.EnsureColdStart(ctx, s.st, t, s.delta_lt); bootErr != nil {
				if s.log != nil {
					s.log.Warn("solver: cold-start baseline failed", "talent_id", t.ID, "err", bootErr)
				}
				continue
			}
			b, err = s.st.GetTalentBaseline(ctx, t.ID)
			if err != nil {
				continue
			}
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
	if result.Assignments == nil {
		result.Assignments = []algo.Assignment{}
	}
	if result.Unassigned == nil {
		result.Unassigned = []string{}
	}

	return SolverOutput{
		Cycle_id:           cycle_id,
		Assignments:        result.Assignments,
		Unassigned:         result.Unassigned,
		Unassigned_details: explainUnassigned(result.Unassigned, talent_profiles, algo_slots, all_tiers, campaign.Target_cpa),
		Total_cost:         result.Total_cost,
		Match_scores:       match_scores,
		Computed_at:        time.Now().UTC(),
	}, nil
}

func hasUsableSlot(slots []store.BudgetSlot, target_cpa float64) bool {
	for _, slot := range slots {
		if float64(slot.Tier_value) >= target_cpa && !slot.Allocated {
			return true
		}
	}
	return false
}

func explainUnassigned(
	ids []string,
	talents []algo.TalentProfile,
	slots []algo.BudgetSlot,
	all_tiers []int,
	target_cpa float64,
) []UnassignedDetail {
	details := make([]UnassignedDetail, 0, len(ids))
	for _, id := range ids {
		reason := "no_available_slot"
		for _, talent := range talents {
			if talent.ID != id {
				continue
			}
			if talent.Max_tier > 0 && float64(talent.Max_tier) < target_cpa {
				reason = "max_tier_below_target_cpa"
				break
			}
			for _, slot := range slots {
				if float64(slot.Tier_value) < target_cpa {
					reason = "slot_below_target_cpa"
					continue
				}
				q, err := algo.QualifyTalentForSlot(talent, slot, all_tiers)
				if err == nil && !q.Qualified {
					reason = q.Reason
				}
			}
			break
		}
		details = append(details, UnassignedDetail{Talent_id: id, Reason: reason})
	}
	return details
}

func (s *AssignmentService) ListCycleAssignments(ctx context.Context, campaignID, cycleID string) ([]HumanAssignmentView, error) {
	campaign, err := s.st.GetCampaignByID(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	cycle, err := s.st.GetCycleByID(ctx, cycleID)
	if err != nil {
		return nil, err
	}
	if cycle.Campaign_id != campaign.ID {
		return nil, errors.New("cycle_campaign_mismatch")
	}
	assignments, err := s.st.ListAssignedTalents(ctx, cycleID)
	if err != nil {
		return nil, err
	}
	links, err := s.st.ListTrackingLinksByCycle(ctx, cycleID)
	if err != nil {
		return nil, err
	}
	payouts, err := s.st.ListPayoutsByCycle(ctx, cycleID)
	if err != nil {
		return nil, err
	}

	linksByTalent := make(map[string]store.TrackingLink, len(links))
	for _, link := range links {
		existing, exists := linksByTalent[link.Talent_id]
		if !exists || (!existing.Active && link.Active) {
			linksByTalent[link.Talent_id] = link
		}
	}
	payoutsByTalent := make(map[string]store.PayoutRecord, len(payouts))
	for _, payout := range payouts {
		payoutsByTalent[payout.Talent_id] = payout
	}

	views := make([]HumanAssignmentView, 0, len(assignments))
	for _, assignment := range assignments {
		talent, err := s.st.GetTalentByID(ctx, assignment.Talent_id)
		if err != nil {
			return nil, err
		}
		user, err := s.st.GetUserByID(ctx, talent.User_id)
		if err != nil {
			return nil, err
		}
		conversions, err := s.st.GetTalentConversions(ctx, talent.ID, cycleID)
		if err != nil {
			return nil, err
		}
		skills := talent.Skills
		if skills == nil {
			skills = []string{}
		}
		var trackingLink *AssignmentTrackingLink
		if link, ok := linksByTalent[talent.ID]; ok {
			trackingLink = &AssignmentTrackingLink{
				Token: link.Token, URL: "/t/" + link.Token, Active: link.Active,
			}
		}
		payout, hasPayout := payoutsByTalent[talent.ID]
		paymentStatus := store.Payout_pending
		totalEarned := 0.0
		allocatedBudget := float64(assignment.Effective_tier)
		if hasPayout {
			paymentStatus = payout.Status
			totalEarned = payout.Final_payout
			if payout.Allocated_budget > 0 {
				allocatedBudget = payout.Allocated_budget
			}
		}
		views = append(views, HumanAssignmentView{
			TalentID: talent.ID,
			Human: AssignmentHuman{
				UserID: user.ID, FullName: user.Full_name, Email: user.Email,
				PhoneNumber: user.Phone_number, AvatarURL: user.Avatar_url,
				Category: talent.Category, Status: talent.Status, Skills: skills,
				RatePerDay: talent.Rate_per_day, ReportCompliance: talent.Report_compliance,
			},
			Assignment: AssignmentSummary{
				RoleLabel: assignment.Role_label, Status: assignment.Status,
				AssignmentSource: assignment.Assignment_source, PDCMode: assignment.PDC_mode,
				PDCValue: assignment.PDC_value, EffectiveTier: assignment.Effective_tier,
				AssignedAt: assignment.Assigned_at,
			},
			TrackingLink: trackingLink,
			Campaign: AssignmentCampaign{
				ID: campaign.ID, HumanID: campaign.Human_id, Name: campaign.Name,
				Status: campaign.Status, Type: campaign.Campaign_type,
				TotalBudget: campaign.Total_budget, TargetCPA: campaign.Target_cpa, MaxCPA: campaign.Max_cpa,
			},
			Cycle: AssignmentCycle{
				ID: cycle.ID, HumanID: cycle.Human_id, CycleNumber: cycle.Cycle_number,
				Status: cycle.Status, CycleLength: campaign.Cycle_length,
				CycleBudget: cycle.Cycle_budget, AllocatedBudget: allocatedBudget,
				StartDate: cycle.Start_date, EndDate: cycle.End_date,
			},
			Performance: AssignmentPerformance{
				AOC: conversions, Conversions: conversions, MatchScore: assignment.Match_score,
				MatchDM: assignment.Match_dm, MatchGP: assignment.Match_gp, MatchOH: assignment.Match_oh,
			},
			Earnings: AssignmentEarnings{TotalEarned: totalEarned, PaymentStatus: paymentStatus},
		})
	}
	return views, nil
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
		if s.auditor != nil {
			_ = s.auditor.Record(ctx, audit.Entry{
				Action:      "assignment_override",
				Entity_type: "cycle",
				Entity_id:   cycle_id,
			})
		} else {
			_ = s.st.WriteAuditLog(ctx, store.AuditLog{
				Actor_id:    actor_id,
				Action_type: "assignment_override",
				Entity_type: "cycle",
				Entity_id:   cycle_id,
			})
		}
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

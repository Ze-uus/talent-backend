package payout

import (
	"context"
	"time"

	"github.com/Ze-uus/talent-backend/internal/algo"
	"github.com/Ze-uus/talent-backend/internal/jsonutil"
	"github.com/Ze-uus/talent-backend/internal/store"
)

type PayoutService struct {
	st store.Store
}

func New(s store.Store) *PayoutService { return &PayoutService{st: s} }

// ComputeAndStoreCyclePayout runs the full Story 17 algorithm for a cycle.
// All payout_records are created in "report_pending" state.
func (s *PayoutService) ComputeAndStoreCyclePayout(ctx context.Context, cycle_id string) error {
	cycle, err := s.st.GetCycleByID(ctx, cycle_id)
	if err != nil {
		return err
	}
	assignments, err := s.st.ListAssignedTalents(ctx, cycle_id)
	if err != nil {
		return err
	}

	pool := algo.CyclePoolInput{
		Available_pool: cycle.Remaining_budget,
	}

	var talent_inputs []algo.TalentPayoutInput
	for _, a := range assignments {
		conversions, _ := s.st.GetTalentConversions(ctx, a.Talent_id, cycle_id)

		ti := algo.TalentPayoutInput{
			Talent_id:        a.Talent_id,
			Pipeline:         campaignTypeToAlgoPipeline(cycle.Campaign_type),
			Allocated_budget: float64(a.Effective_tier),
			Commission_rate:  0.30,
			Status:           a.Status,
		}

		switch cycle.Campaign_type {
		case store.Type_direct_traffic:
			ti.Destinations = []algo.DestinationParams{{
				Conversions:   conversions,
				Anchor_cost:   cycleCostFromCycle(ctx, s, cycle),
				Cost_variance: 0,
			}}
		case store.Type_lead_validation:
			ti.Valid_leads = conversions
			ti.Anchor_lead_cost = cycleCostFromCycle(ctx, s, cycle)
			ti.Cost_variance_l = 0
			ti.KPB_bonuses = buildKPBInputs(cycle.KPB_config, a.Talent_id, cycle_id, s.st, ctx)
		}

		talent_inputs = append(talent_inputs, ti)
	}

	output := algo.ComputeCyclePayout(talent_inputs, pool)

	for _, r := range output.Results {
		record := store.PayoutRecord{
			Talent_id:         r.Talent_id,
			Cycle_id:          cycle_id,
			Campaign_id:       cycle.Campaign_id,
			Pipeline_type:     campaignTypeFromAlgoPipeline(r.Pipeline),
			Status:            store.Payout_report_pending,
			Gross_base:        r.Gross_base,
			KPB_total:         r.KPB_total,
			Gross_total:       r.Gross_total,
			Cost_per_unit:     r.Cost_per_unit,
			Cap_applied:       r.Cap_applied,
			Cap_exceeded:      r.Cap_exceeded,
			Excess_forfeited:  r.Excess_forfeited,
			Commission_rate:   0.30,
			Commission_amount: r.Commission_amount,
			E_net:             r.E_net,
			Scale_factor:      output.Scale_factor,
			Final_payout:      r.Final_payout,
			KPB_pool_source:   r.KPB_pool_source,
			Fallback_flagged:  r.Fallback_flagged,
			Created_at:        time.Now().UTC(),
			Updated_at:        time.Now().UTC(),
		}
		if err := s.st.CreatePayoutRecord(ctx, record); err != nil {
			return err
		}
	}

	return nil
}

// ApprovePayout moves a payout record from pending → approved.
func (s *PayoutService) ApprovePayout(ctx context.Context, talent_id, cycle_id, actor_id string) error {
	record, err := s.st.GetPayoutRecord(ctx, talent_id, cycle_id)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	approved := store.Payout_approved
	return s.st.UpdatePayoutRecord(ctx, record.ID, store.PayoutPatch{
		Status:      &approved,
		Approved_by: &actor_id,
		Approved_at: &now,
	})
}

// FlagPayout marks a payout for manual review.
func (s *PayoutService) FlagPayout(ctx context.Context, talent_id, cycle_id, reason, actor_id string) error {
	_ = s.st.WriteAuditLog(ctx, store.AuditLog{
		Actor_id:    actor_id,
		Action_type: "payout_flagged",
		Entity_type: "payout_record",
		Entity_id:   talent_id + ":" + cycle_id,
		After_state: jsonutil.Marshal(map[string]string{"reason": reason}),
	})
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func campaignTypeToAlgoPipeline(t store.Campaign_type) algo.PipelineType {
	if t == store.Type_lead_validation {
		return algo.Pipeline_lead
	}
	return algo.Pipeline_traffic
}

func campaignTypeFromAlgoPipeline(p algo.PipelineType) store.Campaign_type {
	if p == algo.Pipeline_lead {
		return store.Type_lead_validation
	}
	return store.Type_direct_traffic
}

func cycleCostFromCycle(ctx context.Context, s *PayoutService, cycle store.Cycle) float64 {
	campaign, err := s.st.GetCampaignByID(ctx, cycle.Campaign_id)
	if err != nil {
		return 0
	}
	return campaign.Target_cpa
}

func buildKPBInputs(kpb_config []store.KPBDefinition, talent_id, cycle_id string, st store.Store, ctx context.Context) []algo.KPBBonusParams {
	var params []algo.KPBBonusParams
	for _, k := range kpb_config {
		// Conversions logged with event_type matching KPB label are the trigger count
		count, err := st.GetTalentConversions(ctx, talent_id, cycle_id)
		if err != nil {
			continue
		}
		params = append(params, algo.KPBBonusParams{
			Option_id:     k.Label,
			Bonus_amount:  k.Cost,
			Trigger_count: int(count),
		})
	}
	return params
}

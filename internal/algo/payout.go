package algo

import "math"

// ─── Money precision ──────────────────────────────────────────────────────────

// MoneyRound rounds a float64 NGN value to the nearest kobo (0.01 NGN).
// This is the only rounding function permitted on monetary values (Story 17).
func MoneyRound(x float64) float64 {
	return math.Round(x*100) / 100
}

// ─── Pipeline types ───────────────────────────────────────────────────────────

type PipelineType string

const (
	Pipeline_traffic PipelineType = "direct_traffic"
	Pipeline_lead    PipelineType = "lead_validation"
)

// ─── Input types ──────────────────────────────────────────────────────────────

// DestinationParams holds cost parameters per destination for the Traffic Pipeline.
type DestinationParams struct {
	Destination_id string  `json:"destination_id"`
	Anchor_cost    float64 `json:"anchor_cost"`    // A_d — ideal cost per conversion
	Cost_variance  float64 `json:"cost_variance"`  // B_d — additional ceiling above anchor
	Conversions    float64 `json:"conversions"`    // I_{t,d} — actual conversions for this destination
}

// KPBBonusParams holds one KPB option configuration for the Lead Pipeline.
type KPBBonusParams struct {
	Option_id     string  `json:"option_id"`
	Bonus_amount  float64 `json:"bonus_amount"`  // kpb_k — fixed NGN bonus per trigger
	Trigger_count int     `json:"trigger_count"` // KPB_{t,k} — how many times this advocate triggered it
}

// TalentPayoutInput holds per-advocate input for one cycle payout calculation.
type TalentPayoutInput struct {
	Talent_id        string  `json:"talent_id"`
	Pipeline         PipelineType `json:"pipeline"`
	Allocated_budget float64 `json:"allocated_budget"` // C_t

	// Traffic Pipeline fields
	Destinations []DestinationParams `json:"destinations,omitempty"`

	// Lead Pipeline fields
	Valid_leads      float64          `json:"valid_leads,omitempty"`
	Anchor_lead_cost float64          `json:"anchor_lead_cost,omitempty"` // A_L
	Cost_variance_l  float64          `json:"cost_variance_l,omitempty"`  // B_L
	KPB_bonuses      []KPBBonusParams `json:"kpb_bonuses,omitempty"`
	KPB_from_pool    bool             `json:"kpb_from_pool,omitempty"` // true = separate pool; false = deducted from C_t

	// Shared
	Commission_rate  float64 `json:"commission_rate"`  // r_c (e.g. 0.30 = 30%)
	Status           string  `json:"status"`           // "active" | "removed_forfeit" | "removed_payout"
	Fallback_flagged bool    `json:"fallback_flagged"` // any conversions are fallback-sourced (Story 24)
}

// CyclePoolInput holds cycle-level pool data.
type CyclePoolInput struct {
	Available_pool float64 `json:"available_pool"` // AvailablePool_cycle
	KPB_pool       float64 `json:"kpb_pool"`       // separate KPB pool balance (Lead Pipeline, toggle OFF)
}

// ─── Output types ─────────────────────────────────────────────────────────────

// TalentPayoutResult is the per-advocate audit record (Story 17 P.10).
type TalentPayoutResult struct {
	Talent_id string       `json:"talent_id"`
	Pipeline  PipelineType `json:"pipeline"`
	Status    string       `json:"status"`

	// Pre-commission
	Gross_base  float64 `json:"gross_base"`
	KPB_total   float64 `json:"kpb_total"`
	Gross_total float64 `json:"gross_total"`

	// Cap evaluation
	Cost_per_unit   float64 `json:"cost_per_unit"`   // cost per conversion (Traffic) or per lead (Lead)
	Cap_applied     float64 `json:"cap_applied"`     // A+B cap enforced
	Cap_exceeded    bool    `json:"cap_exceeded"`
	Excess_forfeited float64 `json:"excess_forfeited"`

	// Post-commission
	Commission_amount float64 `json:"commission_amount"`
	E_net             float64 `json:"e_net"`

	// Post-scaling
	Scale_factor float64 `json:"scale_factor"`
	Final_payout float64 `json:"final_payout"`

	// Audit flags
	Fallback_flagged bool   `json:"fallback_flagged"`
	KPB_pool_source  string `json:"kpb_pool_source"` // "allocated_budget" | "separate_pool" | "n/a"
}

// CyclePayoutOutput holds the full cycle payout result.
type CyclePayoutOutput struct {
	Results           []TalentPayoutResult `json:"results"`
	Total_paid        float64              `json:"total_paid"`
	Unspent           float64              `json:"unspent"`
	Forfeits_returned float64              `json:"forfeits_returned"`
	Unallocated_next  float64              `json:"unallocated_next"` // rolls to next cycle
	KPB_unspent       float64              `json:"kpb_unspent"`
	Scale_factor      float64              `json:"scale_factor"`
}

// ─── Main entry point ─────────────────────────────────────────────────────────

// ComputeCyclePayout runs the full 10-stage payout algorithm for a cycle.
// Handles both pipeline types. All monetary values in NGN with kobo precision.
// Story 17 P.1 through P.10.
func ComputeCyclePayout(talents []TalentPayoutInput, pool CyclePoolInput) CyclePayoutOutput {
	results := make([]TalentPayoutResult, len(talents))
	total_gross_net := 0.0
	total_kpb_demanded := 0.0
	forfeits_returned := 0.0

	// P.1–P.6: per-talent calculation
	for i, t := range talents {
		r := TalentPayoutResult{
			Talent_id:        t.Talent_id,
			Pipeline:         t.Pipeline,
			Status:           t.Status,
			Fallback_flagged: t.Fallback_flagged,
		}

		// P.6 — forfeit: return full allocation to pool, payout = 0
		if t.Status == "removed_forfeit" {
			r.Final_payout = 0
			forfeits_returned += t.Allocated_budget
			results[i] = r
			continue
		}

		// P.2 / P.3 — base payout by pipeline
		switch t.Pipeline {
		case Pipeline_traffic:
			r = computeTrafficPayout(t, r)
		case Pipeline_lead:
			r = computeLeadPayout(t, r)
		}

		// P.5 — commission
		r.Commission_amount = MoneyRound(r.Gross_total * t.Commission_rate)
		r.E_net = MoneyRound(r.Gross_total * (1 - t.Commission_rate))

		total_gross_net += r.E_net
		total_kpb_demanded += r.KPB_total
		results[i] = r
	}

	// P.7 — pool scaling
	scale_factor := 1.0
	if total_gross_net > pool.Available_pool && pool.Available_pool > 0 {
		scale_factor = pool.Available_pool / total_gross_net
	}

	total_paid := 0.0
	for i, t := range talents {
		if results[i].Status == "removed_forfeit" {
			continue
		}
		results[i].Scale_factor = scale_factor
		results[i].Final_payout = MoneyRound(results[i].E_net * scale_factor)
		total_paid += results[i].Final_payout

		// Determine KPB pool source label for audit
		if t.Pipeline == Pipeline_lead {
			if t.KPB_from_pool {
				results[i].KPB_pool_source = "separate_pool"
			} else {
				results[i].KPB_pool_source = "allocated_budget"
			}
		} else {
			results[i].KPB_pool_source = "n/a"
		}
	}

	// P.8 — reconcile
	unspent := MoneyRound(pool.Available_pool - total_paid)
	kpb_unspent := MoneyRound(pool.KPB_pool - total_kpb_demanded)
	if kpb_unspent < 0 {
		kpb_unspent = 0
	}
	unallocated_next := MoneyRound(unspent + forfeits_returned)

	return CyclePayoutOutput{
		Results:           results,
		Total_paid:        MoneyRound(total_paid),
		Unspent:           unspent,
		Forfeits_returned: forfeits_returned,
		Unallocated_next:  unallocated_next,
		KPB_unspent:       kpb_unspent,
		Scale_factor:      scale_factor,
	}
}

// ─── Traffic Pipeline (Story 17 P.2) ─────────────────────────────────────────

func computeTrafficPayout(t TalentPayoutInput, r TalentPayoutResult) TalentPayoutResult {
	base_value := 0.0
	total_conversions := 0.0
	cap_d := 0.0

	for _, d := range t.Destinations {
		contribution := MoneyRound(d.Conversions * (d.Anchor_cost + d.Cost_variance))
		base_value += contribution
		total_conversions += d.Conversions
		if cap_d == 0 {
			cap_d = d.Anchor_cost + d.Cost_variance
		}
	}

	// Gross base capped at C_t
	gross_base := base_value
	if gross_base > t.Allocated_budget {
		gross_base = t.Allocated_budget
	}
	gross_base = MoneyRound(gross_base)

	// Cost per conversion vs cap check
	cost_per_conversion := 0.0
	if total_conversions > 0 {
		cost_per_conversion = MoneyRound(t.Allocated_budget / total_conversions)
	}

	cap_exceeded := false
	excess_forfeited := 0.0
	if cost_per_conversion > cap_d && cap_d > 0 && total_conversions > 0 {
		cap_exceeded = true
		capped_payout := MoneyRound(total_conversions * cap_d)
		excess_forfeited = MoneyRound(gross_base - capped_payout)
		if excess_forfeited < 0 {
			excess_forfeited = 0
		}
		gross_base = capped_payout
	}

	r.Gross_base = gross_base
	r.Gross_total = gross_base
	r.Cost_per_unit = cost_per_conversion
	r.Cap_applied = cap_d
	r.Cap_exceeded = cap_exceeded
	r.Excess_forfeited = excess_forfeited
	return r
}

// ─── Lead Pipeline (Story 17 P.3) ────────────────────────────────────────────

func computeLeadPayout(t TalentPayoutInput, r TalentPayoutResult) TalentPayoutResult {
	cap_l := t.Anchor_lead_cost + t.Cost_variance_l

	gross_base := 0.0
	cost_per_lead := 0.0
	cap_exceeded := false
	excess_forfeited := 0.0

	if t.Valid_leads > 0 {
		cost_per_lead = MoneyRound(t.Allocated_budget / t.Valid_leads)
		if cost_per_lead <= cap_l {
			gross_base = MoneyRound(t.Allocated_budget)
		} else {
			cap_exceeded = true
			gross_base = MoneyRound(t.Valid_leads * cap_l)
			excess_forfeited = MoneyRound(t.Allocated_budget - gross_base)
			if excess_forfeited < 0 {
				excess_forfeited = 0
			}
		}
	}

	// KPB bonuses (Story 17 P.3.3)
	kpb_total := 0.0
	for _, k := range t.KPB_bonuses {
		kpb_total += MoneyRound(float64(k.Trigger_count) * k.Bonus_amount)
	}

	if !t.KPB_from_pool {
		// KPB deducted from C_t — combined base + KPB cannot exceed C_t
		combined := gross_base + kpb_total
		if combined > t.Allocated_budget {
			kpb_total = MoneyRound(t.Allocated_budget - gross_base)
			if kpb_total < 0 {
				kpb_total = 0
			}
		}
	}
	kpb_total = MoneyRound(kpb_total)

	r.Gross_base = gross_base
	r.KPB_total = kpb_total
	r.Gross_total = MoneyRound(gross_base + kpb_total)
	r.Cost_per_unit = cost_per_lead
	r.Cap_applied = cap_l
	r.Cap_exceeded = cap_exceeded
	r.Excess_forfeited = excess_forfeited
	return r
}

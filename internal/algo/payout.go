package algo

import "errors"

// CycleRecord holds all data needed to compute end-of-cycle payout.
// PDC source branches on cycle_number:
//
//	cycle_number == 1  -> use PDC1_baseline (LTEC, from ltl-bayesian)
//	cycle_number >= 2  -> use STEC_pdc_next (from stec, via daily job)
type CycleRecord struct {
	Talent_id         string  `json:"talent_id"`
	Cycle_id          string  `json:"cycle_id"`
	Cycle_number      int     `json:"cycle_number"`
	Total_conversions float64 `json:"total_conversions"`
	Days_active       int     `json:"days_active"`
	Rate_per_day      float64 `json:"rate_per_day"`
	Slot_budget       float64 `json:"slot_budget"`
	KPI_target        float64 `json:"kpi_target"`
	PDC1_baseline     float64 `json:"pdc1_baseline"`
	STEC_pdc_next     float64 `json:"stec_pdc_next"`
	Market_cap        float64 `json:"market_cap"`
	Score_cap_pct     float64 `json:"score_cap_pct"`
}

type PayoutResult struct {
	Talent_id         string  `json:"talent_id"`
	Performance_score float64 `json:"performance_score"`
	Raw_payout        float64 `json:"raw_payout"`
	Final_payout      float64 `json:"final_payout"`
	Flagged           bool    `json:"flagged"`
	Flag_reason       string  `json:"flag_reason"`
}

func ComputePayout(r CycleRecord) (PayoutResult, error) {
	if r.KPI_target <= 0 {
		return PayoutResult{}, errors.New("kpi_target must be positive")
	}
	if err := requireNonNegative("total_conversions", r.Total_conversions); err != nil {
		return PayoutResult{}, err
	}
	if err := requireNonNegative("rate_per_day", r.Rate_per_day); err != nil {
		return PayoutResult{}, err
	}
	if err := requireNonNegative("slot_budget", r.Slot_budget); err != nil {
		return PayoutResult{}, err
	}
	if r.Score_cap_pct <= 0 {
		return PayoutResult{}, errors.New("score_cap_pct must be positive")
	}
	if r.Days_active < 0 {
		return PayoutResult{}, errors.New("days_active must be non-negative")
	}

	var pdc_reference float64
	if r.Cycle_number <= 1 {
		pdc_reference = r.PDC1_baseline
	} else {
		pdc_reference = r.STEC_pdc_next
	}

	score := r.Total_conversions / r.KPI_target
	if score > r.Score_cap_pct {
		score = r.Score_cap_pct
	}

	raw := r.Rate_per_day * float64(r.Days_active) * score
	final := raw
	if final > r.Slot_budget {
		final = r.Slot_budget
	}

	flagged := false
	flag_reason := ""

	switch {
	case pdc_reference <= 0:
		flagged = true
		flag_reason = "invalid_pdc_reference"
	case score < 0.5:
		flagged = true
		flag_reason = "performance_below_50_percent"
	}

	return PayoutResult{
		Talent_id:         r.Talent_id,
		Performance_score: score,
		Raw_payout:        raw,
		Final_payout:      final,
		Flagged:           flagged,
		Flag_reason:       flag_reason,
	}, nil
}

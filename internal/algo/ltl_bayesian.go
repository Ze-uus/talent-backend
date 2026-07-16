package algo

import "errors"

// TalentBaseline holds the long-term performance record per talent.
// Lambda_lt is the LTEC (Long-Term Efficiency Coefficient) used as PDC for Cycle 1.
type TalentBaseline struct {
	Talent_id  string  `json:"talent_id"`
	Alpha_lt   float64 `json:"alpha_lt"`
	Beta_lt    float64 `json:"beta_lt"`
	Lambda_lt  float64 `json:"lambda_lt"`
	Sigma_hist float64 `json:"sigma_hist"`
	Delta_lt   float64 `json:"delta_lt"`
}

// CategoryBaseline holds the conservative median per talent category.
// Per SOP: use 40-50th percentile, NOT mean.
// Tier C (student):    2/day
// Tier B (micro):      5/day
// Tier A (community): 15/day
type CategoryBaseline struct {
	Category string  `json:"category"`
	Median   float64 `json:"median"`
}

func UpdateBaseline(b TalentBaseline, k_today float64) (TalentBaseline, error) {
	if !isFinite(k_today) {
		return TalentBaseline{}, errors.New("k_today must be finite")
	}
	if k_today < 0 {
		return TalentBaseline{}, errors.New("k_today must be non-negative")
	}
	if !isFinite(b.Alpha_lt) || !isFinite(b.Beta_lt) || !isFinite(b.Delta_lt) {
		return TalentBaseline{}, errors.New("baseline contains non-finite values")
	}

	b.Alpha_lt = b.Alpha_lt*b.Delta_lt + k_today
	b.Beta_lt = b.Beta_lt*b.Delta_lt + 1.0
	if b.Beta_lt > 0 {
		b.Lambda_lt = b.Alpha_lt / b.Beta_lt
	}
	return b, nil
}

func UpdateSigmaHist(b TalentBaseline, recent_outputs []float64) (TalentBaseline, error) {
	if len(recent_outputs) == 0 {
		return b, nil
	}
	for _, v := range recent_outputs {
		if !isFinite(v) {
			return TalentBaseline{}, errors.New("recent_outputs must contain only finite values")
		}
	}
	mu := mean(recent_outputs)
	b.Sigma_hist = stdDev(recent_outputs, mu)
	return b, nil
}

func BootstrapBaseline(cat CategoryBaseline, delta_lt float64) (TalentBaseline, error) {
	if err := requirePositive("median", cat.Median); err != nil {
		return TalentBaseline{}, err
	}
	if !isFinite(delta_lt) || delta_lt <= 0 || delta_lt >= 1 {
		return TalentBaseline{}, errors.New("delta_lt must be in (0, 1)")
	}
	return TalentBaseline{
		Alpha_lt:   cat.Median,
		Beta_lt:    1.0,
		Lambda_lt:  cat.Median,
		Sigma_hist: 0,
		Delta_lt:   delta_lt,
	}, nil
}

// Cycle1PDC returns safe starting allocation for Cycle 1 (cold-start).
// PDC1 = min(lambda_lt, market_cap). No STEC, no Z factor applied.
func Cycle1PDC(b TalentBaseline, market_cap float64) (float64, error) {
	if !isFinite(b.Lambda_lt) {
		return 0, errors.New("lambda_lt must be finite")
	}
	pdc := b.Lambda_lt
	if pdc < 0 {
		pdc = 0
	}
	if isFinite(market_cap) && market_cap > 0 && pdc > market_cap {
		pdc = market_cap
	}
	return pdc, nil
}

// CycleNPDC returns safe allocation for Cycle 2+ using STEC output.
func CycleNPDC(stec_pdc float64, market_cap float64) (float64, error) {
	if !isFinite(stec_pdc) {
		return 0, errors.New("stec_pdc must be finite")
	}
	if stec_pdc < 0 {
		return 0, nil
	}
	if isFinite(market_cap) && market_cap > 0 && stec_pdc > market_cap {
		return market_cap, nil
	}
	return stec_pdc, nil
}

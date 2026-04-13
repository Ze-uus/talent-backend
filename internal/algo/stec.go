package algo

import (
	"errors"
	"math"
)

type DailyOutputs struct {
	Talent_id     string    `json:"talent_id"`
	Cycle_id      string    `json:"cycle_id"`
	Outputs       []float64 `json:"outputs"`
	PDC_allocated float64   `json:"pdc_allocated"`
}

// STECResult exposes all three SOP report metrics:
// Mu_ur    = Output Efficiency Score (SOP section 15.7)
// Sigma_ur = Stability Score (SOP section 15.7)
// ZDR      = Hard Exhaustion metric (SOP section 15.7)
type STECResult struct {
	URs      []float64 `json:"urs"`
	Mu_ur    float64   `json:"mu_ur"`
	Sigma_ur float64   `json:"sigma_ur"`
	REL      float64   `json:"rel"`
	ZDR      float64   `json:"zdr"`
	Pattern  string    `json:"pattern"`
	Delta_st float64   `json:"delta_st"`
	PDC_next float64   `json:"pdc_next"`
}

func ComputeSTEC(d DailyOutputs, z float64) (STECResult, error) {
	if len(d.Outputs) == 0 {
		return STECResult{}, errors.New("outputs must not be empty")
	}
	if d.PDC_allocated <= 0 {
		return STECResult{}, errors.New("pdc_allocated must be positive")
	}
	if !isFinite(d.PDC_allocated) {
		return STECResult{}, errors.New("pdc_allocated must be finite")
	}
	if !isFinite(z) {
		return STECResult{}, errors.New("z must be finite")
	}

	n := len(d.Outputs)
	urs := make([]float64, n)
	for i, x := range d.Outputs {
		if !isFinite(x) {
			return STECResult{}, errors.New("outputs must contain only finite values")
		}
		urs[i] = x / d.PDC_allocated
	}

	mu_ur := mean(urs)
	sigma_ur := stdDev(urs, mu_ur)

	half := n / 2
	if half == 0 {
		half = 1
	}
	mu_early := mean(urs[:half])
	mu_late := mean(urs[half:])
	rel := 1.0
	if mu_early > 0 {
		rel = mu_late / mu_early
	}

	zero_days := 0
	for _, u := range urs {
		if u <= 0.1 {
			zero_days++
		}
	}
	zdr := float64(zero_days) / float64(n)

	pattern, delta_st := classifyPattern(mu_ur, sigma_ur, rel, zdr)

	weights := make([]float64, n)
	for i := range d.Outputs {
		exp_val := float64(n - 1 - i)
		weights[i] = math.Pow(delta_st, exp_val)
	}
	mu_st := weightedMean(d.Outputs, weights)
	sigma_st := weightedStdDev(d.Outputs, weights, mu_st)
	pdc_next := mu_st - z*sigma_st
	if pdc_next < 0 {
		pdc_next = 0
	}
	if !isFinite(pdc_next) {
		pdc_next = 0
	}

	return STECResult{
		URs: urs, Mu_ur: mu_ur, Sigma_ur: sigma_ur,
		REL: rel, ZDR: zdr,
		Pattern: pattern, Delta_st: delta_st, PDC_next: pdc_next,
	}, nil
}

func classifyPattern(mu_ur, sigma_ur, rel, zdr float64) (string, float64) {
	switch {
	case zdr >= 0.3 || rel <= 0.5:
		return "collapse", 0.50
	case mu_ur >= 0.8 && sigma_ur <= 0.2 && rel >= 0.85 && rel <= 1.15 && zdr == 0:
		return "stable", 0.20
	case mu_ur >= 0.6 && rel < 0.75 && sigma_ur > 0.2 && zdr < 0.3:
		if rel > 0.6 {
			return "burnout", 0.30
		}
		return "burnout", 0.40
	case sigma_ur >= 0.35 && rel >= 0.6 && rel <= 1.2 && zdr < 0.3:
		return "volatile", 0.30
	default:
		return "stable", 0.20
	}
}

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	s := 0.0
	for _, v := range vals {
		s += v
	}
	return s / float64(len(vals))
}

func stdDev(vals []float64, mu float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	s := 0.0
	for _, v := range vals {
		s += (v - mu) * (v - mu)
	}
	return math.Sqrt(s / float64(len(vals)))
}

func weightedMean(vals, weights []float64) float64 {
	sw, swv := 0.0, 0.0
	for i, v := range vals {
		sw += weights[i]
		swv += weights[i] * v
	}
	if sw == 0 {
		return 0
	}
	return swv / sw
}

func weightedStdDev(vals, weights []float64, mu float64) float64 {
	sw, swv := 0.0, 0.0
	for i, v := range vals {
		sw += weights[i]
		swv += weights[i] * (v - mu) * (v - mu)
	}
	if sw == 0 {
		return 0
	}
	return math.Sqrt(swv / sw)
}

package algo

import (
	"fmt"
	"math"
)

const minCyclesForSuggestion = 7

type CycleHistory struct {
	Cycle_number     int         `json:"cycle_number"`
	Cycle_budget     float64     `json:"cycle_budget"`
	Config           CycleConfig `json:"config"`
	Total_allocated  float64     `json:"total_allocated"`
	Remainder        float64     `json:"remainder"`
	Utilization      float64     `json:"utilization"`
	Avg_performance  float64     `json:"avg_performance"`
	Talent_count     int         `json:"talent_count"`
	Unassigned_count int         `json:"unassigned_count"`
	Avg_payout_ratio float64     `json:"avg_payout_ratio"`
}

type SuggestionFactor struct {
	Metric    string  `json:"metric"`
	Value     float64 `json:"value"`
	Direction string  `json:"direction"`
	Reasoning string  `json:"reasoning"`
}

type CapSuggestion struct {
	Suggested_config CycleConfig        `json:"suggested_config"`
	Confidence       float64            `json:"confidence"`
	Factors          []SuggestionFactor `json:"factors"`
	Min_cycles_used  int                `json:"min_cycles_used"`
}

func SuggestionReady(cycle_count int) bool {
	return cycle_count >= minCyclesForSuggestion
}

func SuggestConfig(history []CycleHistory, next_budget float64) (CapSuggestion, error) {
	if len(history) < minCyclesForSuggestion {
		return CapSuggestion{}, fmt.Errorf(
			"insufficient data: %d cycles provided, need at least %d",
			len(history), minCyclesForSuggestion,
		)
	}
	if err := requirePositive("next_budget", next_budget); err != nil {
		return CapSuggestion{}, err
	}

	last := history[len(history)-1]
	base := last.Config
	if err := base.Validate(); err != nil {
		return CapSuggestion{}, fmt.Errorf("last cycle config invalid: %w", err)
	}

	var factors []SuggestionFactor
	suggested := base

	util_factor := analyzeUtilization(history)
	factors = append(factors, util_factor)
	suggested = applyUtilizationAdjustment(suggested, util_factor, next_budget)

	waste_factor := analyzeWaste(history)
	factors = append(factors, waste_factor)
	suggested = applyWasteAdjustment(suggested, waste_factor)

	unassign_factor := analyzeUnassignment(history)
	factors = append(factors, unassign_factor)
	suggested = applyUnassignmentAdjustment(suggested, unassign_factor)

	perf_factor := analyzePerformance(history)
	factors = append(factors, perf_factor)
	suggested = applyPerformanceAdjustment(suggested, perf_factor)

	suggested = clampConfig(suggested)
	confidence := computeConfidence(history, factors)

	return CapSuggestion{
		Suggested_config: suggested,
		Confidence:       confidence,
		Factors:          factors,
		Min_cycles_used:  len(history),
	}, nil
}

func analyzeUtilization(history []CycleHistory) SuggestionFactor {
	n := len(history)
	recent := history
	if n > 10 {
		recent = history[n-10:]
	}

	vals := make([]float64, len(recent))
	for i, h := range recent {
		vals[i] = h.Utilization
	}
	avg := mean(vals)

	half := len(vals) / 2
	early_avg := mean(vals[:half])
	late_avg := mean(vals[half:])
	trend := late_avg - early_avg

	direction := "maintain"
	reasoning := ""

	switch {
	case avg < 0.60:
		direction = "decrease"
		reasoning = fmt.Sprintf(
			"Budget utilization averaged %.0f%% over the last %d cycles, "+
				"indicating the tier cap is likely too high for the available talent pool.",
			avg*100, len(recent),
		)
	case avg > 0.95:
		direction = "increase"
		reasoning = fmt.Sprintf(
			"Budget utilization averaged %.0f%% over the last %d cycles, "+
				"suggesting the cap could be raised to allow more allocation headroom.",
			avg*100, len(recent),
		)
	case trend < -0.10:
		direction = "decrease"
		reasoning = fmt.Sprintf(
			"Budget utilization is declining (%.0f%% early vs %.0f%% recent), "+
				"suggesting the cap should be lowered to match shrinking demand.",
			early_avg*100, late_avg*100,
		)
	default:
		reasoning = fmt.Sprintf(
			"Budget utilization is healthy at %.0f%% with stable trend. No cap adjustment needed.",
			avg*100,
		)
	}

	return SuggestionFactor{
		Metric:    "utilization_trend",
		Value:     avg,
		Direction: direction,
		Reasoning: reasoning,
	}
}

func analyzeWaste(history []CycleHistory) SuggestionFactor {
	n := len(history)
	recent := history
	if n > 10 {
		recent = history[n-10:]
	}

	ratios := make([]float64, 0, len(recent))
	for _, h := range recent {
		if h.Cycle_budget > 0 {
			ratios = append(ratios, h.Remainder/h.Cycle_budget)
		}
	}
	if len(ratios) == 0 {
		return SuggestionFactor{
			Metric: "waste_ratio", Direction: "maintain",
			Reasoning: "No budget data available to analyze waste.",
		}
	}

	avg_waste := mean(ratios)
	direction := "maintain"
	reasoning := ""

	switch {
	case avg_waste > 0.30:
		direction = "decrease"
		reasoning = fmt.Sprintf(
			"Average budget waste is %.0f%%. The budget share percentage should be reduced "+
				"to tighten allocation and minimize unspent budget.",
			avg_waste*100,
		)
	case avg_waste < 0.05:
		direction = "increase"
		reasoning = fmt.Sprintf(
			"Budget waste is very low at %.0f%%, indicating allocation is near-optimal "+
				"or potentially too tight. A small increase may provide flexibility.",
			avg_waste*100,
		)
	default:
		reasoning = fmt.Sprintf(
			"Budget waste is at %.0f%%, within acceptable range. No budget share adjustment needed.",
			avg_waste*100,
		)
	}

	return SuggestionFactor{
		Metric:    "waste_ratio",
		Value:     avg_waste,
		Direction: direction,
		Reasoning: reasoning,
	}
}

func analyzeUnassignment(history []CycleHistory) SuggestionFactor {
	n := len(history)
	recent := history
	if n > 10 {
		recent = history[n-10:]
	}

	rates := make([]float64, 0, len(recent))
	for _, h := range recent {
		if h.Talent_count > 0 {
			rates = append(rates, float64(h.Unassigned_count)/float64(h.Talent_count))
		}
	}
	if len(rates) == 0 {
		return SuggestionFactor{
			Metric: "unassignment_rate", Direction: "maintain",
			Reasoning: "No talent data available to analyze unassignment.",
		}
	}

	avg_rate := mean(rates)

	half := len(rates) / 2
	if half == 0 {
		half = 1
	}
	early_rate := mean(rates[:half])
	late_rate := mean(rates[half:])
	trending_up := late_rate-early_rate > 0.05

	direction := "maintain"
	reasoning := ""

	switch {
	case avg_rate > 0.25:
		direction = "decrease"
		reasoning = fmt.Sprintf(
			"%.0f%% of talents are going unassigned on average. "+
				"Tier caps may be too restrictive -- consider lowering the minimum tier "+
				"or widening the tier range.",
			avg_rate*100,
		)
	case trending_up && avg_rate > 0.10:
		direction = "decrease"
		reasoning = fmt.Sprintf(
			"Unassignment rate is rising (%.0f%% early vs %.0f%% recent). "+
				"Tier constraints are becoming too narrow for the talent pool.",
			early_rate*100, late_rate*100,
		)
	case avg_rate < 0.05:
		reasoning = fmt.Sprintf(
			"Unassignment rate is excellent at %.0f%%. Tier configuration is well-matched to the talent pool.",
			avg_rate*100,
		)
	default:
		reasoning = fmt.Sprintf(
			"Unassignment rate is %.0f%%, within normal range.",
			avg_rate*100,
		)
	}

	return SuggestionFactor{
		Metric:    "unassignment_rate",
		Value:     avg_rate,
		Direction: direction,
		Reasoning: reasoning,
	}
}

func analyzePerformance(history []CycleHistory) SuggestionFactor {
	n := len(history)
	recent := history
	if n > 10 {
		recent = history[n-10:]
	}

	perfs := make([]float64, 0, len(recent))
	for _, h := range recent {
		if h.Avg_performance > 0 {
			perfs = append(perfs, h.Avg_performance)
		}
	}
	if len(perfs) == 0 {
		return SuggestionFactor{
			Metric: "performance_distribution", Direction: "maintain",
			Reasoning: "No performance data available to analyze.",
		}
	}

	avg_perf := mean(perfs)
	last_cfg := history[n-1].Config

	direction := "maintain"
	reasoning := ""

	near_cap_threshold := last_cfg.Score_cap_pct * 0.90

	switch {
	case avg_perf >= near_cap_threshold:
		direction = "increase"
		reasoning = fmt.Sprintf(
			"Average performance score (%.2f) is near the score cap (%.2f). "+
				"Consider raising the score cap to reward top performers and differentiate payouts.",
			avg_perf, last_cfg.Score_cap_pct,
		)
	case avg_perf < 0.50:
		direction = "decrease"
		reasoning = fmt.Sprintf(
			"Average performance score is low at %.2f. "+
				"Tier values may be too aggressive relative to talent capability. "+
				"Consider lowering the max tier cap.",
			avg_perf,
		)
	default:
		reasoning = fmt.Sprintf(
			"Average performance score is %.2f, within healthy range.",
			avg_perf,
		)
	}

	return SuggestionFactor{
		Metric:    "performance_distribution",
		Value:     avg_perf,
		Direction: direction,
		Reasoning: reasoning,
	}
}

func applyUtilizationAdjustment(cfg CycleConfig, f SuggestionFactor, next_budget float64) CycleConfig {
	switch f.Direction {
	case "decrease":
		new_cap := int(math.Floor(next_budget*cfg.Budget_share_pct*0.85/float64(cfg.Tier_increment))) * cfg.Tier_increment
		if new_cap >= cfg.Min_tier && new_cap < cfg.Max_tier_cap {
			cfg.Max_tier_cap = new_cap
		}
	case "increase":
		new_cap := cfg.Max_tier_cap + cfg.Tier_increment
		cfg.Max_tier_cap = new_cap
	}
	return cfg
}

func applyWasteAdjustment(cfg CycleConfig, f SuggestionFactor) CycleConfig {
	switch f.Direction {
	case "decrease":
		cfg.Budget_share_pct = math.Max(0.05, cfg.Budget_share_pct*0.90)
	case "increase":
		cfg.Budget_share_pct = math.Min(1.0, cfg.Budget_share_pct*1.05)
	}
	return cfg
}

func applyUnassignmentAdjustment(cfg CycleConfig, f SuggestionFactor) CycleConfig {
	if f.Direction == "decrease" {
		if cfg.Min_tier > cfg.Tier_increment {
			cfg.Min_tier -= cfg.Tier_increment
		}
	}
	return cfg
}

func applyPerformanceAdjustment(cfg CycleConfig, f SuggestionFactor) CycleConfig {
	switch f.Direction {
	case "increase":
		cfg.Score_cap_pct = math.Min(10.0, cfg.Score_cap_pct+0.25)
	case "decrease":
		new_cap := cfg.Max_tier_cap - cfg.Tier_increment
		if new_cap >= cfg.Min_tier {
			cfg.Max_tier_cap = new_cap
		}
	}
	return cfg
}

func clampConfig(cfg CycleConfig) CycleConfig {
	if cfg.Min_tier < cfg.Tier_increment {
		cfg.Min_tier = cfg.Tier_increment
	}
	if cfg.Max_tier_cap < cfg.Min_tier {
		cfg.Max_tier_cap = cfg.Min_tier
	}
	if cfg.Budget_share_pct < 0.05 {
		cfg.Budget_share_pct = 0.05
	}
	if cfg.Budget_share_pct > 1.0 {
		cfg.Budget_share_pct = 1.0
	}
	if cfg.Score_cap_pct < 0.5 {
		cfg.Score_cap_pct = 0.5
	}
	if cfg.Score_cap_pct > 10.0 {
		cfg.Score_cap_pct = 10.0
	}
	cfg.Min_tier = roundToIncrement(cfg.Min_tier, cfg.Tier_increment)
	cfg.Max_tier_cap = roundToIncrement(cfg.Max_tier_cap, cfg.Tier_increment)
	if cfg.Max_tier_cap < cfg.Min_tier {
		cfg.Max_tier_cap = cfg.Min_tier
	}
	return cfg
}

func roundToIncrement(val, increment int) int {
	if increment <= 0 {
		return val
	}
	return (val / increment) * increment
}

func computeConfidence(history []CycleHistory, factors []SuggestionFactor) float64 {
	base := 0.5

	cycle_bonus := math.Min(0.3, float64(len(history)-minCyclesForSuggestion)*0.03)
	base += cycle_bonus

	maintain_count := 0
	for _, f := range factors {
		if f.Direction == "maintain" {
			maintain_count++
		}
	}
	if len(factors) > 0 {
		consistency := float64(maintain_count) / float64(len(factors))
		base += consistency * 0.2
	}

	if base > 1.0 {
		base = 1.0
	}
	return math.Round(base*100) / 100
}

func SuggestConfigWithError(history []CycleHistory, next_budget float64) (CapSuggestion, []error) {
	suggestion, err := SuggestConfig(history, next_budget)
	if err != nil {
		return CapSuggestion{}, []error{err}
	}

	var warnings []error
	if err := suggestion.Suggested_config.Validate(); err != nil {
		warnings = append(warnings, fmt.Errorf("suggested config has validation issues: %w", err))
	}
	return suggestion, warnings
}

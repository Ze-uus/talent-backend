package algo

import (
	"errors"
	"fmt"
	"math"
)

type CycleConfig struct {
	Max_tier_cap     int     `json:"max_tier_cap"`
	Min_tier         int     `json:"min_tier"`
	Tier_increment   int     `json:"tier_increment"`
	Budget_share_pct float64 `json:"budget_share_pct"`
	Score_cap_pct    float64 `json:"score_cap_pct"`
}

func DefaultConfig() CycleConfig {
	return CycleConfig{
		Max_tier_cap:     55000,
		Min_tier:         5000,
		Tier_increment:   5000,
		Budget_share_pct: 0.25,
		Score_cap_pct:    1.5,
	}
}

func (c CycleConfig) Validate() error {
	if c.Max_tier_cap <= 0 {
		return errors.New("max_tier_cap must be positive")
	}
	if c.Min_tier <= 0 {
		return errors.New("min_tier must be positive")
	}
	if c.Tier_increment <= 0 {
		return errors.New("tier_increment must be positive")
	}
	if c.Min_tier > c.Max_tier_cap {
		return fmt.Errorf("min_tier (%d) exceeds max_tier_cap (%d)", c.Min_tier, c.Max_tier_cap)
	}
	if c.Min_tier%c.Tier_increment != 0 {
		return fmt.Errorf("min_tier (%d) must be a multiple of tier_increment (%d)", c.Min_tier, c.Tier_increment)
	}
	if c.Max_tier_cap%c.Tier_increment != 0 {
		return fmt.Errorf("max_tier_cap (%d) must be a multiple of tier_increment (%d)", c.Max_tier_cap, c.Tier_increment)
	}
	if c.Budget_share_pct <= 0 || c.Budget_share_pct > 1 {
		return fmt.Errorf("budget_share_pct must be in (0, 1], got %f", c.Budget_share_pct)
	}
	if c.Score_cap_pct <= 0 || c.Score_cap_pct > 10 {
		return fmt.Errorf("score_cap_pct must be in (0, 10], got %f", c.Score_cap_pct)
	}
	return nil
}

func ValidTiers(cfg CycleConfig) []int {
	if cfg.Tier_increment <= 0 || cfg.Min_tier <= 0 || cfg.Max_tier_cap <= 0 {
		return nil
	}
	var tiers []int
	for v := cfg.Min_tier; v <= cfg.Max_tier_cap; v += cfg.Tier_increment {
		tiers = append(tiers, v)
	}
	return tiers
}

func isFinite(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}

func requirePositive(name string, v float64) error {
	if !isFinite(v) {
		return fmt.Errorf("%s must be a finite number, got %f", name, v)
	}
	if v <= 0 {
		return fmt.Errorf("%s must be positive, got %f", name, v)
	}
	return nil
}

func requireNonNegative(name string, v float64) error {
	if !isFinite(v) {
		return fmt.Errorf("%s must be a finite number, got %f", name, v)
	}
	if v < 0 {
		return fmt.Errorf("%s must be non-negative, got %f", name, v)
	}
	return nil
}

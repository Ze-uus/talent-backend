package algo_test

import (
	"math"
	"testing"

	"github.com/Ze-uus/talent-backend/internal/algo"
)

func TestMatchScore_PerfectMatch(t *testing.T) {
	d := algo.MatchDimensions{DM: 1.0, GP: 1.0, OH: 1.0}
	r := algo.ComputeMatchScore(d)
	if math.Abs(r.MS_t-1.0) > 0.001 {
		t.Errorf("expected MS_t=1.0, got %.4f", r.MS_t)
	}
}

func TestMatchScore_ZeroMatch(t *testing.T) {
	d := algo.MatchDimensions{DM: 0.0, GP: 0.0, OH: 0.0}
	r := algo.ComputeMatchScore(d)
	if r.MS_t != 0 {
		t.Errorf("expected MS_t=0, got %.4f", r.MS_t)
	}
}

func TestMatchScore_Partial(t *testing.T) {
	d := algo.MatchDimensions{DM: 1.0, GP: 0.7, OH: 0.5}
	r := algo.ComputeMatchScore(d)
	expected := (1.0 + 0.7 + 0.5) / 3.0
	if math.Abs(r.MS_t-expected) > 0.001 {
		t.Errorf("expected %.4f, got %.4f", expected, r.MS_t)
	}
}

func TestMatchAdjustment_DefaultAlpha(t *testing.T) {
	// MS_t=1.0, alpha=0.20 → 1 - (1.0 × 0.20) = 0.80
	adj := algo.MatchAdjustment(1.0, algo.DefaultAlpha)
	if math.Abs(adj-0.80) > 0.001 {
		t.Errorf("expected 0.80, got %.4f", adj)
	}
}

func TestMatchAdjustment_ZeroMatchNoAdjustment(t *testing.T) {
	// MS_t=0.0 → no reduction, cost unchanged
	adj := algo.MatchAdjustment(0.0, algo.DefaultAlpha)
	if adj != 1.0 {
		t.Errorf("expected 1.0 (no adjustment), got %.4f", adj)
	}
}

func TestMatchAdjustment_AlphaCeiling(t *testing.T) {
	// alpha > 0.40 is clamped to 0.40 → 1 - (1.0 × 0.40) = 0.60 (floor)
	adj := algo.MatchAdjustment(1.0, 0.99)
	if math.Abs(adj-0.60) > 0.001 {
		t.Errorf("expected 0.60 (floor), got %.4f", adj)
	}
}

package algo_test

import (
	"math"
	"testing"

	"github.com/Ze-uus/talent-backend/internal/algo"
)

func TestSTEC_Stable(t *testing.T) {
	d := algo.DailyOutputs{
		Talent_id: "t1", Cycle_id: "c1",
		Outputs: []float64{20, 22, 19, 21, 20}, PDC_allocated: 20,
	}
	r, err := algo.ComputeSTEC(d, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Pattern != "stable" {
		t.Errorf("expected stable, got %s", r.Pattern)
	}
	if r.PDC_next <= 0 {
		t.Error("PDC_next should be positive for stable talent")
	}
}

func TestSTEC_Collapse(t *testing.T) {
	d := algo.DailyOutputs{Outputs: []float64{100, 100, 10, 0, 0}, PDC_allocated: 50}
	r, err := algo.ComputeSTEC(d, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Pattern != "collapse" {
		t.Errorf("expected collapse, got %s", r.Pattern)
	}
}

func TestSTEC_Burnout(t *testing.T) {
	// Declining output with rel ~0.64 (above collapse threshold 0.5, below burnout ceiling 0.75)
	d := algo.DailyOutputs{Outputs: []float64{30, 28, 22, 18, 16}, PDC_allocated: 25}
	r, err := algo.ComputeSTEC(d, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Pattern != "burnout" {
		t.Errorf("expected burnout, got %s (mu_ur=%.3f sigma_ur=%.3f rel=%.3f zdr=%.3f)",
			r.Pattern, r.Mu_ur, r.Sigma_ur, r.REL, r.ZDR)
	}
}

func TestSTEC_ReportMetrics(t *testing.T) {
	d := algo.DailyOutputs{Outputs: []float64{20, 20, 20}, PDC_allocated: 20}
	r, err := algo.ComputeSTEC(d, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Mu_ur != 1.0 {
		t.Errorf("Output Efficiency Score should be 1.0, got %.2f", r.Mu_ur)
	}
	if r.ZDR != 0 {
		t.Errorf("ZDR should be 0 for fully active talent, got %.2f", r.ZDR)
	}
}

func TestSTEC_ZeroPDCAllocated(t *testing.T) {
	d := algo.DailyOutputs{Outputs: []float64{10, 20}, PDC_allocated: 0}
	_, err := algo.ComputeSTEC(d, 1.0)
	if err == nil {
		t.Error("expected error for zero PDC_allocated")
	}
}

func TestSTEC_EmptyOutputs(t *testing.T) {
	d := algo.DailyOutputs{PDC_allocated: 10}
	_, err := algo.ComputeSTEC(d, 1.0)
	if err == nil {
		t.Error("expected error for empty outputs")
	}
}

func TestSTEC_NaNInOutputs(t *testing.T) {
	d := algo.DailyOutputs{Outputs: []float64{10, math.NaN(), 20}, PDC_allocated: 10}
	_, err := algo.ComputeSTEC(d, 1.0)
	if err == nil {
		t.Error("expected error for NaN in outputs")
	}
}

func TestSTEC_SingleOutput(t *testing.T) {
	d := algo.DailyOutputs{Outputs: []float64{15}, PDC_allocated: 10}
	r, err := algo.ComputeSTEC(d, 1.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.PDC_next < 0 {
		t.Error("PDC_next should be non-negative")
	}
}

func TestSTEC_NaNZ(t *testing.T) {
	d := algo.DailyOutputs{Outputs: []float64{10, 20}, PDC_allocated: 10}
	_, err := algo.ComputeSTEC(d, math.NaN())
	if err == nil {
		t.Error("expected error for NaN z")
	}
}

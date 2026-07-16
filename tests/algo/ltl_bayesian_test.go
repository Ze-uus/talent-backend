package algo_test

import (
	"math"
	"testing"

	"github.com/Ze-uus/talent-backend/internal/algo"
)

func TestUpdateBaseline_Decay(t *testing.T) {
	b := algo.TalentBaseline{Alpha_lt: 100, Beta_lt: 5, Delta_lt: 0.97}
	updated, err := algo.UpdateBaseline(b, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected_alpha := 100*0.97 + 20
	if math.Abs(updated.Alpha_lt-expected_alpha) > 0.001 {
		t.Errorf("alpha: got %.4f want %.4f", updated.Alpha_lt, expected_alpha)
	}
	expected_lambda := updated.Alpha_lt / updated.Beta_lt
	if math.Abs(updated.Lambda_lt-expected_lambda) > 0.001 {
		t.Error("lambda not recomputed")
	}
}

func TestBootstrap_UsesMedian(t *testing.T) {
	cat := algo.CategoryBaseline{Category: "student", Median: 2.0}
	b, err := algo.BootstrapBaseline(cat, 0.97)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b.Lambda_lt != 2.0 {
		t.Errorf("bootstrap lambda should be median 2.0, got %.2f", b.Lambda_lt)
	}
}

func TestCycle1PDC_MarketCapGate(t *testing.T) {
	b := algo.TalentBaseline{Lambda_lt: 30}
	pdc, err := algo.Cycle1PDC(b, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pdc != 20 {
		t.Errorf("expected 20 (market cap), got %.2f", pdc)
	}
}

func TestCycle1PDC_NoMarketCap(t *testing.T) {
	b := algo.TalentBaseline{Lambda_lt: 30}
	pdc, err := algo.Cycle1PDC(b, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pdc != 30 {
		t.Errorf("expected 30, got %.2f", pdc)
	}
}

func TestUpdateBaseline_NegativeKToday(t *testing.T) {
	b := algo.TalentBaseline{Alpha_lt: 100, Beta_lt: 5, Delta_lt: 0.97}
	_, err := algo.UpdateBaseline(b, -1)
	if err == nil {
		t.Error("expected error for negative k_today")
	}
}

func TestUpdateBaseline_NaNKToday(t *testing.T) {
	b := algo.TalentBaseline{Alpha_lt: 100, Beta_lt: 5, Delta_lt: 0.97}
	_, err := algo.UpdateBaseline(b, math.NaN())
	if err == nil {
		t.Error("expected error for NaN k_today")
	}
}

func TestBootstrap_ZeroMedian(t *testing.T) {
	cat := algo.CategoryBaseline{Category: "test", Median: 0}
	_, err := algo.BootstrapBaseline(cat, 0.97)
	if err == nil {
		t.Error("expected error for zero median")
	}
}

func TestBootstrap_InvalidDeltaLt(t *testing.T) {
	cat := algo.CategoryBaseline{Category: "test", Median: 5}
	_, err := algo.BootstrapBaseline(cat, 1.5)
	if err == nil {
		t.Error("expected error for delta_lt > 1")
	}
	_, err = algo.BootstrapBaseline(cat, 0)
	if err == nil {
		t.Error("expected error for delta_lt = 0")
	}
}

func TestCycleNPDC_NegativeClamp(t *testing.T) {
	pdc, err := algo.CycleNPDC(-5, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pdc != 0 {
		t.Errorf("expected 0 for negative stec_pdc, got %.2f", pdc)
	}
}

func TestCycleNPDC_MarketCapGate(t *testing.T) {
	pdc, err := algo.CycleNPDC(50, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pdc != 30 {
		t.Errorf("expected 30 (market cap), got %.2f", pdc)
	}
}

func TestCycleNPDC_NaN(t *testing.T) {
	_, err := algo.CycleNPDC(math.NaN(), 100)
	if err == nil {
		t.Error("expected error for NaN stec_pdc")
	}
}

func TestUpdateSigmaHist(t *testing.T) {
	b := algo.TalentBaseline{Talent_id: "t1"}
	updated, err := algo.UpdateSigmaHist(b, []float64{10, 12, 8, 11, 9})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Sigma_hist <= 0 {
		t.Error("sigma_hist should be positive for varied outputs")
	}
}

func TestUpdateSigmaHist_Empty(t *testing.T) {
	b := algo.TalentBaseline{Sigma_hist: 5}
	updated, err := algo.UpdateSigmaHist(b, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Sigma_hist != 5 {
		t.Error("empty outputs should preserve existing sigma_hist")
	}
}

func TestCycle1PDC_NaNLambda(t *testing.T) {
	b := algo.TalentBaseline{Lambda_lt: math.NaN()}
	_, err := algo.Cycle1PDC(b, 100)
	if err == nil {
		t.Error("expected error for NaN lambda_lt")
	}
}

package algo

// MatchDimensions holds the three component scores for MS_t computation.
// Each dimension is 0.0–1.0.
//
// DM (Demographic Match)    — advocate profile vs target audience category
// GP (Geographic Proximity) — advocate location vs campaign geography
// OH (Objective History)    — advocate track record on this campaign objective type
type MatchDimensions struct {
	DM float64 `json:"dm"`
	GP float64 `json:"gp"`
	OH float64 `json:"oh"`
}

// MatchScoreResult holds the composite MS_t and its components.
type MatchScoreResult struct {
	Dimensions MatchDimensions `json:"dimensions"`
	MS_t       float64         `json:"ms_t"`       // composite = (DM+GP+OH)/3, range 0.0–1.0
	Rank_score float64         `json:"rank_score"` // extended by Story 21 proximity boost at service layer
}

// DefaultAlpha is the platform default match weight factor (Story 18 A.6).
// Overridden by Global Settings (Story 19 Section 1).
const DefaultAlpha = 0.20

// ComputeMatchScore computes MS_t from the three match dimensions.
func ComputeMatchScore(d MatchDimensions) MatchScoreResult {
	ms := (d.DM + d.GP + d.OH) / 3.0
	if ms < 0 {
		ms = 0
	}
	if ms > 1 {
		ms = 1
	}
	return MatchScoreResult{
		Dimensions: d,
		MS_t:       ms,
		Rank_score: ms,
	}
}

// MatchAdjustment computes the cost multiplier applied to Cij in the Hungarian solver.
// A perfect match (MS_t=1.0, alpha=0.20) reduces Cij by 20%.
// A zero-match advocate has no reduction.
// Formula: 1 - (MS_t × alpha), floored at 0.60 (Story 18 A.6).
func MatchAdjustment(ms_t, alpha float64) float64 {
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 0.40 {
		alpha = 0.40
	}
	adj := 1.0 - (ms_t * alpha)
	if adj < 0.60 {
		adj = 0.60
	}
	return adj
}

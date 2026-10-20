package fetcher

import (
	"math"
	"testing"

	"fund-trace/internal/model"
)

func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}

func fundWithChange(change, previousNAV, alt float64) *model.RealTimeFund {
	return &model.RealTimeFund{
		Code:           "020602",
		EstimatedNAV:   previousNAV,
		PreviousNAV:    previousNAV,
		DailyChangePct: change,
		AltChangePct:   alt,
		Available:      true,
	}
}

func indexMeta(code, secid string, weight float64) model.FundMeta {
	return model.FundMeta{
		Code:        code,
		IndexCode:   "H30269",
		IndexSecID:  secid,
		IndexWeight: weight,
		DeriveOK:    true,
	}
}

// The captured 11:29 payload: Sina reported +0.0341% while the benchmark index
// was down 1.37%. The stale value must be replaced by the anchored estimate.
func TestApplyIndexAnchor_StaleUpstreamReplaced(t *testing.T) {
	funds := map[string]*model.RealTimeFund{
		"020602": fundWithChange(0.0341, 1.1185, -1.3633),
	}
	metas := map[string]model.FundMeta{
		"020602": indexMeta("020602", "2.H30269", 0.95),
	}
	changes := map[string]float64{"2.H30269": -1.37}

	if n := ApplyIndexAnchor(funds, metas, changes, DefaultAnchorTolerance); n != 1 {
		t.Fatalf("adjusted = %d, want 1", n)
	}

	f := funds["020602"]
	if !f.Derived {
		t.Error("Derived = false, want true")
	}
	if want := -1.3015; !approxEqual(f.DailyChangePct, want, 0.0001) {
		t.Errorf("DailyChangePct = %v, want %v", f.DailyChangePct, want)
	}
	if want := 1.1039; !approxEqual(f.EstimatedNAV, want, 0.0001) {
		t.Errorf("EstimatedNAV = %v, want %v", f.EstimatedNAV, want)
	}
}

// The 11:33 payload already agreed with the index; it must be left untouched.
func TestApplyIndexAnchor_ConsistentUpstreamKept(t *testing.T) {
	funds := map[string]*model.RealTimeFund{
		"020602": fundWithChange(-1.272, 1.1185, -1.3633),
	}
	metas := map[string]model.FundMeta{
		"020602": indexMeta("020602", "2.H30269", 0.95),
	}
	changes := map[string]float64{"2.H30269": -1.37}

	if n := ApplyIndexAnchor(funds, metas, changes, DefaultAnchorTolerance); n != 0 {
		t.Fatalf("adjusted = %d, want 0", n)
	}
	f := funds["020602"]
	if f.Derived {
		t.Error("Derived = true, want false for a consistent row")
	}
	if f.DailyChangePct != -1.272 {
		t.Errorf("DailyChangePct = %v, want -1.272", f.DailyChangePct)
	}
}

// Active funds have no benchmark anchor: a material disagreement between the
// two upstream pairs is flagged instead of corrected.
func TestApplyIndexAnchor_ActiveFundFlaggedUncertain(t *testing.T) {
	funds := map[string]*model.RealTimeFund{
		"018300": fundWithChange(2.3774, 1.5684, 0.5775),
	}
	metas := map[string]model.FundMeta{
		"018300": {Code: "018300"},
	}

	if n := ApplyIndexAnchor(funds, metas, nil, DefaultAnchorTolerance); n != 0 {
		t.Fatalf("adjusted = %d, want 0", n)
	}
	if !funds["018300"].Uncertain {
		t.Error("Uncertain = false, want true")
	}
	if funds["018300"].Derived {
		t.Error("Derived = true, want false")
	}
}

// QDII funds carry a benchmark but must not be overridden: FX settlement and a
// one-day NAV lag distort the index relationship.
func TestApplyIndexAnchor_QDIIUntouched(t *testing.T) {
	funds := map[string]*model.RealTimeFund{
		"012349": fundWithChange(-0.3594, 0.5515, -0.8333),
	}
	funds["012349"].Code = "012349"
	metas := map[string]model.FundMeta{
		"012349": {Code: "012349", IndexCode: "HSTECH", IndexSecID: "124.HSTECH", IndexWeight: 0.95, DeriveOK: false},
	}
	changes := map[string]float64{"124.HSTECH": -1.21}

	if n := ApplyIndexAnchor(funds, metas, changes, DefaultAnchorTolerance); n != 0 {
		t.Fatalf("adjusted = %d, want 0", n)
	}
	if f := funds["012349"]; f.Derived || f.Uncertain || f.DailyChangePct != -0.3594 {
		t.Errorf("QDII row must stay untouched, got %+v", f)
	}
}

// When the index quote request fails there is nothing to anchor against.
func TestApplyIndexAnchor_MissingIndexQuote(t *testing.T) {
	funds := map[string]*model.RealTimeFund{
		"020602": fundWithChange(0.0341, 1.1185, -1.3633),
	}
	metas := map[string]model.FundMeta{
		"020602": indexMeta("020602", "2.H30269", 0.95),
	}

	if n := ApplyIndexAnchor(funds, metas, nil, DefaultAnchorTolerance); n != 0 {
		t.Fatalf("adjusted = %d, want 0", n)
	}
	if funds["020602"].Derived {
		t.Error("Derived = true, want false without an index quote")
	}
}

func TestApplyIndexAnchor_UnavailableFundSkipped(t *testing.T) {
	funds := map[string]*model.RealTimeFund{
		"020602": {Code: "020602", Available: false},
	}
	metas := map[string]model.FundMeta{
		"020602": indexMeta("020602", "2.H30269", 0.95),
	}
	changes := map[string]float64{"2.H30269": -1.37}

	if n := ApplyIndexAnchor(funds, metas, changes, DefaultAnchorTolerance); n != 0 {
		t.Fatalf("adjusted = %d, want 0", n)
	}
	if funds["020602"].Derived {
		t.Error("unavailable fund must not be adjusted")
	}
}

func TestApplyIndexAnchor_WeightFallback(t *testing.T) {
	funds := map[string]*model.RealTimeFund{
		"020602": fundWithChange(0, 1.0, 0),
	}
	// Zero weight must fall back to the 95% default instead of zeroing rows.
	metas := map[string]model.FundMeta{
		"020602": {Code: "020602", IndexCode: "H30269", IndexSecID: "2.H30269", DeriveOK: true},
	}
	changes := map[string]float64{"2.H30269": -1.0}

	ApplyIndexAnchor(funds, metas, changes, DefaultAnchorTolerance)
	if want := -0.95; !approxEqual(funds["020602"].DailyChangePct, want, 0.0001) {
		t.Errorf("DailyChangePct = %v, want %v (95%% default weight)", funds["020602"].DailyChangePct, want)
	}
}

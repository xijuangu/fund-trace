package fetcher

import (
	"math"

	"fund-trace/internal/model"
)

const (
	// DefaultAnchorTolerance is how far (in percentage points) the upstream
	// change may deviate from the index-implied change before it is treated as
	// stale and replaced by the anchored value.
	DefaultAnchorTolerance = 0.35

	// DefaultPairTolerance is how far the two upstream valuation pairs may
	// drift apart before a fund without a benchmark index is flagged uncertain.
	DefaultPairTolerance = 0.50
)

// ApplyIndexAnchor reconciles each fund's real-time estimate with the move of
// its benchmark index.
//
// Sina serves two valuation pairs that update on different schedules, and
// either one can freeze near the previous NAV while the market is open. The
// benchmark index provides an independent reference:
//
//	implied = index change × benchmark weight (usually 95%)
//
// When the upstream change deviates from the implied change by more than
// tolerance, the upstream value is replaced by the anchored estimate and the
// row is marked Derived. Funds without a usable anchor are left untouched;
// active funds (no benchmark) are marked Uncertain when the two upstream pairs
// disagree materially, since there is no way to tell which one is stale.
//
// It mutates funds in place and returns the number of adjusted rows.
func ApplyIndexAnchor(
	funds map[string]*model.RealTimeFund,
	metas map[string]model.FundMeta,
	indexChanges map[string]float64,
	tolerance float64,
) int {
	if tolerance <= 0 {
		tolerance = DefaultAnchorTolerance
	}

	adjusted := 0
	for code, fund := range funds {
		if fund == nil || !fund.Available {
			continue
		}

		meta, hasMeta := metas[code]
		if hasMeta && meta.DeriveOK && meta.IndexSecID != "" {
			if indexChange, ok := indexChanges[meta.IndexSecID]; ok {
				weight := meta.IndexWeight
				if weight <= 0 || weight > 1 {
					weight = DefaultIndexWeight
				}
				implied := indexChange * weight
				if math.Abs(fund.DailyChangePct-implied) > tolerance {
					fund.DailyChangePct = roundTo(implied, 4)
					if fund.PreviousNAV > 0 {
						fund.EstimatedNAV = roundTo(fund.PreviousNAV*(1+implied/100), 4)
					}
					fund.Derived = true
					adjusted++
				}
				continue
			}
		}

		// No usable anchor: flag a material disagreement between the two
		// upstream pairs, but only for active funds. QDII and commodity funds
		// legitimately diverge from the index-level estimate.
		if hasMeta && meta.IndexCode != "" {
			continue
		}
		if fund.AltChangePct != 0 {
			expected := fund.AltChangePct * DefaultIndexWeight
			if math.Abs(fund.DailyChangePct-expected) > DefaultPairTolerance {
				fund.Uncertain = true
			}
		}
	}
	return adjusted
}

func roundTo(v float64, digits int) float64 {
	pow := math.Pow(10, float64(digits))
	return math.Round(v*pow) / pow
}

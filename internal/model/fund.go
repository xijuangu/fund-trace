package model

import "time"

type FundType int

const (
	FundMixed FundType = iota
	FundStock
	FundBond
	FundIndex
	FundUnknown
)

type Fund struct {
	Code    string
	Name    string
	Type    FundType
	AddedAt time.Time
}

// RealTimeFund is one fund's intraday estimate.
//
// Sina's fu_ payload carries two independent valuation pairs that are updated
// on separate schedules:
//
//	pair 1: EstimatedNAV / DailyChangePct  (fund's own estimate)
//	pair 2: index-based estimate (AltChangePct)
//
// Either pair can go stale while the market is open, so each value is
// reconciled against the fund's benchmark index (see ApplyIndexAnchor) before
// it is shown. Derived marks rows whose DailyChangePct was replaced by the
// index-anchored estimate; Uncertain marks rows where the two upstream pairs
// disagree and no index anchor was available to break the tie.
type RealTimeFund struct {
	Code           string
	Name           string
	EstimatedNAV   float64 // gsz
	PreviousNAV    float64 // dwjz
	NAVDate        string  // jzrq
	DailyChangePct float64 // gszzl (already percent)
	UpdateTime     string  // gztime
	Available      bool    // false when real-time data not available

	AltChangePct float64 // second valuation pair (fields 8/9), already percent
	Derived      bool    // DailyChangePct replaced by the index-anchored estimate
	Uncertain    bool    // upstream pairs disagree and no anchor was available
}

// FundMeta describes how a fund tracks its benchmark index.
// It is cached in the funds table so the lookup only happens once per fund.
type FundMeta struct {
	Code        string
	IndexCode   string  // benchmark index code, e.g. "H30269"; empty for active funds
	IndexSecID  string  // East Money secid, e.g. "2.H30269"
	IndexWeight float64 // benchmark weight of the index, e.g. 0.95
	// DeriveOK reports whether the index-anchored estimate is trustworthy
	// enough to replace the upstream value. QDII and commodity funds are
	// excluded because FX settlement and NAV lag distort the relationship.
	DeriveOK bool
}

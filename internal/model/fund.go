package model

import "time"

type FundType int

const (
	FundMixed  FundType = iota
	FundStock
	FundBond
	FundIndex
	FundUnknown
)

type Fund struct {
	Code   string
	Name   string
	Type   FundType
	AddedAt time.Time
}

type RealTimeFund struct {
	Code           string
	Name           string
	EstimatedNAV   float64 // gsz
	PreviousNAV    float64 // dwjz
	NAVDate        string  // jzrq
	DailyChangePct float64 // gszzl (already percent)
	UpdateTime     string  // gztime
	Available      bool    // false when real-time data not available
}

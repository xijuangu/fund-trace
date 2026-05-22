package model

import "time"

type NavSnapshot struct {
	FundCode       string
	Date           string  // FSRQ YYYY-MM-DD
	UnitNAV        float64 // DWJZ
	AccumulatedNAV float64 // LJJZ
	DailyGrowthPct float64 // JZZZL
	RecordedAt     time.Time
}

type DailySummary struct {
	Date      string
	FundCode  string
	NAV       float64
	ChangePct float64
	Note      string
}

type PriceSnapshot struct {
	Kind       AssetKind
	Market     string
	Code       string
	Date       string // YYYY-MM-DD
	Open       float64
	High       float64
	Low        float64
	Close      float64
	Volume     float64
	Amount     float64
	ChangePct  float64
	RecordedAt time.Time
}

// FundListEntry represents one fund from eastmoney fundcode_search.js
type FundListEntry struct {
	Code       string
	Pinyin     string
	Name       string
	TypeName   string // e.g. "混合型-灵活", "指数型-股票"
	FullPinyin string
}

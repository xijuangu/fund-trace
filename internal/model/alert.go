package model

import "database/sql"

type AlertType int

const (
	AlertDrop AlertType = iota
	AlertRise
)

type Alert struct {
	ID              int64
	FundCode        string        // kept for backward compat
	Kind            AssetKind     // AssetKindFund or AssetKindStock
	Market          string        // empty for funds, "sh"/"sz" for stocks
	Code            string        // fund code or stock code
	Type            AlertType
	ThresholdPct    float64       // negative = drop alert, positive = rise alert
	Enabled         bool
	LastTriggeredAt sql.NullTime
}

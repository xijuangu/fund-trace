package model

import "database/sql"

type AlertType int

const (
	AlertDrop AlertType = iota
	AlertRise
)

type Alert struct {
	ID              int64
	FundCode        string
	Type            AlertType
	ThresholdPct    float64       // negative = drop alert, positive = rise alert
	Enabled         bool
	LastTriggeredAt sql.NullTime
}

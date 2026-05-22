package model

import "time"

// AssetKind distinguishes between fund and stock assets.
type AssetKind int

const (
	AssetKindFund  AssetKind = iota // 0
	AssetKindStock                  // 1
)

// String returns a human-readable label for the asset kind.
func (k AssetKind) String() string {
	switch k {
	case AssetKindFund:
		return "Fund"
	case AssetKindStock:
		return "Stock"
	default:
		return "Unknown"
	}
}

// Asset is the persistent representation of a tracked asset (fund or stock).
type Asset struct {
	ID      int64
	Kind    AssetKind
	Market  string // empty for funds; "sh" or "sz" for A-share stocks
	Code    string
	Name    string
	Type    int // FundType for funds, 0 for stocks
	AddedAt time.Time
}

// Quote is the ephemeral (fetched) real-time data for an asset.
// For funds: Value = EstimatedNAV, Previous = PreviousNAV.
// For stocks: Value = CurrentPrice, Previous = PreviousClose.
type Quote struct {
	Kind       AssetKind
	Market     string
	Code       string
	Name       string
	Value      float64 // current price or estimated NAV
	Previous   float64 // previous close or previous NAV
	ChangePct  float64 // daily change percent
	UpdateTime string
	Available  bool
}

// InferStockMarket guesses the market from a 6-digit A-share stock code.
// Rules: codes starting with "6" → "sh", "0" or "3" → "sz".
// Codes starting with "4" or "8" are Beijing Stock Exchange → returns error.
// Returns ("", error) if the code cannot be inferred.
func InferStockMarket(code string) (string, error) {
	if len(code) != 6 {
		return "", &MarketError{Code: code, Reason: "stock code must be 6 digits"}
	}
	switch code[0] {
	case '6':
		return "sh", nil
	case '0', '3':
		return "sz", nil
	case '4', '8':
		return "", &MarketError{Code: code, Reason: "Beijing Stock Exchange not yet supported"}
	default:
		return "", &MarketError{Code: code, Reason: "cannot infer market from code prefix"}
	}
}

// MarketError is returned when stock market inference fails.
type MarketError struct {
	Code   string
	Reason string
}

func (e *MarketError) Error() string {
	return "stock code " + e.Code + ": " + e.Reason
}

// QuoteKey builds a unique lookup key for a quote: "kind:market:code".
func QuoteKey(kind AssetKind, market, code string) string {
	if market == "" {
		return "fund:" + code
	}
	return "stock:" + market + ":" + code
}

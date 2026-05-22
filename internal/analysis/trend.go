package analysis

import (
	"math"

	"fund-trace/internal/model"
)

// SMA calculates Simple Moving Average for a given period.
// Returns a slice of same length as values, with NaN for positions before period-1.
func SMA(values []float64, period int) []float64 {
	if period <= 0 {
		return nil
	}
	if len(values) == 0 {
		return []float64{}
	}
	result := make([]float64, len(values))
	for i := range result {
		result[i] = math.NaN()
	}
	if period > len(values) {
		return result
	}
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	result[period-1] = sum / float64(period)
	for i := period; i < len(values); i++ {
		sum += values[i] - values[i-period]
		result[i] = sum / float64(period)
	}
	return result
}

// EMA calculates Exponential Moving Average.
// Uses period*2/(period+1) as the smoothing factor.
func EMA(values []float64, period int) []float64 {
	if period <= 0 {
		return nil
	}
	if len(values) == 0 {
		return []float64{}
	}
	result := make([]float64, len(values))
	for i := range result {
		result[i] = math.NaN()
	}
	if period > len(values) {
		return result
	}
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	result[period-1] = sum / float64(period)
	multiplier := 2.0 / float64(period+1)
	for i := period; i < len(values); i++ {
		result[i] = (values[i]-result[i-1])*multiplier + result[i-1]
	}
	return result
}

// RSI calculates Relative Strength Index for a given period (typically 14).
// Returns NaN for positions before 'period'.
func RSI(values []float64, period int) ([]float64, error) {
	if period <= 0 {
		return nil, nil
	}
	if len(values) <= period {
		result := make([]float64, len(values))
		for i := range result {
			result[i] = math.NaN()
		}
		return result, nil
	}

	result := make([]float64, len(values))
	for i := range result {
		result[i] = math.NaN()
	}

	var avgGain, avgLoss float64
	for i := 1; i <= period; i++ {
		change := values[i] - values[i-1]
		if change > 0 {
			avgGain += change
		} else {
			avgLoss += -change
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	if avgLoss == 0 {
		result[period] = 100.0
	} else {
		rs := avgGain / avgLoss
		result[period] = 100.0 - (100.0 / (1.0 + rs))
	}

	for i := period + 1; i < len(values); i++ {
		change := values[i] - values[i-1]
		var gain, loss float64
		if change > 0 {
			gain = change
		} else {
			loss = -change
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)

		if avgLoss == 0 {
			result[i] = 100.0
		} else {
			rs := avgGain / avgLoss
			result[i] = 100.0 - (100.0 / (1.0 + rs))
		}
	}
	return result, nil
}

// TrendResult holds analysis summary for a fund.
type TrendResult struct {
	SMA5      []float64
	SMA10     []float64
	SMA20     []float64
	RSI14     []float64
	Direction string  // "up", "down", "sideways"
	Change5D  float64 // 5-day NAV change percentage
}

// TrendSummaryFromValues computes all indicators from a slice of float64 values
// (e.g. NAV series, stock close prices). Values must be in chronological order.
func TrendSummaryFromValues(values []float64) TrendResult {
	if len(values) == 0 {
		return TrendResult{Direction: "sideways"}
	}

	tr := TrendResult{
		SMA5:  SMA(values, 5),
		SMA10: SMA(values, 10),
		SMA20: SMA(values, 20),
	}
	rsi14, _ := RSI(values, 14)
	tr.RSI14 = rsi14

	if len(tr.SMA5) >= 20 && len(tr.SMA20) >= 20 && !math.IsNaN(tr.SMA5[len(tr.SMA5)-1]) && !math.IsNaN(tr.SMA20[len(tr.SMA20)-1]) {
		if tr.SMA5[len(tr.SMA5)-1] > tr.SMA20[len(tr.SMA20)-1]*1.01 {
			tr.Direction = "up"
		} else if tr.SMA5[len(tr.SMA5)-1] < tr.SMA20[len(tr.SMA20)-1]*0.99 {
			tr.Direction = "down"
		} else {
			tr.Direction = "sideways"
		}
	} else {
		tr.Direction = "sideways"
	}

	if len(values) >= 5 {
		tr.Change5D = ((values[len(values)-1] - values[len(values)-5]) / values[len(values)-5]) * 100
	}

	return tr
}

// TrendSummary computes all indicators from NAV snapshots.
// Snapshots must be in chronological order (oldest first).
func TrendSummary(snapshots []model.NavSnapshot) TrendResult {
	if len(snapshots) == 0 {
		return TrendResult{Direction: "sideways"}
	}
	navs := make([]float64, len(snapshots))
	for i, s := range snapshots {
		navs[i] = s.UnitNAV
	}
	return TrendSummaryFromValues(navs)
}

// Latest returns the last non-NaN value from a float64 slice, or NaN if none.
func Latest(values []float64) float64 {
	for i := len(values) - 1; i >= 0; i-- {
		if !math.IsNaN(values[i]) {
			return values[i]
		}
	}
	return math.NaN()
}

package analysis

import (
	"math"
	"testing"
	"time"

	"fund-trace/internal/model"
)

// approxEqual checks if two float64 values are within tolerance.
func approxEqual(a, b, tol float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	if math.IsNaN(a) || math.IsNaN(b) {
		return false
	}
	return math.Abs(a-b) <= tol
}

func TestSMA(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		period int
		want   []float64
	}{
		{
			name:   "period 3 on five values",
			values: []float64{1, 2, 3, 4, 5},
			period: 3,
			want:   []float64{math.NaN(), math.NaN(), 2.0, 3.0, 4.0},
		},
		{
			name:   "period 2 on four values",
			values: []float64{2, 4, 6, 8},
			period: 2,
			want:   []float64{math.NaN(), 3.0, 5.0, 7.0},
		},
		{
			name:   "period larger than values",
			values: []float64{1, 2, 3},
			period: 10,
			want:   []float64{math.NaN(), math.NaN(), math.NaN()},
		},
		{
			name:   "period equals length",
			values: []float64{10, 20, 30},
			period: 3,
			want:   []float64{math.NaN(), math.NaN(), 20.0},
		},
		{
			name:   "empty values",
			values: []float64{},
			period: 3,
			want:   []float64{},
		},
		{
			name:   "period zero",
			values: []float64{1, 2, 3},
			period: 0,
			want:   []float64{},
		},
		{
			name:   "period negative",
			values: []float64{1, 2, 3},
			period: -1,
			want:   []float64{},
		},
		{
			name:   "single value period 1",
			values: []float64{42},
			period: 1,
			want:   []float64{42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SMA(tt.values, tt.period)
			if len(got) != len(tt.want) {
				t.Fatalf("SMA() length = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if !approxEqual(got[i], tt.want[i], 1e-9) {
					t.Errorf("SMA() at index %d = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestEMA(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		period int
		want   []float64
	}{
		{
			name:   "period 3 on five values (linear data)",
			values: []float64{1, 2, 3, 4, 5},
			period: 3,
			// SMA(1,2,3)=2, EMA[3]=(4-2)*0.5+2=3, EMA[4]=(5-3)*0.5+3=4
			want: []float64{math.NaN(), math.NaN(), 2.0, 3.0, 4.0},
		},
		{
			name:   "period 2 on four values",
			values: []float64{2, 4, 6, 8},
			period: 2,
			// SMA(2,4)=3, EMA[2]=(6-3)*2/3+3=5, EMA[3]=(8-5)*2/3+5=7
			want: []float64{math.NaN(), 3.0, 5.0, 7.0},
		},
		{
			name:   "period larger than values",
			values: []float64{1, 2},
			period: 5,
			want:   []float64{math.NaN(), math.NaN()},
		},
		{
			name:   "empty values",
			values: []float64{},
			period: 3,
			want:   []float64{},
		},
		{
			name:   "period zero",
			values: []float64{1, 2, 3},
			period: 0,
			want:   []float64{},
		},
		{
			name:   "single value period 1",
			values: []float64{42},
			period: 1,
			want:   []float64{42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EMA(tt.values, tt.period)
			if len(got) != len(tt.want) {
				t.Fatalf("EMA() length = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if !approxEqual(got[i], tt.want[i], 1e-9) {
					t.Errorf("EMA() at index %d = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRSI(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		period int
		want   []float64
	}{
		{
			name:   "all rising values period 14",
			values: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			period: 14,
			want: []float64{
				math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(),
				math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(),
				math.NaN(), math.NaN(), math.NaN(), math.NaN(), 100.0,
				100.0, 100.0, 100.0, 100.0, 100.0,
			},
		},
		{
			name:   "all falling values period 14",
			values: []float64{20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
			period: 14,
			want: []float64{
				math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(),
				math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(),
				math.NaN(), math.NaN(), math.NaN(), math.NaN(), 0.0,
				0.0, 0.0, 0.0, 0.0, 0.0,
			},
		},
		{
			name:   "period larger than values",
			values: []float64{1, 2, 3},
			period: 10,
			want:   []float64{math.NaN(), math.NaN(), math.NaN()},
		},
		{
			name:   "period equals length",
			values: []float64{1, 2, 3},
			period: 3,
			want:   []float64{math.NaN(), math.NaN(), math.NaN()},
		},
		{
			name:   "period zero returns nil",
			values: []float64{1, 2, 3},
			period: 0,
			want:   nil,
		},
		{
			name:   "period negative returns nil",
			values: []float64{1, 2, 3},
			period: -1,
			want:   nil,
		},
		{
			name:   "empty values",
			values: []float64{},
			period: 14,
			want:   []float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RSI(tt.values, tt.period)
			if err != nil {
				t.Fatalf("RSI() unexpected error: %v", err)
			}
			if tt.want == nil {
				if got != nil {
					t.Errorf("RSI() = %v, want nil", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("RSI() length = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if !approxEqual(got[i], tt.want[i], 1e-9) {
					t.Errorf("RSI() at index %d = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestTrendSummary(t *testing.T) {
	t.Run("upward trend", func(t *testing.T) {
		snapshots := make([]model.NavSnapshot, 30)
		for i := range snapshots {
			snapshots[i] = model.NavSnapshot{
				FundCode:       "000001",
				Date:           time.Date(2025, 1, 1+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02"),
				UnitNAV:        1.0 + float64(i)*0.02,
				AccumulatedNAV: 1.0 + float64(i)*0.02,
			}
		}

		result := TrendSummary(snapshots)

		if result.Direction != "up" {
			t.Errorf("TrendSummary() Direction = %q, want %q", result.Direction, "up")
		}

		if len(result.SMA5) != 30 {
			t.Errorf("TrendSummary() SMA5 length = %d, want 30", len(result.SMA5))
		}
		if len(result.SMA10) != 30 {
			t.Errorf("TrendSummary() SMA10 length = %d, want 30", len(result.SMA10))
		}
		if len(result.SMA20) != 30 {
			t.Errorf("TrendSummary() SMA20 length = %d, want 30", len(result.SMA20))
		}
		if len(result.RSI14) != 30 {
			t.Errorf("TrendSummary() RSI14 length = %d, want 30", len(result.RSI14))
		}

		// Verify last SMA5 > last SMA20
		if result.SMA5[29] <= result.SMA20[29] {
			t.Errorf("TrendSummary() SMA5[29]=%v should be > SMA20[29]=%v for uptrend", result.SMA5[29], result.SMA20[29])
		}

		// Verify Change5D: nav=1.58 at idx29, nav=1.50 at idx25
		expectedChange := ((1.58 - 1.50) / 1.50) * 100
		if !approxEqual(result.Change5D, expectedChange, 1e-9) {
			t.Errorf("TrendSummary() Change5D = %v, want %v", result.Change5D, expectedChange)
		}

		// Verify RSI14 is 100.0 for consistently rising values
		if !approxEqual(result.RSI14[29], 100.0, 1e-9) {
			t.Errorf("TrendSummary() RSI14[29] = %v, want 100.0", result.RSI14[29])
		}
	})

	t.Run("empty snapshots", func(t *testing.T) {
		result := TrendSummary(nil)

		if result.Direction != "sideways" {
			t.Errorf("TrendSummary() Direction = %q, want %q for empty input", result.Direction, "sideways")
		}
		if result.Change5D != 0 {
			t.Errorf("TrendSummary() Change5D = %v, want 0 for empty input", result.Change5D)
		}
	})

	t.Run("fewer than 20 snapshots", func(t *testing.T) {
		snapshots := make([]model.NavSnapshot, 10)
		for i := range snapshots {
			snapshots[i] = model.NavSnapshot{
				FundCode: "000001",
				Date:     time.Date(2025, 1, 1+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02"),
				UnitNAV:  1.0 + float64(i)*0.05,
			}
		}

		result := TrendSummary(snapshots)

		// Without 20+ points, direction defaults to sideways
		if result.Direction != "sideways" {
			t.Errorf("TrendSummary() Direction = %q, want %q for <20 snapshots", result.Direction, "sideways")
		}

		// Change5D should still compute if >=5
		expectedChange := ((1.45 - 1.25) / 1.25) * 100
		if !approxEqual(result.Change5D, expectedChange, 1e-9) {
			t.Errorf("TrendSummary() Change5D = %v, want %v", result.Change5D, expectedChange)
		}
	})
}

func TestLatest(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
	}{
		{
			name:   "last non-NaN in middle",
			values: []float64{math.NaN(), math.NaN(), 3.0, math.NaN()},
			want:   3.0,
		},
		{
			name:   "all NaN",
			values: []float64{math.NaN(), math.NaN(), math.NaN()},
			want:   math.NaN(),
		},
		{
			name:   "single value",
			values: []float64{42.5},
			want:   42.5,
		},
		{
			name:   "last value is non-NaN",
			values: []float64{math.NaN(), math.NaN(), 5.0},
			want:   5.0,
		},
		{
			name:   "empty slice",
			values: []float64{},
			want:   math.NaN(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Latest(tt.values)
			if !approxEqual(got, tt.want, 1e-9) {
				t.Errorf("Latest() = %v, want %v", got, tt.want)
			}
		})
	}
}

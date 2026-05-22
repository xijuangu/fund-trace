package tui

import "math"

var blockChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// SparkBlock pairs a rendered character with the bucket's average value.
type SparkBlock struct {
	Char  rune
	Value float64
}

// Sparkline returns blocks where Char encodes the relative height and Value
// is the bucket's average daily change %, for per-block coloring.
func Sparkline(values []float64, width int) []SparkBlock {
	if len(values) == 0 || width <= 0 {
		return nil
	}
	if width > len(values) {
		width = len(values)
	}

	maxAbs := maxAbs(values)
	if maxAbs == 0 {
		blocks := make([]SparkBlock, width)
		for i := range blocks {
			blocks[i] = SparkBlock{Char: '▄', Value: 0}
		}
		return blocks
	}

	rangeSize := 2 * maxAbs
	bucketSize := float64(len(values)) / float64(width)
	blocks := make([]SparkBlock, width)

	for i := 0; i < width; i++ {
		start := int(float64(i) * bucketSize)
		end := int(float64(i+1) * bucketSize)
		if end > len(values) {
			end = len(values)
		}
		if start >= end {
			start = end - 1
		}

		sum := 0.0
		for j := start; j < end; j++ {
			sum += values[j]
		}
		avg := sum / float64(end - start)

		normalized := (avg + maxAbs) / rangeSize
		idx := int(normalized * float64(len(blockChars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blockChars) {
			idx = len(blockChars) - 1
		}
		blocks[i] = SparkBlock{Char: blockChars[idx], Value: avg}
	}

	return blocks
}

func maxAbs(values []float64) float64 {
	m := 0.0
	for _, v := range values {
		if math.Abs(v) > m {
			m = math.Abs(v)
		}
	}
	return m
}

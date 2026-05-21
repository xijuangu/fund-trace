package tui

import (
	"math"
)

// blockChars are Unicode block elements from lowest to highest density.
var blockChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Sparkline renders a miniature trend line using Unicode block characters.
// values should be a series of numeric data points (e.g., historical NAVs).
// width is the desired number of output characters.
// Returns an empty string if values is empty or width <= 0.
func Sparkline(values []float64, width int) string {
	if len(values) == 0 || width <= 0 {
		return ""
	}

	// Never request more buckets than we have data points.
	if width > len(values) {
		width = len(values)
	}

	min, max := minMax(values)

	// Flat line edge case — all values identical.
	if max == min {
		flat := make([]rune, width)
		for i := range flat {
			flat[i] = '─'
		}
		return string(flat)
	}

	// Downsample into 'width' equal-width buckets, averaging each bucket.
	bucketSize := float64(len(values)) / float64(width)
	result := make([]rune, width)

	for i := 0; i < width; i++ {
		start := int(float64(i) * bucketSize)
		end := int(float64(i+1) * bucketSize)
		if end > len(values) {
			end = len(values)
		}
		if start >= end {
			start = end - 1
		}

		// Average the values in this bucket.
		sum := 0.0
		for j := start; j < end; j++ {
			sum += values[j]
		}
		avg := sum / float64(end-start)

		// Normalize to [0, 1] and pick a block character.
		normalized := (avg - min) / (max - min)
		idx := int(normalized * float64(len(blockChars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blockChars) {
			idx = len(blockChars) - 1
		}
		result[i] = blockChars[idx]
	}

	return string(result)
}

// minMax returns the minimum and maximum values in a slice.
func minMax(values []float64) (float64, float64) {
	min := math.MaxFloat64
	max := -math.MaxFloat64
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

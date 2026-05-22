package tui

import "math"

var blockChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func Sparkline(values []float64, width int) string {
	if len(values) == 0 || width <= 0 {
		return ""
	}
	if width > len(values) {
		width = len(values)
	}

	maxAbs := maxAbs(values)
	if maxAbs == 0 {
		flat := make([]rune, width)
		for i := range flat {
			flat[i] = '▄'
		}
		return string(flat)
	}

	rangeSize := 2 * maxAbs
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

		sum := 0.0
		for j := start; j < end; j++ {
			sum += values[j]
		}
		avg := sum / float64(end-start)

		normalized := (avg + maxAbs) / rangeSize
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

func maxAbs(values []float64) float64 {
	m := 0.0
	for _, v := range values {
		if math.Abs(v) > m {
			m = math.Abs(v)
		}
	}
	return m
}

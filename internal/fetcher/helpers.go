package fetcher

import (
	"strconv"
	"strings"
)

func parseFloatSafe(s string) float64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func padLeft(s string, n int, pad string) string {
	if len(s) >= n {
		return s
	}
	return strings.Repeat(pad, n-len(s)) + s
}

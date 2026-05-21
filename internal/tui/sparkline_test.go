package tui

import (
	"strings"
	"testing"
)

func TestSparkline_Empty(t *testing.T) {
	if s := Sparkline(nil, 10); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
	if s := Sparkline([]float64{}, 10); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}

func TestSparkline_ZeroWidth(t *testing.T) {
	if s := Sparkline([]float64{1, 2, 3}, 0); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}

func TestSparkline_FlatLine(t *testing.T) {
	s := Sparkline([]float64{5, 5, 5, 5, 5}, 5)
	if !strings.ContainsRune(s, '─') {
		t.Errorf("expected flat line dashes, got %q", s)
	}
}

func TestSparkline_FewerValuesThanWidth(t *testing.T) {
	// This was the crash case: 5 values, width 10 caused index -1.
	s := Sparkline([]float64{1.0, 1.1, 1.2, 1.3, 1.4}, 10)
	runes := []rune(s)
	if len(runes) != 5 {
		t.Errorf("expected width capped to 5, got %d", len(runes))
	}
	for _, r := range runes {
		if !strings.ContainsRune(string(blockChars), r) {
			t.Errorf("unexpected rune %c in %q", r, s)
		}
	}
}

func TestSparkline_Normal(t *testing.T) {
	vals := []float64{1.0, 1.1, 1.2, 1.15, 1.3, 1.25, 1.4, 1.35, 1.5, 1.6}
	s := Sparkline(vals, 5)
	if len([]rune(s)) != 5 {
		t.Errorf("expected width 5, got %d", len([]rune(s)))
	}
}

func TestSparkline_SingleValue(t *testing.T) {
	s := Sparkline([]float64{3.14}, 3)
	if len([]rune(s)) != 1 {
		t.Errorf("expected width capped to 1, got %d", len([]rune(s)))
	}
}

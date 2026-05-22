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

func TestSparkline_FewerValuesThanWidth(t *testing.T) {
	s := Sparkline([]float64{1.0, 1.1, 1.2, 1.3, 1.4}, 10)
	runes := []rune(s)
	if len(runes) != 5 {
		t.Errorf("expected width capped to 5, got %d", len(runes))
	}
}

func TestSparkline_SingleValue(t *testing.T) {
	s := Sparkline([]float64{3.14}, 3)
	if len([]rune(s)) != 1 {
		t.Errorf("expected width capped to 1, got %d", len([]rune(s)))
	}
}

func TestSparkline_AllZero(t *testing.T) {
	s := Sparkline([]float64{0, 0, 0, 0, 0}, 3)
	if !strings.Contains(s, "▄") {
		t.Errorf("all-zero should produce ▄ blocks, got %q", s)
	}
}

func TestSparkline_ZeroMapsToMid(t *testing.T) {
	values := []float64{-5, -2.5, 0, 2.5, 5}
	s := Sparkline(values, 5)
	runes := []rune(s)
	if len(runes) != 5 {
		t.Fatalf("expected 5 runes, got %d", len(runes))
	}
	if runes[2] != '▄' {
		t.Errorf("0%% should be ▄, got %c (%q)", runes[2], s)
	}
}

func TestSparkline_AllPositiveAboveMid(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}
	s := Sparkline(values, 5)
	for _, r := range s {
		if r == '▃' || r == '▂' || r == '▁' {
			t.Errorf("all-positive should stay >= ▄, got %c in %q", r, s)
		}
	}
}

func TestSparkline_AllNegativeBelowMid(t *testing.T) {
	values := []float64{-6, -3, -4, -2, -5}
	s := Sparkline(values, 5)
	for _, r := range s {
		if r == '▅' || r == '▆' || r == '▇' || r == '█' {
			t.Errorf("all-negative should stay <= ▄, got %c in %q", r, s)
		}
	}
}

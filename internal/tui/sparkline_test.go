package tui

import "testing"

func TestSparkline_Empty(t *testing.T) {
	if b := Sparkline(nil, 10); b != nil {
		t.Errorf("expected nil, got %v", b)
	}
	if b := Sparkline([]float64{}, 10); b != nil {
		t.Errorf("expected nil, got %v", b)
	}
}

func TestSparkline_ZeroWidth(t *testing.T) {
	if b := Sparkline([]float64{1, 2, 3}, 0); b != nil {
		t.Errorf("expected nil, got %v", b)
	}
}

func TestSparkline_FewerValuesThanWidth(t *testing.T) {
	b := Sparkline([]float64{1.0, 1.1, 1.2, 1.3, 1.4}, 10)
	if len(b) != 5 {
		t.Errorf("expected width capped to 5, got %d", len(b))
	}
}

func TestSparkline_SingleValue(t *testing.T) {
	b := Sparkline([]float64{3.14}, 3)
	if len(b) != 1 {
		t.Errorf("expected width capped to 1, got %d", len(b))
	}
}

func TestSparkline_AllZero(t *testing.T) {
	b := Sparkline([]float64{0, 0, 0, 0, 0}, 3)
	for _, blk := range b {
		if blk.Char != '▄' {
			t.Errorf("all-zero should be ▄, got %c", blk.Char)
		}
		if blk.Value != 0 {
			t.Errorf("expected value 0, got %f", blk.Value)
		}
	}
}

func TestSparkline_ZeroMapsToMid(t *testing.T) {
	b := Sparkline([]float64{-5, -2.5, 0, 2.5, 5}, 5)
	if len(b) != 5 {
		t.Fatalf("expected 5 blocks, got %d", len(b))
	}
	if b[2].Char != '▄' {
		t.Errorf("0%% should be ▄, got %c", b[2].Char)
	}
}

func TestSparkline_AllPositiveAboveMid(t *testing.T) {
	b := Sparkline([]float64{1, 2, 3, 4, 5}, 5)
	for _, blk := range b {
		if blk.Char == '▃' || blk.Char == '▂' || blk.Char == '▁' {
			t.Errorf("all-positive should stay >= ▄, got %c", blk.Char)
		}
	}
}

func TestSparkline_AllNegativeBelowMid(t *testing.T) {
	b := Sparkline([]float64{-6, -3, -4, -2, -5}, 5)
	for _, blk := range b {
		if blk.Char == '▅' || blk.Char == '▆' || blk.Char == '▇' || blk.Char == '█' {
			t.Errorf("all-negative should stay <= ▄, got %c", blk.Char)
		}
	}
}

func TestSparkline_ValueSignMatchesChar(t *testing.T) {
	values := []float64{3, -1, 2, -4, 1}
	b := Sparkline(values, 5)
	for _, blk := range b {
		if blk.Value < 0 && blk.Char > '▄' {
			t.Errorf("negative value %.2f should have char ≤ ▄, got %c", blk.Value, blk.Char)
		}
		if blk.Value > 0 && blk.Char < '▄' {
			t.Errorf("positive value %.2f should have char ≥ ▄, got %c", blk.Value, blk.Char)
		}
	}
}

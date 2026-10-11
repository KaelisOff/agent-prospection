package main

import (
	"math"
	"testing"
	"time"
)

func TestProbability(t *testing.T) {
	// Same results: one chance in two.
	if p := probBBeatsA(11, 91, 11, 91); math.Abs(p-0.5) > 0.01 {
		t.Errorf("equal: %f", p)
	}
	// 10 % against 3 %, 150 sends each: B is clearly better.
	if p := probBBeatsA(1+5, 1+145, 1+15, 1+135); p < 0.98 || p > 0.995 { // z ≈ 2.3
		t.Errorf("clear: %f", p)
	}
	// Symmetry.
	p1 := probBBeatsA(3, 40, 8, 35)
	p2 := probBBeatsA(8, 35, 3, 40)
	if math.Abs(p1+p2-1) > 1e-9 {
		t.Errorf("not symmetric: %f + %f", p1, p2)
	}
}

func TestDecide(t *testing.T) {
	start := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		a, b Counts
		days int
		want string
	}{
		{Counts{50, 1}, Counts{50, 9}, 3, ""},      // too few sends
		{Counts{120, 3}, Counts{120, 14}, 10, "b"}, // B clearly better
		{Counts{120, 15}, Counts{120, 2}, 10, "a"}, // A clearly better
		{Counts{150, 6}, Counts{150, 8}, 10, ""},   // not clear yet
		{Counts{150, 6}, Counts{150, 8}, 30, "b"},  // too long: the best rate
		{Counts{150, 6}, Counts{150, 6}, 30, "a"},  // tie: keep A
	}
	for _, c := range cases {
		if got, _ := decide(c.a, c.b, start, start.Add(time.Duration(c.days)*24*time.Hour)); got != c.want {
			t.Errorf("%+v vs %+v after %d days: %q, want %q", c.a, c.b, c.days, got, c.want)
		}
	}
}

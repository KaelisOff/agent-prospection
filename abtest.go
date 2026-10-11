package main

import (
	"math"
	"time"
)

// A/B tests of the messages. Two variants are sent to contacts picked at
// random; the measure is the share of positive replies (interested or
// asking a question). Opens are not measured: that would need a tracking
// pixel.
//
// Each variant's rate gets a beta distribution (uniform prior); a
// variant wins when it has been sent at least minSent times and has more
// than 95 % chances of being the better one. After maxTestAge without a
// clear winner, the one with the best observed rate is kept.

const (
	minSent    = 100
	confidence = 0.95
	maxTestAge = 28 * 24 * time.Hour
)

// probBBeatsA is P(pB > pA) for pA ~ Beta(aA, bA) and pB ~ Beta(aB, bB),
// exact for an integer aB (Evan Miller's formula).
func probBBeatsA(aA, bA, aB, bB float64) float64 {
	total := 0.0
	for i := 0.0; i < aB; i++ {
		total += math.Exp(lbeta(aA+i, bA+bB) - math.Log(bB+i) - lbeta(1+i, bB) - lbeta(aA, bA))
	}
	return math.Min(1, math.Max(0, total))
}

func lbeta(a, b float64) float64 {
	la, _ := math.Lgamma(a)
	lb, _ := math.Lgamma(b)
	lab, _ := math.Lgamma(a + b)
	return la + lb - lab
}

// Counts are a variant's results.
type Counts struct {
	Sent     int `json:"sent"`
	Positive int `json:"positive"`
}

func (c Counts) rate() float64 {
	if c.Sent == 0 {
		return 0
	}
	return float64(c.Positive) / float64(c.Sent)
}

// decide returns the winner ("a" or "b"), or "" while the test goes on,
// and P(B beats A).
func decide(a, b Counts, started, now time.Time) (string, float64) {
	p := probBBeatsA(float64(1+a.Positive), float64(1+a.Sent-a.Positive), float64(1+b.Positive), float64(1+b.Sent-b.Positive))
	switch {
	case a.Sent >= minSent && b.Sent >= minSent && p >= confidence:
		return "b", p
	case a.Sent >= minSent && b.Sent >= minSent && p <= 1-confidence:
		return "a", p
	case now.Sub(started) >= maxTestAge && a.Sent+b.Sent > 0:
		if b.rate() > a.rate() {
			return "b", p
		}
		return "a", p
	}
	return "", p
}

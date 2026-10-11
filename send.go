package main

import (
	"hash/fnv"
	"math"
	"time"

	"github.com/KaelisOff/agent-sdk-go/agent"
)

// Sending: a share of the day's emails at each run, so that they spread
// over the day. The box decides in the end (one email per address,
// hours, its own limit, brakes): a refusal is not an error.

// Send times (box time), the manifest's schedules.
var slots = []struct{ h, m int }{{8, 30}, {10, 0}, {14, 0}, {16, 30}}

// slotsLeft is the number of send times left today, this run included.
func slotsLeft(now time.Time) int {
	left := 0
	for _, s := range slots {
		if now.Hour() < s.h || (now.Hour() == s.h && now.Minute() <= s.m+5) {
			left++
		}
	}
	return max(left, 1)
}

// variant picks A or B for an address, half and half, the same for the
// same address.
func variant(email string) string {
	h := fnv.New32a()
	h.Write([]byte(email))
	if h.Sum32()%2 == 0 {
		return "a"
	}
	return "b"
}

// sendShare sends this run's share of the day's emails. It returns the
// queue without the companies dealt with.
func sendShare(box *agent.Box, cfg Config, st *State, queue []Prospect, seen map[string]string, now time.Time, manual bool, bud *budget) []Prospect {
	if st.Test == nil || len(queue) == 0 {
		return queue
	}
	left := cfg.Daily - st.SentToday
	if left <= 0 {
		return queue
	}
	share := left
	if !manual {
		share = int(math.Ceil(float64(left) / float64(slotsLeft(now))))
	}
	sent := 0
	for len(queue) > 0 && sent < share && bud.spend(1) {
		p := queue[0]
		v := variant(p.Email)
		tpl := st.Test.A
		if v == "b" {
			tpl = st.Test.B
		}
		subject, body := render(tpl, p)
		_, err := box.SendMail(agent.Mail{To: p.Email, Subject: subject, Body: body, Campaign: p.Target, Variant: st.Test.ID + v})
		switch {
		case err == nil:
			sent++
			st.SentToday++
			st.Since.Sent++
			st.campaign(p.Target).Sent++
			if v == "a" {
				st.Test.CountsA.Sent++
			} else {
				st.Test.CountsB.Sent++
			}
			seen[p.SIREN] = seenSent
			queue = queue[1:]
		case agent.IsCode(err, agent.CodeAlreadyContacted), agent.IsCode(err, agent.CodeOptedOut),
			agent.IsCode(err, agent.CodeBounced), agent.IsCode(err, agent.CodeInvalid):
			// Not for this address: next one.
			box.Logf("%s refused: %v", p.Email, err)
			seen[p.SIREN] = seenRejected
			queue = queue[1:]
			if agent.IsCode(err, agent.CodeBounced) {
				st.Since.Bounce++
				st.campaign(p.Target).Bounce++
			}
		default:
			// Outside the hours, limit reached, paused, not configured,
			// server down: try again at the next run.
			box.Logf("sending stops: %v", err)
			return queue
		}
	}
	box.Logf("%d emails sent", sent)
	return queue
}

// Command prospection is a Kaelis agent that finds clients by email:
// companies of the owner's targets in the official directory, their
// generic contact address on their own site, a short message written by
// the box's model and improved by A/B tests, a report every morning.
//
// It runs in the box's sandbox, without network: the directory, the web,
// the emails and the model are the box's tools, with the box's
// guardrails (one email per address, STOP, hours, limits, brakes).
package main

import (
	"fmt"
	"time"

	"github.com/KaelisOff/agent-sdk-go/agent"
)

// morning is the schedule of the morning run (report, review, messages).
const morning = "30 8 * * 1-5"

func main() { agent.Main(run) }

func run(box *agent.Box, r agent.Run) (string, error) {
	started := time.Now()
	settings, err := box.Settings()
	if err != nil {
		return "", err
	}
	cfg := loadConfig(settings)
	st, queue, seen, err := load(box)
	if err != nil {
		return "", err
	}
	now := r.Now
	st.newDay(now)
	// The manifest gives a run 20 minutes and the box 200 calls: keep a
	// margin to save the state.
	bud := &budget{maxCalls: 185, deadline: started.Add(15 * time.Minute)}

	if r.Trigger.Kind == agent.TriggerApproval && r.Trigger.Approval != nil {
		summary := onApproval(box, st, r.Trigger.Approval, now)
		return summary, save(box, st, queue, seen)
	}

	if err := handleReplies(box, st, bud); err != nil {
		box.Logf("replies: %v", err)
	}
	today := now.Format("2006-01-02")
	first := st.LastReport.IsZero() || st.LastReport.Format("2006-01-02") != today
	if (r.Trigger.Schedule == morning || r.Trigger.Kind == agent.TriggerManual) && first && now.Weekday() != time.Saturday && now.Weekday() != time.Sunday {
		if now.Sub(st.LastReview) >= 7*24*time.Hour && !st.LastReview.IsZero() {
			review(box, cfg, st, now, bud)
		}
		if st.LastReview.IsZero() {
			st.LastReview = now // the first review comes a week from now
		}
		mail, err := box.MailStatus()
		if err != nil {
			box.Logf("mail status: %v", err)
		}
		report(box, cfg, st, len(queue), mail, now)
	}
	ensureMessages(box, cfg, st, now, bud)

	// Find a day's worth of companies ahead, then send this run's share.
	queue = fill(box, cfg, st, queue, seen, 2*cfg.Daily, bud, today)
	queue = sendShare(box, cfg, st, queue, seen, now, r.Trigger.Kind == agent.TriggerManual, bud)

	if err := save(box, st, queue, seen); err != nil {
		return "", err
	}
	summary := fmt.Sprintf("%s aujourd'hui, %s", plural(st.SentToday, "email envoyé", "emails envoyés"), plural(len(queue), "contact prêt", "contacts prêts"))
	if st.Pending != nil {
		summary += ", messages en attente de votre accord"
	}
	return summary, nil
}

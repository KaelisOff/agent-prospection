package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/KaelisOff/agent-sdk-go/agent"
)

// The owner's view: the morning report, the weekly review, the requests
// for approval of new messages.

func plural(n int, one, many string) string {
	if n == 1 || n == 0 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

// report sends the morning report.
func report(box *agent.Box, cfg Config, st *State, queueLen int, mail agent.MailStatus, now time.Time) {
	var b strings.Builder
	s := st.Since
	fmt.Fprintf(&b, "Depuis le dernier point : %s, %s, %s, %s, %s, %s.\n",
		plural(s.Sent, "email envoyé", "emails envoyés"), plural(s.Positive, "réponse intéressée", "réponses intéressées"),
		plural(s.Negative, "réponse négative", "réponses négatives"), plural(s.Auto, "réponse automatique", "réponses automatiques"),
		plural(s.Unsubscribe, "désinscription", "désinscriptions"), plural(s.Bounce, "adresse en erreur", "adresses en erreur"))
	fmt.Fprintf(&b, "Prêts à écrire : %s.\n", plural(queueLen, "entreprise", "entreprises"))
	if t := st.Test; t != nil {
		fmt.Fprintf(&b, "Test %s : A (%s) %d envois, %d positifs ; B (%s) %d envois, %d positifs.",
			t.ID, t.A.Angle, t.CountsA.Sent, t.CountsA.Positive, t.B.Angle, t.CountsB.Sent, t.CountsB.Positive)
		if need := max(minSent-t.CountsA.Sent, minSent-t.CountsB.Sent); need > 0 {
			fmt.Fprintf(&b, " Encore environ %d envois par message avant de conclure.", need)
		}
		b.WriteString("\n")
	}
	switch {
	case st.Pending != nil:
		b.WriteString("De nouveaux messages attendent votre accord (Centre d'agents).\n")
	case st.Test == nil && st.Rejected >= 3:
		b.WriteString("Vous avez refusé 3 propositions de message cette semaine : je reprends lundi. Vous pouvez préciser votre offre dans les réglages.\n")
	}
	if !cfg.Review {
		b.WriteString("Mode autonome : les nouveaux messages partent sans votre relecture.\n")
	}
	switch {
	case !mail.Configured:
		b.WriteString("La messagerie de la boîte n'est pas configurée : rien ne peut partir (Réglages → Messagerie).\n")
	case mail.Paused:
		b.WriteString("Les envois sont en pause (trop d'erreurs ou de désinscriptions) : à reprendre dans Réglages → Messagerie.\n")
	default:
		fmt.Fprintf(&b, "Aujourd'hui : jusqu'à %d emails (la boîte en autorise %d en ce moment).\n", cfg.Daily, mail.LimitToday)
	}
	if st.SearchDone == now.Format("2006-01-02") {
		b.WriteString("Le quota de recherches web du jour a été atteint hier ou aujourd'hui.\n")
	}
	exhausted := 0
	for _, t := range cfg.Targets {
		if p := st.Progress[t.Key()]; p != nil && p.Exhausted {
			exhausted++
		}
	}
	if len(cfg.Targets) > 0 && exhausted == len(cfg.Targets) {
		b.WriteString("Toutes les cibles ont été parcourues : ajoutez des métiers ou des lieux dans les réglages.\n")
	}
	for _, p := range cfg.Problems {
		b.WriteString("Réglage à corriger : " + p + ".\n")
	}
	if err := notify(box, cfg, "Point du jour", strings.TrimSpace(b.String())); err != nil {
		box.Logf("report: %v", err)
	}
	st.Since = Tally{}
	st.LastReport = now
}

// review is the weekly review: the test's outcome, the targets.
func review(box *agent.Box, cfg Config, st *State, now time.Time, bud *budget) {
	var b strings.Builder
	if t := st.Test; t != nil {
		winner, p := decide(t.CountsA, t.CountsB, t.Started, now)
		if winner != "" && st.Pending == nil {
			w := t.A
			if winner == "b" {
				w = t.B
			}
			fmt.Fprintf(&b, "Test %s terminé : le message %s l'emporte (%s ; probabilité que B soit meilleur : %.0f %%).\n",
				t.ID, strings.ToUpper(winner), w.Angle, p*100)
			st.TestsDone++
			if bud.spend(4) {
				next := newTest(box, cfg, st, &w)
				proposeOrStart(box, cfg, st, next, now, "Le message gagnant du test précédent (A) contre un nouveau message (B).")
			}
		} else {
			fmt.Fprintf(&b, "Test %s en cours : pas encore de gagnant net (probabilité que B soit meilleur : %.0f %%).\n", t.ID, p*100)
		}
	}
	type line struct {
		key string
		t   Tally
	}
	var lines []line
	for k, t := range st.Campaigns {
		if t.Sent > 0 {
			lines = append(lines, line{k, *t})
		}
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].t.Sent > lines[j].t.Sent })
	for i, l := range lines {
		if i == 8 {
			break
		}
		rate := 100 * float64(l.t.Positive) / float64(l.t.Sent)
		fmt.Fprintf(&b, "• %s : %d envois, %d intéressés (%.1f %%), %d désinscriptions.\n", l.key, l.t.Sent, l.t.Positive, rate, l.t.Unsubscribe)
		if l.t.Sent >= 60 && l.t.Positive == 0 {
			fmt.Fprintf(&b, "  Cette cible ne répond pas : remplacez-la dans les réglages ?\n")
		}
	}
	if b.Len() == 0 {
		b.WriteString("Pas encore assez d'envois pour un bilan.")
	}
	if err := notify(box, cfg, "Bilan de la semaine", strings.TrimSpace(b.String())); err != nil {
		box.Logf("review: %v", err)
	}
	st.LastReview = now
}

// proposeOrStart puts new messages to the test: after the owner's
// approval in review mode, at once otherwise.
func proposeOrStart(box *agent.Box, cfg Config, st *State, t Test, now time.Time, why string) {
	if !cfg.Review {
		t.Started = now
		st.Test = &t
		_ = box.Notify("Nouveaux messages en test", why+"\n\n"+describe("Message A", t.A)+"\n\n"+describe("Message B", t.B))
		return
	}
	data, _ := json.Marshal(map[string]string{"kind": "messages", "test": t.ID})
	details := why + "\n\n" + describe("Message A", t.A) + "\n\n" + describe("Message B", t.B) +
		"\n\nLa boîte ajoute votre identité, la raison du message et « répondez STOP ». {entreprise} et {ville} sont remplis pour chaque entreprise."
	if r := []rune(details); len(r) > 4000 {
		details = string(r[:3990]) + "…"
	}
	id, err := box.RequestApproval("Nouveaux messages à valider ("+t.ID+")", details, json.RawMessage(data), 72)
	if err != nil {
		box.Logf("approval: %v", err)
		return
	}
	st.Pending = &Pending{ApprovalID: id, Test: t, Asked: now}
}

// ensureMessages makes sure there are messages to send, or a proposal on
// the way.
func ensureMessages(box *agent.Box, cfg Config, st *State, now time.Time, bud *budget) {
	if st.Pending != nil && !cfg.Review {
		// Switched to autonomous mode: the proposal goes ahead.
		t := st.Pending.Test
		t.Started = now
		st.Test, st.Pending = &t, nil
		return
	}
	if st.Test != nil || st.Pending != nil || st.Rejected >= 3 || !bud.spend(4) {
		return
	}
	t := newTest(box, cfg, st, nil)
	proposeOrStart(box, cfg, st, t, now, "Les premiers messages : deux façons d'aborder les entreprises, testées l'une contre l'autre.")
}

// onApproval handles the owner's answer about new messages.
func onApproval(box *agent.Box, st *State, a *agent.Approval, now time.Time) string {
	var data struct {
		Kind string `json:"kind"`
		Test string `json:"test"`
	}
	_ = json.Unmarshal(a.Data, &data)
	if data.Kind != "messages" || st.Pending == nil || st.Pending.Test.ID != data.Test {
		return "réponse sans objet"
	}
	if a.Approved {
		t := st.Pending.Test
		t.Started = now
		st.Test, st.Pending = &t, nil
		return "messages " + t.ID + " acceptés : ils partent dès le prochain envoi"
	}
	st.Pending = nil
	st.Rejected++
	if a.Expired {
		return "pas de réponse : de nouveaux messages seront proposés"
	}
	return "messages refusés : de nouveaux seront proposés"
}

// notify sends a notification, also by email if the owner asked so (and
// the box has a mailbox).
func notify(box *agent.Box, cfg Config, title, body string) error {
	if !cfg.ByEmail {
		return box.Notify(title, body)
	}
	_, err := box.NotifyByEmail(title, body)
	return err
}

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/KaelisOff/agent-sdk-go/agent"
	"github.com/KaelisOff/agent-sdk-go/agenttest"
)

const goodMessage = `Objet : Une idée pour {entreprise}

Bonjour,

Je me permets d'écrire à {entreprise}, à {ville}, car beaucoup d'artisans du même métier perdent du temps sur leurs devis et leurs relances.
Nous les aidons à tenir leur comptabilité à jour chaque mois, sans y passer leurs soirées, avec un interlocuteur unique.
Est-ce un sujet pour vous en ce moment ?

Cordialement,
[Votre nom]`

// fakeModel writes messages and sorts replies like the box's model would.
func fakeModel(msgs []agent.Message, _ int) (string, error) {
	sys := msgs[0].Content
	switch {
	case strings.Contains(sys, "emails de prospection"):
		return goodMessage, nil
	case strings.Contains(sys, "Tu classes"):
		if strings.Contains(strings.ToLower(msgs[len(msgs)-1].Content), "oui") {
			return "INTERESSE", nil
		}
		return "PAS_INTERESSE", nil
	}
	return "", nil
}

func newBox(t *testing.T) *agenttest.Box {
	t.Helper()
	box, err := agenttest.Load("manifest.toml")
	if err != nil {
		t.Fatal(err)
	}
	paris, _ := time.LoadLocation("Europe/Paris")
	box.Location = paris
	box.Model = fakeModel
	box.Settings = map[string]any{
		"offer":      "Tenue de comptabilité mensuelle pour les artisans du bâtiment.",
		"activities": []any{"43.22A"},
		"places":     []any{"75011", "Paris"},
	}
	box.Directory = func(q agent.CompanyQuery) []agent.Company {
		if q.PostalCode != "75011" || q.Activities[0] != "43.22A" {
			return nil
		}
		return []agent.Company{
			{SIREN: "111111111", Name: "PLOMBERIE DUBERT", City: "PARIS", Activity: "43.22A"},
			{SIREN: "222222222", Name: "MARIE MARTIN", City: "PARIS", SoleTrader: true},
			{SIREN: "333333333", Name: "CHAUFFAGE ROQUETTE", City: "PARIS", PostalCode: "75011"},
			{SIREN: "444444444", Name: "SANS SITE SARL", City: "PARIS"},
		}
	}
	box.Web = func(q string) []agent.WebResult {
		switch {
		case strings.Contains(q, "DUBERT"):
			return []agent.WebResult{{URL: "https://www.pagesjaunes.fr/dubert"}, {URL: "https://www.plomberie-dubert.fr/"}}
		case strings.Contains(q, "ROQUETTE"):
			return []agent.WebResult{{URL: "https://chauffage-roquette.fr/accueil"}}
		}
		return []agent.WebResult{{URL: "https://www.pagesjaunes.fr/autre"}}
	}
	box.Pages = map[string]agent.Page{
		"https://www.plomberie-dubert.fr/":      {Text: "Plombier à Paris 11e. Contact : contact@plomberie-dubert.fr"},
		"https://chauffage-roquette.fr/":        {Text: "Chauffagiste, 12 rue de la Roquette, 75011 Paris", Links: []agent.Link{{Text: "Nous écrire", URL: "https://chauffage-roquette.fr/contact"}}},
		"https://chauffage-roquette.fr/contact": {Links: []agent.Link{{URL: "mailto:accueil@chauffage-roquette.fr"}, {URL: "mailto:paul.roquette@gmail.com"}}},
	}
	return box
}

func at(box *agenttest.Box, day, h, m int) {
	box.Now = time.Date(2026, 10, day, h, m, 0, 0, box.Location)
}

func mustRun(t *testing.T, box *agenttest.Box, trig agent.Trigger) agenttest.Result {
	t.Helper()
	res := box.Run(run, trig)
	if res.Status != "ok" {
		t.Fatalf("run: %+v\n%s", res, res.Logs)
	}
	return res
}

func TestProspection(t *testing.T) {
	box := newBox(t)

	// Monday 8:30: report, messages proposed, companies found; nothing
	// sent before the owner's approval.
	at(box, 12, 8, 30)
	mustRun(t, box, agenttest.Scheduled(morning))
	if len(box.SentMail()) != 0 || len(box.Approvals) != 1 {
		t.Fatalf("sent %d, approvals %d", len(box.SentMail()), len(box.Approvals))
	}
	report := box.Notifications[0]
	if report.Title != "Point du jour" || !strings.Contains(report.Body, "Prêts à écrire : 0 entreprise") || !strings.Contains(report.Body, "lieu « PARIS »") {
		t.Errorf("report = %+v", report)
	}
	details := box.Approvals[0].Details
	if !strings.Contains(details, "Objet : Une idée pour {entreprise}") || strings.Contains(details, "[Votre nom]") || strings.Contains(details, "Cordialement") {
		t.Errorf("proposal = %s", details)
	}
	// Generic addresses only, from the companies' own sites.
	var queue []Prospect
	if _, err := getJSON(box, "queue", &queue); err != nil || len(queue) != 2 {
		t.Fatalf("queue = %+v, %v", queue, err)
	}
	if queue[0].Email != "contact@plomberie-dubert.fr" || queue[1].Email != "accueil@chauffage-roquette.fr" {
		t.Errorf("contacts = %+v", queue)
	}

	// The owner approves; at 10:00 the emails go.
	mustRun(t, box, box.Answer(box.Approvals[0].ID, true))
	at(box, 12, 10, 0)
	mustRun(t, box, agenttest.Scheduled("0 10 * * 1-5"))
	sent := box.SentMail()
	if len(sent) != 2 {
		t.Fatalf("sent = %+v", sent)
	}
	for _, m := range sent {
		if strings.ContainsAny(m.Body+m.Subject, "{}[]") || !strings.Contains(m.Body, "Paris") || m.Campaign != "43-22a-75011" || !strings.HasPrefix(m.Variant, "t1") {
			t.Errorf("email = %+v", m)
		}
	}
	if !strings.Contains(sent[0].Subject, "Plomberie Dubert") {
		t.Errorf("subject = %q", sent[0].Subject)
	}

	// Replies: the interested one is notified at once.
	box.Reply("contact@plomberie-dubert.fr", "Oui, rappelez-moi jeudi.")
	box.Unsubscribe("accueil@chauffage-roquette.fr")
	at(box, 12, 14, 0)
	before := len(box.Notifications)
	mustRun(t, box, agenttest.Scheduled("0 14 * * 1-5"))
	if len(box.Notifications) != before+1 || !strings.Contains(box.Notifications[before].Title, "est intéressé") {
		t.Errorf("notifications = %+v", box.Notifications[before:])
	}

	// Tuesday morning: the report counts it all.
	at(box, 13, 8, 30)
	mustRun(t, box, agenttest.Scheduled(morning))
	last := box.Notifications[len(box.Notifications)-1]
	for _, want := range []string{"2 emails envoyés", "1 réponse intéressée", "1 désinscription", "Test t1"} {
		if !strings.Contains(last.Body, want) {
			t.Errorf("report lacks %q:\n%s", want, last.Body)
		}
	}
}

func TestAutonomousMode(t *testing.T) {
	box := newBox(t)
	box.Settings["review"] = false
	at(box, 12, 8, 30)
	mustRun(t, box, agenttest.Scheduled(morning))
	// No approval asked: the messages are tested at once.
	if len(box.Approvals) != 0 || len(box.SentMail()) == 0 {
		t.Errorf("approvals %d, sent %d", len(box.Approvals), len(box.SentMail()))
	}
	found := false
	for _, n := range box.Notifications {
		found = found || n.Title == "Nouveaux messages en test"
	}
	if !found {
		t.Error("the owner was not shown the messages")
	}
}

func TestRefusedMessages(t *testing.T) {
	box := newBox(t)
	at(box, 12, 8, 30)
	mustRun(t, box, agenttest.Scheduled(morning))
	res := mustRun(t, box, box.Answer(box.Approvals[0].ID, false))
	if !strings.Contains(res.Summary, "refusés") {
		t.Errorf("summary = %q", res.Summary)
	}
	// Nothing goes out; new messages are proposed the next morning.
	at(box, 12, 10, 0)
	mustRun(t, box, agenttest.Scheduled("0 10 * * 1-5"))
	if len(box.SentMail()) != 0 {
		t.Errorf("sent without approval: %+v", box.SentMail())
	}
	at(box, 13, 8, 30)
	mustRun(t, box, agenttest.Scheduled(morning))
	if len(box.Approvals) != 2 {
		t.Errorf("approvals = %d", len(box.Approvals))
	}
}

func getJSON(box *agenttest.Box, key string, v any) (bool, error) {
	raw, ok := box.Storage[key]
	if !ok {
		return false, nil
	}
	return true, jsonUnmarshal(raw, v)
}

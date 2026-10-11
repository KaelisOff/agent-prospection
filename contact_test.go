package main

import (
	"testing"

	"github.com/KaelisOff/agent-sdk-go/agent"
)

func TestChooseSite(t *testing.T) {
	results := []agent.WebResult{
		{URL: "https://www.pagesjaunes.fr/pros/plomberie-dubert"},
		{URL: "https://www.facebook.com/plomberiedubert"},
		{URL: "https://www.plomberie-dubert.fr/realisations"},
	}
	if site, ok := chooseSite(results, "PLOMBERIE DUBERT"); !ok || site != "https://www.plomberie-dubert.fr/" {
		t.Errorf("site = %q, %v", site, ok)
	}
	// No distinctive match: no site.
	if site, ok := chooseSite([]agent.WebResult{{URL: "https://www.plombier-paris-11.fr/"}}, "SARL DUBERT ET FILS"); ok {
		t.Errorf("guessed %q", site)
	}
	// Accents and generic words.
	if site, ok := chooseSite([]agent.WebResult{{URL: "https://electricite-genereux.fr/"}}, "ÉLECTRICITÉ GÉNÉREUX SARL"); !ok || site == "" {
		t.Errorf("accents: %q", site)
	}
}

func TestEmailsAndContact(t *testing.T) {
	page := agent.Page{
		Text: "Écrivez-nous : contact [at] plomberie-dubert [dot] fr ou jean.dubert@gmail.com. Recrutement : jobs@plomberie-dubert.fr",
		Links: []agent.Link{
			{URL: "mailto:devis@plomberie-dubert.fr"},
			{URL: "https://www.plomberie-dubert.fr/nous-contacter", Text: "Nous contacter"},
			{URL: "https://www.facebook.com/x", Text: "Contact Facebook"},
		},
	}
	emails := emailsOf(page)
	want := map[string]bool{"devis@plomberie-dubert.fr": true, "contact@plomberie-dubert.fr": true, "jean.dubert@gmail.com": true, "jobs@plomberie-dubert.fr": true}
	if len(emails) != len(want) {
		t.Fatalf("emails = %v", emails)
	}
	for _, e := range emails {
		if !want[e] {
			t.Errorf("unexpected %s", e)
		}
	}
	if got := pickContact(emails, "www.plomberie-dubert.fr", false, "PLOMBERIE DUBERT"); got != "devis@plomberie-dubert.fr" && got != "contact@plomberie-dubert.fr" {
		t.Errorf("contact = %q", got)
	}
	// A person's address is never taken.
	if got := pickContact([]string{"jean.dubert@gmail.com"}, "www.plomberie-dubert.fr", false, "JEAN DUBERT"); got != "" {
		t.Errorf("personal address taken: %q", got)
	}
	// The company's name in a free mailbox is (not for sole traders).
	if got := pickContact([]string{"plomberie.dubert@orange.fr"}, "", false, "PLOMBERIE DUBERT"); got != "plomberie.dubert@orange.fr" {
		t.Errorf("company mailbox: %q", got)
	}
	if got := pickContact([]string{"marie.martin@orange.fr"}, "", true, "MARIE MARTIN"); got != "" {
		t.Errorf("sole trader's own address: %q", got)
	}
	if got := contactPage(page, "www.plomberie-dubert.fr"); got != "https://www.plomberie-dubert.fr/nous-contacter" {
		t.Errorf("contact page = %q", got)
	}
}

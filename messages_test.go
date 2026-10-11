package main

import (
	"strings"
	"testing"

	"github.com/KaelisOff/agent-sdk-go/agent"
)

const offer = "Tenue de comptabilité mensuelle pour les artisans du bâtiment."

func TestParseTemplate(t *testing.T) {
	ok := "Objet : **Votre comptabilité, simplement**\n\nBonjour,\n\nLes artisans comme {entreprise}, à {ville}, passent souvent leurs soirées sur les factures et les relances.\n" +
		"**Et si nous nous en occupions ?** Nous tenons la comptabilité chaque mois, avec un interlocuteur unique et des bilans clairs.\n" +
		"* sans paperasse\n* sans surprise\n\nEn parlons-nous cette semaine ?\n\nCordialement,\n[Votre nom]"
	tpl, good := parseTemplate(ok, offer)
	if !good {
		t.Fatal("a good message was refused")
	}
	if strings.Contains(tpl.Body+tpl.Subject, "*") || strings.Contains(tpl.Body, "[Votre nom]") || strings.Contains(tpl.Body, "Cordialement") ||
		!strings.Contains(tpl.Body, "- sans paperasse") || tpl.Subject != "Votre comptabilité, simplement" {
		t.Errorf("template = %+v", tpl)
	}
	for name, bad := range map[string]string{
		"recipient as sender": strings.Replace(ok, "Les artisans comme {entreprise}", "Chez {entreprise}, nous aidons les artisans", 1),
		"made-up numbers":     strings.Replace(ok, "passent souvent leurs soirées", "perdent 2 heures par semaine", 1),
		"no subject":          strings.Replace(ok, "Objet : ", "", 1),
		"too short":           "Objet : Bonjour\n\nUne offre pour {entreprise}.",
	} {
		if _, good := parseTemplate(bad, offer); good {
			t.Errorf("%s: accepted", name)
		}
	}
	// Numbers are fine when the offer has them.
	if _, good := parseTemplate(strings.Replace(ok, "passent souvent leurs soirées", "perdent 2 heures par semaine", 1), "Comptabilité dès 90 € par mois en 2 heures."); !good {
		t.Error("numbers of the offer refused")
	}
}

func TestAboutCompany(t *testing.T) {
	// The real case: same name, same city, another trade.
	plumber := agent.Company{Name: "ARTISAN BERTRAND", City: "PARIS", Activity: "43.22A"}
	if aboutCompany(agent.Page{Title: "Bertrand, Artisan Floral à Paris", Text: "Nos bouquets livrés chez vous"}, plumber) {
		t.Error("a florist taken for a plumber of the same name")
	}
	if !aboutCompany(agent.Page{Title: "Artisan Bertrand", Text: "Plombier chauffagiste à Paris"}, plumber) {
		t.Error("the plumber's site refused")
	}
	c := agent.Company{Name: "ARTISAN BERTRAND CHARENTES", PostalCode: "75011", City: "PARIS"}
	florist := agent.Page{Title: "Bertrand, artisan fleuriste", Text: "Bouquets et compositions florales, 12 rue Oberkampf, 75011 Paris."}
	if aboutCompany(florist, c) {
		t.Error("a florist taken for a plumber")
	}
	if !aboutCompany(agent.Page{Text: "Artisan Bertrand Charentes, plombier"}, c) {
		t.Error("the company's own site refused")
	}
	rousseau := agent.Company{Name: "ENTREPRISE ROUSSEAU", PostalCode: "75011", City: "PARIS"}
	if aboutCompany(agent.Page{Text: "Entreprise Rousseau, couvreur à Nantes (44000)"}, rousseau) {
		t.Error("another Rousseau taken")
	}
	if !aboutCompany(agent.Page{Text: "Entreprise Rousseau, plomberie, 75011 Paris"}, rousseau) {
		t.Error("the right Rousseau refused")
	}
	if !aboutCompany(agent.Page{Text: "Plomberie, 3 rue de Charonne 75011"}, agent.Company{Name: "SARL PLOMBERIE", PostalCode: "75011"}) {
		t.Error("a name without distinctive word, with its postal code, refused")
	}
}

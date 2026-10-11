package main

import (
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/KaelisOff/agent-sdk-go/agent"
)

// Finding a company's contact address, from public sources only: the
// company's own website (found by a web search), its home page and its
// contact page. Only generic addresses are kept (contact@, accueil@…): a
// named person's address is left alone.

// Sites that list companies rather than being one: never taken as a
// company's own site.
var directories = []string{
	"pagesjaunes.fr", "societe.com", "pappers.fr", "infogreffe.fr", "verif.com", "manageo.fr", "kompass.com",
	"annuaire-entreprises.data.gouv.fr", "data.gouv.fr", "gouv.fr", "facebook.com", "linkedin.com", "instagram.com",
	"twitter.com", "x.com", "youtube.com", "tiktok.com", "google.com", "google.fr", "yelp.fr", "yelp.com",
	"tripadvisor.fr", "mappy.com", "118712.fr", "118000.fr", "habitatpresto.com", "starofservice.com", "travaux.com",
	"houzz.fr", "leboncoin.fr", "indeed.com", "indeed.fr", "wikipedia.org", "b-reputation.com", "trustpilot.com",
	"lefigaro.fr", "ouest-france.fr", "francebleu.fr", "doctrine.fr", "corporama.com", "dun-bradstreet.com",
	"ellisphere.com", "entreprises.lefigaro.fr", "annuaire.laposte.fr", "bilansgratuits.fr", "rubypayeur.com",
	"societeinfo.com", "score3.fr", "infonet.fr", "lagazettefrance.fr", "pagespro.com", "hoodspot.fr",
}

func isDirectory(host string) bool {
	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	for _, d := range directories {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

var accents = strings.NewReplacer("à", "a", "â", "a", "ä", "a", "á", "a", "ç", "c", "é", "e", "è", "e", "ê", "e", "ë", "e",
	"î", "i", "ï", "i", "í", "i", "ô", "o", "ö", "o", "ó", "o", "ù", "u", "û", "u", "ü", "u", "ú", "u", "ÿ", "y", "ñ", "n",
	"œ", "oe", "æ", "ae", "’", "'")

// fold lowercases and removes accents: "Électricité" → "electricite".
func fold(s string) string { return accents.Replace(strings.ToLower(s)) }

// Words of company names that say nothing about which company it is.
var commonWords = map[string]bool{
	"sarl": true, "sas": true, "sasu": true, "eurl": true, "sci": true, "snc": true, "scop": true, "earl": true,
	"societe": true, "entreprise": true, "ets": true, "etablissements": true, "groupe": true, "france": true,
	"services": true, "service": true, "the": true, "les": true, "des": true, "et": true, "de": true, "du": true,
	"la": true, "le": true, "aux": true, "pour": true, "paris": true, "lyon": true, "marseille": true,
}

// nameTokens are the distinctive words of a company's names (4 letters
// or more).
func nameTokens(names ...string) []string {
	var out []string
	for _, n := range names {
		for _, w := range strings.FieldsFunc(fold(n), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
			if len(w) >= 4 && !commonWords[w] && !slices.Contains(out, w) {
				out = append(out, w)
			}
		}
	}
	return out
}

// chooseSite picks, among search results, the company's own site: not a
// directory, and its domain carries a distinctive word of the company's
// name. No match: no site (better none than someone else's).
func chooseSite(results []agent.WebResult, names ...string) (string, bool) {
	tokens := nameTokens(names...)
	if len(tokens) == 0 {
		return "", false
	}
	for _, r := range results {
		u, err := url.Parse(r.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || isDirectory(u.Hostname()) {
			continue
		}
		domain := fold(strings.ReplaceAll(strings.TrimPrefix(u.Hostname(), "www."), "-", ""))
		for _, t := range tokens {
			if strings.Contains(domain, t) {
				return u.Scheme + "://" + u.Host + "/", true
			}
		}
	}
	return "", false
}

var (
	emailRe = regexp.MustCompile(`(?i)\b[a-z0-9][a-z0-9._%+-]{0,63}@[a-z0-9.-]+\.[a-z]{2,24}\b`)
	// "contact [at] exemple [dot] fr", "contact(at)exemple.fr"
	atRe  = regexp.MustCompile(`(?i)\s*[\[(]\s*(at|arobase)\s*[\])]\s*`)
	dotRe = regexp.MustCompile(`(?i)\s*[\[(]\s*(dot|point)\s*[\])]\s*`)
)

// emailsOf collects the addresses of a page: mailto links and addresses
// written in the text (also "[at]"/"[dot]" spellings).
func emailsOf(p agent.Page) []string {
	var out []string
	add := func(e string) {
		e = strings.ToLower(strings.Trim(e, ".-_"))
		if strings.Count(e, "@") == 1 && !slices.Contains(out, e) {
			out = append(out, e)
		}
	}
	for _, l := range p.Links {
		if addr, ok := strings.CutPrefix(l.URL, "mailto:"); ok {
			add(addr)
		}
	}
	text := dotRe.ReplaceAllString(atRe.ReplaceAllString(p.Text, "@"), ".")
	for _, e := range emailRe.FindAllString(text, 50) {
		add(e)
	}
	return out
}

// Local parts of addresses meant for any visitor of a company.
var genericLocal = []string{
	"contact", "contacts", "info", "infos", "information", "informations", "accueil", "bonjour", "hello", "salut",
	"agence", "cabinet", "secretariat", "commercial", "commerciale", "devis", "office", "bureau", "administration",
	"gestion", "entreprise", "societe", "courrier", "mail", "email", "reception", "atelier", "boutique", "magasin",
	"service", "clients", "client", "serviceclient", "sav", "direction", "general", "nous",
}

// Trade words: an address made of the company's name and one of them is
// the business's, not a person's ("plomberie.dubert@…", not
// "jean.dubert@…").
var tradeWords = []string{
	"plomberie", "plombier", "electricite", "electricien", "elec", "chauffage", "climatisation", "menuiserie", "menuisier",
	"maconnerie", "macon", "peinture", "peintre", "couverture", "couvreur", "charpente", "carrelage", "renovation",
	"batiment", "travaux", "garage", "auto", "boulangerie", "patisserie", "coiffure", "beaute", "institut", "restaurant",
	"cafe", "hotel", "pharmacie", "cabinet", "agence", "atelier", "studio", "conseil", "consulting", "expertise",
	"avocat", "avocats", "notaire", "immobilier", "immo", "transport", "transports", "nettoyage", "jardin", "paysage",
	"sarl", "sas", "sasu", "eurl", "ets", "entreprise", "societe", "groupe", "services", "boutique", "bureau",
}

// Addresses never to prospect: machines, privacy officers, recruiting.
var refusedLocal = []string{
	"noreply", "no-reply", "donotreply", "postmaster", "webmaster", "abuse", "hostmaster", "mailer-daemon",
	"dpo", "rgpd", "gdpr", "privacy", "cnil", "dpd", "jobs", "job", "recrutement", "candidature", "rh", "careers",
	"compta", "comptabilite", "facturation", "factures", "invoice", "unsubscribe", "newsletter",
}

// Domains that are not a company's: examples, trackers and site builders.
var refusedDomains = []string{"example.com", "exemple.fr", "domain.com", "sentry.io", "wixpress.com", "sentry-next.wixpress.com"}

// pickContact chooses the address to write to: a generic one, on the
// company's site domain if possible. For sole traders, only clearly
// generic addresses: their name is a person's.
func pickContact(emails []string, siteHost string, soleTrader bool, names ...string) string {
	site := strings.TrimPrefix(strings.ToLower(siteHost), "www.")
	tokens := nameTokens(names...)
	best, bestScore := "", 0
	for _, e := range emails {
		local, domain, _ := strings.Cut(e, "@")
		base, _, _ := strings.Cut(local, "+")
		if slices.Contains(refusedLocal, base) || slices.Contains(refusedDomains, domain) || strings.HasSuffix(domain, ".png") || strings.HasSuffix(domain, ".jpg") {
			continue
		}
		score := 0
		switch {
		case slices.Contains(genericLocal, base):
			score = 3
		case !soleTrader && containsAny(fold(base), tokens) && containsAny(fold(base), tradeWords):
			// plomberie.dubert@… : the company's name with its trade.
			score = 2
		default:
			continue
		}
		if site != "" && (domain == site || strings.HasSuffix(site, "."+domain) || strings.HasSuffix(domain, "."+site)) {
			score += 2
		}
		if score > bestScore {
			best, bestScore = e, score
		}
	}
	return best
}

func containsAny(s string, tokens []string) bool {
	for _, t := range tokens {
		if strings.Contains(s, t) {
			return true
		}
	}
	return false
}

// contactPage finds the link to a contact page on the same site.
func contactPage(p agent.Page, siteHost string) string {
	for _, l := range p.Links {
		u, err := url.Parse(l.URL)
		if err != nil || !strings.EqualFold(u.Hostname(), siteHost) || u.RawQuery != "" {
			continue
		}
		t := fold(l.Text + " " + u.Path)
		if strings.Contains(t, "contact") || strings.Contains(t, "nous-ecrire") || strings.Contains(t, "mentions") {
			u.Fragment = ""
			return u.String()
		}
	}
	return ""
}

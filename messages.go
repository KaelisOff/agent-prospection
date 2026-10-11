package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/KaelisOff/agent-sdk-go/agent"
)

// Writing the messages. The model of the box writes them from the
// owner's offer; they carry {entreprise} and {ville}, filled in for each
// company. The box adds the sender's identity, the legal notice and
// "reply STOP" itself.

// Angles: what a message leans on. A test pits two angles against each
// other; the winner stays, a new angle challenges it.
var angles = []string{
	"une question directe sur un problème concret que rencontre souvent ce métier",
	"le temps gagné, avec un exemple simple et sans chiffre inventé",
	"une proposition d'échange de dix minutes, très courte",
	"ce que d'autres entreprises du même métier font déjà, sans nommer personne",
	"une offre de premier rendez-vous ou d'essai, sans engagement",
}

func angleFor(n int) string { return angles[n%len(angles)] }

func systemPrompt(cfg Config) string {
	you := "Vouvoie le destinataire."
	if cfg.Tone == "tu" {
		you = "Tutoie le destinataire."
	}
	numbers := "N'écris aucun chiffre."
	if digitsRe.MatchString(cfg.Offer) {
		numbers = "N'utilise que les chiffres présents dans l'offre."
	}
	return "Tu écris, au nom de l'expéditeur, des emails de prospection entre professionnels, en français, pour cette offre de l'expéditeur : " +
		cfg.Offer + "\n" +
		"Règles : 60 à 110 mots ; " + you + " {entreprise} est le nom de l'entreprise DESTINATAIRE (celle à qui tu écris, jamais l'expéditeur) " +
		"et {ville} sa ville ; l'expéditeur parle avec « nous ». " + numbers + " " +
		"Texte brut : pas d'astérisques, pas de gras, pas de titres, pas de lien, pas de pièce jointe, pas de promesse que l'offre ne contient pas, " +
		"pas de signature (elle est ajoutée automatiquement). Termine par une question simple. " +
		"Réponds exactement sous la forme :\nObjet : <objet de 4 à 9 mots>\n\n<corps du message>"
}

var (
	digitsRe   = regexp.MustCompile(`\d`)
	markdownRe = regexp.MustCompile(`\*\*|__|(?m)^#+\s*|(?m)^\s*[*•]\s+`)
	// The recipient taken for the sender: "Chez {entreprise}, nous…".
	confusedRe = regexp.MustCompile(`(?i)chez\s+\{entreprise\}|\{entreprise\}\s*,?\s*(nous|notre)\b|(notre|nos)\s+(société|entreprise|cabinet|équipe)\s+\{entreprise\}|\{entreprise\}\s+(vous\s+)?(propose|accompagne|aide)`)
	subjectRe  = regexp.MustCompile(`(?im)^\s*\**\s*objet\s*:\s*(.+)$`)
	urlRe      = regexp.MustCompile(`(?i)\b(https?://|www\.)\S+`)
	bracesRe   = regexp.MustCompile(`\{[^{}]*\}|\[[^\[\]]*\]`)
	signoffRe  = regexp.MustCompile(`(?im)^\s*(cordialement|bien cordialement|bien à vous|belle journée)[,.!]?\s*$`)
	blankLines = regexp.MustCompile(`\n{3,}`)
)

// parseTemplate reads the model's answer; ok is false when it cannot be
// used: it confuses the recipient with the sender, or has numbers the
// offer does not have (made up).
func parseTemplate(answer, offer string) (Template, bool) {
	m := subjectRe.FindStringSubmatchIndex(answer)
	if m == nil {
		return Template{}, false
	}
	subject := strings.Trim(strings.TrimSpace(answer[m[2]:m[3]]), "*\"« »")
	body := strings.TrimSpace(answer[m[1]:])
	body = strings.Trim(body, "*")
	body = urlRe.ReplaceAllString(body, "")
	// Only the two placeholders stay; any other ([Prénom], {nom}…) goes.
	body = bracesRe.ReplaceAllStringFunc(body, func(s string) string {
		switch strings.ToLower(s) {
		case "{entreprise}":
			return "{entreprise}"
		case "{ville}":
			return "{ville}"
		}
		return ""
	})
	subject = bracesRe.ReplaceAllStringFunc(subject, func(s string) string {
		if strings.EqualFold(s, "{entreprise}") || strings.EqualFold(s, "{ville}") {
			return strings.ToLower(s)
		}
		return ""
	})
	// The box signs: the model's sign-off line goes. Plain text: no
	// markdown.
	body = signoffRe.ReplaceAllString(body, "")
	body = markdownRe.ReplaceAllStringFunc(body, func(m string) string {
		if strings.Contains(m, "*") && strings.TrimSpace(m) != "**" || strings.Contains(m, "•") {
			return "- "
		}
		return ""
	})
	subject = strings.ReplaceAll(strings.ReplaceAll(subject, "**", ""), "__", "")
	if confusedRe.MatchString(body) || confusedRe.MatchString(subject) {
		return Template{}, false
	}
	if !digitsRe.MatchString(offer) && (digitsRe.MatchString(body) || digitsRe.MatchString(subject)) {
		return Template{}, false
	}
	body = strings.ReplaceAll(body, " ,", ",")
	body = strings.TrimSpace(blankLines.ReplaceAllString(body, "\n\n"))
	words := len(strings.Fields(body))
	if subject == "" || utf8.RuneCountInString(subject) > 120 || words < 30 || words > 180 {
		return Template{}, false
	}
	return Template{Subject: strings.TrimSpace(subject), Body: body}, true
}

// fallback is a plain message built from the offer, when the model's
// answers cannot be used.
func fallback(cfg Config) Template {
	hello, you, to := "Bonjour,", "vous", "vous"
	if cfg.Tone == "tu" {
		you, to = "toi", "te"
	}
	offer := strings.TrimSpace(cfg.Offer)
	// "Tenue de…" in the middle of a sentence: "tenue de…" (not "PME…").
	if r := []rune(offer); len(r) > 1 && unicode.IsUpper(r[0]) && unicode.IsLower(r[1]) {
		r[0] = unicode.ToLower(r[0])
		offer = string(r)
	}
	if !strings.HasSuffix(offer, ".") {
		offer += "."
	}
	return Template{
		Subject: "Une proposition pour {entreprise}",
		Body: hello + "\n\nJe me permets d'écrire à {entreprise}, à {ville}, car notre offre pourrait " + to + " être utile : " +
			offer + "\n\nSerait-il possible d'en parler avec " + you + " quelques minutes cette semaine ?\n\nBonne journée,",
		Angle: "message simple",
	}
}

// write asks the model for a message on an angle; fallback if it fails.
func write(box *agent.Box, cfg Config, angle string, avoid *Template) Template {
	user := "Écris un email sur cet angle : " + angle + "."
	if avoid != nil {
		user += "\nIl doit être différent de celui-ci :\nObjet : " + avoid.Subject + "\n\n" + avoid.Body
	}
	for try := 0; try < 3; try++ {
		answer, err := box.ChatMax(400, agent.System(systemPrompt(cfg)), agent.User(user))
		if err != nil {
			box.Logf("model: %v", err)
			break
		}
		if t, ok := parseTemplate(answer, cfg.Offer); ok {
			t.Angle = angle
			return t
		}
		box.Logf("message not usable, try %d", try+1)
	}
	f := fallback(cfg)
	if avoid != nil && avoid.Subject == f.Subject {
		f.Subject = "{entreprise} : une idée à vous soumettre"
	}
	return f
}

// newTest prepares the next test: the previous winner (or a first
// message) against a new angle.
func newTest(box *agent.Box, cfg Config, st *State, winner *Template) Test {
	n := st.TestsDone + st.Rejected
	t := Test{ID: fmt.Sprintf("t%d", st.TestsDone+1)}
	if winner != nil {
		t.A = *winner
	} else {
		t.A = write(box, cfg, angleFor(n), nil)
	}
	t.B = write(box, cfg, angleFor(n+1), &t.A)
	return t
}

// render fills in a template for a company.
func render(t Template, p Prospect) (subject, body string) {
	name := pretty(p.Name)
	city := pretty(p.City)
	r := strings.NewReplacer("{entreprise}", name, "{ville}", city)
	return r.Replace(t.Subject), r.Replace(t.Body)
}

// pretty turns "PLOMBERIE DE CHARONNE" into "Plomberie de Charonne".
func pretty(s string) string {
	small := map[string]bool{"de": true, "du": true, "des": true, "la": true, "le": true, "les": true, "et": true, "en": true, "a": true, "aux": true}
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		if i > 0 && small[w] {
			continue
		}
		if strings.Contains(w, ".") {
			words[i] = strings.ToUpper(w) // S.A.R.L., (A.S.R.)
			continue
		}
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

// describe shows a message to the owner.
func describe(label string, t Template) string {
	return label + "\nObjet : " + t.Subject + "\n\n" + t.Body
}

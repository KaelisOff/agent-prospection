package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/KaelisOff/agent-sdk-go/agent"
)

// --- settings ------------------------------------------------------------------------

// Config is what the owner set in the Agent Center.
type Config struct {
	Offer       string
	Targets     []Target
	Review      bool // each new message waits for the owner's approval
	Daily       int
	Tone        string // "vous" or "tu"
	SoleTraders bool
	// Problems are entries of the settings the agent cannot use.
	Problems []string
}

var (
	nafRe    = regexp.MustCompile(`^\d{2}\.\d{2}[A-Za-z]$`)
	postalRe = regexp.MustCompile(`^\d{5}$`)
	deptRe   = regexp.MustCompile(`^(\d{2,3}|2[ABab])$`)
	slugRe   = regexp.MustCompile(`[^a-z0-9]+`)
)

func loadConfig(s agent.Settings) Config {
	c := Config{
		Offer: strings.TrimSpace(s.String("offer")), Review: s.Bool("review"), Daily: int(s.Number("daily")),
		Tone: s.String("tone"), SoleTraders: s.Bool("sole_traders"),
	}
	if c.Daily <= 0 {
		c.Daily = 20
	}
	if c.Tone == "" {
		c.Tone = "vous"
	}
	var places []string
	for _, p := range s.List("places") {
		p = strings.ToUpper(strings.TrimSpace(p))
		if postalRe.MatchString(p) || deptRe.MatchString(p) {
			places = append(places, p)
		} else {
			c.Problems = append(c.Problems, fmt.Sprintf("lieu « %s » : indiquez un code postal (75011) ou un département (69)", p))
		}
	}
	for _, a := range s.List("activities") {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if nafRe.MatchString(a) {
			a = strings.ToUpper(a)
		}
		for _, p := range places {
			c.Targets = append(c.Targets, Target{Activity: a, Place: p})
		}
	}
	return c
}

// Target is an activity in a place: what the directory is searched for.
type Target struct {
	Activity string `json:"activity"` // NAF code or words
	Place    string `json:"place"`    // postal code or department
}

// Key identifies a target; it is also the campaign of its emails.
func (t Target) Key() string {
	k := strings.Trim(slugRe.ReplaceAllString(fold(t.Activity+"-"+t.Place), "-"), "-")
	if len(k) > 40 {
		k = k[:40]
	}
	return k
}

func (t Target) query(page int) agent.CompanyQuery {
	q := agent.CompanyQuery{Page: page, PerPage: 20}
	if nafRe.MatchString(t.Activity) {
		q.Activities = []string{t.Activity}
	} else {
		q.Text = t.Activity
	}
	if postalRe.MatchString(t.Place) {
		q.PostalCode = t.Place
	} else {
		q.Department = t.Place
	}
	return q
}

// --- state -----------------------------------------------------------------------------

// State is what the agent keeps between runs (storage "state").
type State struct {
	RepliesAfter int64                `json:"replies_after"`
	Progress     map[string]*Progress `json:"progress"`
	NextTarget   int                  `json:"next_target"`
	Test         *Test                `json:"test,omitempty"`
	Pending      *Pending             `json:"pending,omitempty"`
	TestsDone    int                  `json:"tests_done"`
	Rejected     int                  `json:"rejected"` // messages refused this week
	Week         string               `json:"week"`     // ISO week of Rejected
	Day          string               `json:"day"`
	SentToday    int                  `json:"sent_today"`
	SearchDone   string               `json:"search_done,omitempty"` // a day the web quota was used up
	Since        Tally                `json:"since"`                 // since the last report
	Campaigns    map[string]*Tally    `json:"campaigns"`
	LastReport   time.Time            `json:"last_report"`
	LastReview   time.Time            `json:"last_review"`
}

// Progress is where the directory search of a target stands.
type Progress struct {
	Page      int  `json:"page"`
	Pages     int  `json:"pages"`
	Exhausted bool `json:"exhausted"`
}

// Tally counts emails and what came back.
type Tally struct {
	Sent        int `json:"sent"`
	Positive    int `json:"positive"`
	Negative    int `json:"negative"`
	Auto        int `json:"auto"`
	Unsubscribe int `json:"unsubscribe"`
	Bounce      int `json:"bounce"`
}

// Template is a message: subject and body, with {entreprise} and {ville}
// to fill in.
type Template struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	Angle   string `json:"angle,omitempty"`
}

// Test is the A/B test in progress.
type Test struct {
	ID      string    `json:"id"` // "t3"
	A       Template  `json:"a"`
	B       Template  `json:"b"`
	CountsA Counts    `json:"counts_a"`
	CountsB Counts    `json:"counts_b"`
	Started time.Time `json:"started"`
}

// Pending are new messages waiting for the owner's approval.
type Pending struct {
	ApprovalID string    `json:"approval_id"`
	Test       Test      `json:"test"`
	Asked      time.Time `json:"asked"`
}

// Prospect is a company ready to be written to.
type Prospect struct {
	SIREN  string `json:"siren"`
	Name   string `json:"name"`
	City   string `json:"city"`
	Email  string `json:"email"`
	Site   string `json:"site"`
	Target string `json:"target"` // campaign
}

// Seen companies: what came of each (storage "seen").
const (
	seenQueued   = "q" // contact found, waiting to be written to
	seenNoEmail  = "n" // no generic address found
	seenSent     = "s"
	seenSkipped  = "x" // left out (sole trader…)
	seenRejected = "r" // the box refused the address
)

func (s *State) init() {
	if s.Progress == nil {
		s.Progress = map[string]*Progress{}
	}
	if s.Campaigns == nil {
		s.Campaigns = map[string]*Tally{}
	}
}

func (s *State) campaign(key string) *Tally {
	t := s.Campaigns[key]
	if t == nil {
		t = &Tally{}
		s.Campaigns[key] = t
	}
	return t
}

// newDay resets the daily counters.
func (s *State) newDay(now time.Time) {
	day := now.Format("2006-01-02")
	if s.Day != day {
		s.Day, s.SentToday = day, 0
	}
	y, w := now.ISOWeek()
	week := fmt.Sprintf("%d-%02d", y, w)
	if s.Week != week {
		s.Week, s.Rejected = week, 0
	}
}

func load(box *agent.Box) (*State, []Prospect, map[string]string, error) {
	st := &State{}
	if _, err := box.GetJSON("state", st); err != nil {
		return nil, nil, nil, err
	}
	st.init()
	var queue []Prospect
	if _, err := box.GetJSON("queue", &queue); err != nil {
		return nil, nil, nil, err
	}
	seen := map[string]string{}
	if _, err := box.GetJSON("seen", &seen); err != nil {
		return nil, nil, nil, err
	}
	return st, queue, seen, nil
}

func save(box *agent.Box, st *State, queue []Prospect, seen map[string]string) error {
	if err := box.SetJSON("state", st); err != nil {
		return err
	}
	if err := box.SetJSON("queue", queue); err != nil {
		return err
	}
	return box.SetJSON("seen", seen)
}

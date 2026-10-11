package main

import (
	"net/url"
	"strings"
	"time"

	"github.com/KaelisOff/agent-sdk-go/agent"
)

// Finding companies to write to: the official directory gives the
// companies of a target, a web search their own site, their site a
// generic contact address. Everything goes through the box's tools.

// budget keeps a run within the box's limits: 200 calls, the run's
// duration.
type budget struct {
	calls    int
	maxCalls int
	deadline time.Time
}

func (b *budget) spend(n int) bool {
	if b.calls+n > b.maxCalls || time.Now().After(b.deadline) {
		return false
	}
	b.calls += n
	return true
}

// fill adds prospects to the queue until it holds want, the targets are
// exhausted, the web quota is used up or the run's budget is spent.
func fill(box *agent.Box, cfg Config, st *State, queue []Prospect, seen map[string]string, want int, bud *budget, today string) []Prospect {
	if len(cfg.Targets) == 0 || st.SearchDone == today {
		return queue
	}
	tried := 0
	for len(queue) < want && tried < len(cfg.Targets) {
		t := cfg.Targets[st.NextTarget%len(cfg.Targets)]
		pr := st.Progress[t.Key()]
		if pr == nil {
			pr = &Progress{}
			st.Progress[t.Key()] = pr
		}
		if pr.Exhausted {
			st.NextTarget++
			tried++
			continue
		}
		if !bud.spend(1) {
			return queue
		}
		// If this page cannot be finished, it is read again later (the
		// companies already dealt with are in seen).
		before := *pr
		page, err := box.SearchCompanies(t.query(pr.Page + 1))
		if err != nil {
			box.Logf("directory: %v", err)
			if agent.IsCode(err, agent.CodeQuota) {
				st.SearchDone = today
			}
			return queue
		}
		pr.Page, pr.Pages = pr.Page+1, page.Pages
		if pr.Page >= page.Pages {
			pr.Exhausted = true
		}
		found := 0
		for _, c := range page.Companies {
			if _, done := seen[c.SIREN]; done {
				continue
			}
			if c.SoleTrader && !cfg.SoleTraders {
				seen[c.SIREN] = seenSkipped
				continue
			}
			p, stop := contactOf(box, c, t.Key(), bud)
			switch stop {
			case stopQuota:
				st.SearchDone = today // the web quota is used up for today
				*pr = before
				return queue
			case stopBudget, stopUnavailable:
				*pr = before
				return queue
			}
			if p == nil {
				seen[c.SIREN] = seenNoEmail
				continue
			}
			seen[c.SIREN] = seenQueued
			queue = append(queue, *p)
			found++
			if len(queue) >= want {
				// The rest of the page waits for the next search.
				*pr = before
				break
			}
		}
		box.Logf("target %s page %d: %d new contacts", t.Key(), pr.Page, found)
		// One page per target in turn: every target gets its chance.
		st.NextTarget++
		tried = 0
	}
	return queue
}

// Why contactOf stopped.
const (
	stopQuota  = "quota"  // the day's web quota is used up
	stopBudget = "budget" // this run's calls or time are
	// stopUnavailable: the web is not there (plugin turned off, service
	// down); the company is tried again later.
	stopUnavailable = "unavailable"
)

// contactOf looks for a company's generic address.
func contactOf(box *agent.Box, c agent.Company, campaign string, bud *budget) (p *Prospect, stop string) {
	name := c.Name
	if c.TradeName != "" {
		name = c.TradeName
	}
	query := strings.Join(strings.Fields(name+" "+pretty(c.City)), " ")
	if words := strings.Fields(query); len(words) > 10 {
		query = strings.Join(words[:10], " ")
	}
	if !bud.spend(3) {
		return nil, stopBudget
	}
	results, err := box.WebSearch(query, 5)
	if err != nil {
		if agent.IsCode(err, agent.CodeQuota) {
			return nil, stopQuota
		}
		if agent.IsCode(err, agent.CodePluginDisabled) || agent.IsCode(err, agent.CodeFailed) {
			box.Logf("web unavailable: %v", err)
			return nil, stopUnavailable
		}
		box.Logf("search %q: %v", query, err)
		return nil, ""
	}
	site, ok := chooseSite(results, c.Name, c.TradeName)
	if !ok {
		return nil, ""
	}
	u, _ := url.Parse(site)
	home, err := box.ReadPage(site)
	if err != nil {
		if agent.IsCode(err, agent.CodeQuota) {
			return nil, stopQuota
		}
		if agent.IsCode(err, agent.CodePluginDisabled) {
			return nil, stopUnavailable
		}
		return nil, ""
	}
	if !aboutCompany(home, c) {
		box.Logf("site %s left: it does not look like %s's", u.Hostname(), name)
		return nil, ""
	}
	email := pickContact(emailsOf(home), u.Hostname(), c.SoleTrader, c.Name, c.TradeName)
	if email == "" {
		if contact := contactPage(home, u.Hostname()); contact != "" {
			if page, err := box.ReadPage(contact); err == nil {
				email = pickContact(emailsOf(page), u.Hostname(), c.SoleTrader, c.Name, c.TradeName)
			} else if agent.IsCode(err, agent.CodeQuota) {
				return nil, stopQuota
			}
		}
	}
	if email == "" {
		return nil, ""
	}
	box.Logf("contact: %s <%s> (%s)", name, email, u.Hostname())
	return &Prospect{SIREN: c.SIREN, Name: name, City: c.City, Email: email, Site: site, Target: campaign}, ""
}

// Package classify labels what a sensor saw. Each rule reads an event's fields
// — the user agent, the path, the credentials tried, the first bytes sent — and
// adds a tag with a severity, so a feed of raw connections reads as a list of
// recognizable behaviours.
package classify

import (
	"sort"
	"strings"

	"github.com/cansarihan/lure/internal/event"
)

// Severity levels.
const (
	Info   = 1
	Notice = 2
	Alert  = 3
)

// Rule tags an event when it matches.
type Rule struct {
	Tag         string `json:"tag"`
	Severity    int    `json:"severity"`
	Description string `json:"description"`
	match       func(e event.Event) bool
}

var rules []Rule

func add(tag string, severity int, description string, match func(event.Event) bool) {
	rules = append(rules, Rule{Tag: tag, Severity: severity, Description: description, match: match})
}

// Rules returns the catalogue, most severe first.
func Rules() []Rule {
	out := append([]Rule(nil), rules...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Severity != out[j].Severity {
			return out[i].Severity > out[j].Severity
		}
		return out[i].Tag < out[j].Tag
	})
	return out
}

// Apply sets an event's tags and its severity, which is the highest of the
// rules it matched, or Info when none did.
func Apply(e event.Event) event.Event {
	e.Tags = nil
	e.Severity = Info
	for _, r := range rules {
		if r.match(e) {
			e.Tags = append(e.Tags, r.Tag)
			if r.Severity > e.Severity {
				e.Severity = r.Severity
			}
		}
	}
	sort.Strings(e.Tags)
	return e
}

func field(e event.Event, name string) string { return e.Fields[name] }

func containsAny(s string, needles ...string) bool {
	s = strings.ToLower(s)
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

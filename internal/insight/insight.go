// Package insight turns a stream of events into the views a person reads: who
// is knocking, what they try, which doors they try it on, and when.
package insight

import (
	"sort"
	"time"

	"github.com/cansarihan/lure/internal/event"
)

// Source is everything one remote address did.
type Source struct {
	Remote   string         `json:"remote"`
	First    time.Time      `json:"first"`
	Last     time.Time      `json:"last"`
	Events   int            `json:"events"`
	Logins   int            `json:"logins"`
	Severity int            `json:"severity"`
	Sensors  []string       `json:"sensors"`
	Tags     map[string]int `json:"tags"`
}

// Count is a value and how often it was seen.
type Count struct {
	Value string `json:"value"`
	N     int    `json:"n"`
}

// Summary is the rolled-up view of a set of events.
type Summary struct {
	Events     int            `json:"events"`
	Sources    []Source       `json:"sources"`
	BySensor   map[string]int `json:"by_sensor"`
	BySeverity map[int]int    `json:"by_severity"`
	Tags       []Count        `json:"tags"`
	Usernames  []Count        `json:"usernames"`
	Passwords  []Count        `json:"passwords"`
	Pairs      []Count        `json:"pairs"`
	Paths      []Count        `json:"paths"`
	Clients    []Count        `json:"clients"`
	Hourly     []int          `json:"hourly"` // last 24 hours, oldest first
}

// Summarize rolls events up. now anchors the hourly histogram; top limits each
// ranked list.
func Summarize(events []event.Event, now time.Time, top int) Summary {
	s := Summary{
		Events:     len(events),
		BySensor:   map[string]int{},
		BySeverity: map[int]int{},
		Hourly:     make([]int, 24),
	}
	sources := map[string]*Source{}
	sensorSets := map[string]map[string]bool{}
	tags, users, passwords, pairs, paths, clients := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}

	for _, e := range events {
		s.BySensor[e.Sensor]++
		s.BySeverity[e.Severity]++
		if age := now.Sub(e.Time); age >= 0 && age < 24*time.Hour {
			s.Hourly[23-int(age/time.Hour)]++
		}

		src, ok := sources[e.Remote]
		if !ok {
			src = &Source{Remote: e.Remote, First: e.Time, Last: e.Time, Tags: map[string]int{}}
			sources[e.Remote] = src
			sensorSets[e.Remote] = map[string]bool{}
		}
		src.Events++
		if e.Time.Before(src.First) {
			src.First = e.Time
		}
		if e.Time.After(src.Last) {
			src.Last = e.Time
		}
		if e.Severity > src.Severity {
			src.Severity = e.Severity
		}
		sensorSets[e.Remote][e.Sensor] = true
		for _, t := range e.Tags {
			src.Tags[t]++
			tags[t]++
		}

		switch e.Kind {
		case event.KindLogin:
			src.Logins++
			users[e.Fields["user"]]++
			passwords[e.Fields["password"]]++
			pairs[e.Fields["user"]+" / "+e.Fields["password"]]++
		case event.KindRequest:
			if p := e.Fields["path"]; p != "" {
				paths[p]++
			}
		}
		if c := e.Fields["client"]; c != "" {
			clients[c]++
		} else if ua := e.Fields["ua"]; ua != "" && e.Kind == event.KindRequest {
			clients[ua]++
		}
	}

	for remote, src := range sources {
		for name := range sensorSets[remote] {
			src.Sensors = append(src.Sensors, name)
		}
		sort.Strings(src.Sensors)
		s.Sources = append(s.Sources, *src)
	}
	sort.Slice(s.Sources, func(i, j int) bool {
		a, b := s.Sources[i], s.Sources[j]
		if a.Severity != b.Severity {
			return a.Severity > b.Severity
		}
		if a.Events != b.Events {
			return a.Events > b.Events
		}
		return a.Remote < b.Remote
	})

	s.Tags = rank(tags, top)
	s.Usernames = rank(users, top)
	s.Passwords = rank(passwords, top)
	s.Pairs = rank(pairs, top)
	s.Paths = rank(paths, top)
	s.Clients = rank(clients, top)
	return s
}

func rank(m map[string]int, top int) []Count {
	out := make([]Count, 0, len(m))
	for v, n := range m {
		out = append(out, Count{Value: v, N: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].Value < out[j].Value
	})
	if top > 0 && len(out) > top {
		out = out[:top]
	}
	return out
}

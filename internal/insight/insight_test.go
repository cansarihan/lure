package insight

import (
	"testing"
	"time"

	"github.com/cansarihan/lure/internal/event"
)

func TestSummarize(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	events := []event.Event{
		{Time: now.Add(-30 * time.Minute), Remote: "10.0.0.1", Sensor: "telnet", Kind: event.KindLogin, Severity: 3,
			Fields: map[string]string{"user": "root", "password": "root"}, Tags: []string{"default-credentials"}},
		{Time: now.Add(-20 * time.Minute), Remote: "10.0.0.1", Sensor: "telnet", Kind: event.KindLogin, Severity: 3,
			Fields: map[string]string{"user": "root", "password": "123456"}, Tags: []string{"default-credentials"}},
		{Time: now.Add(-5 * time.Hour), Remote: "10.0.0.2", Sensor: "http", Kind: event.KindRequest, Severity: 2,
			Fields: map[string]string{"path": "/.env", "ua": "zgrab"}, Tags: []string{"config-probe"}},
		{Time: now.Add(-48 * time.Hour), Remote: "10.0.0.3", Sensor: "ssh", Kind: event.KindConnect, Severity: 1,
			Fields: map[string]string{"client": "SSH-2.0-Go"}},
	}
	s := Summarize(events, now, 10)
	if s.Events != 4 || len(s.Sources) != 3 {
		t.Fatalf("events=%d sources=%d", s.Events, len(s.Sources))
	}
	top := s.Sources[0]
	if top.Remote != "10.0.0.1" || top.Logins != 2 || top.Severity != 3 || top.Tags["default-credentials"] != 2 {
		t.Errorf("top source = %+v", top)
	}
	if s.Usernames[0] != (Count{"root", 2}) || len(s.Pairs) != 2 {
		t.Errorf("usernames=%v pairs=%v", s.Usernames, s.Pairs)
	}
	if s.Paths[0].Value != "/.env" || s.BySensor["telnet"] != 2 {
		t.Errorf("paths=%v sensors=%v", s.Paths, s.BySensor)
	}
	if s.Hourly[23] != 2 || s.Hourly[18] != 1 {
		t.Errorf("hourly = %v", s.Hourly)
	}
	sum := 0
	for _, n := range s.Hourly {
		sum += n
	}
	if sum != 3 {
		t.Errorf("an event older than a day leaked into the histogram: %v", s.Hourly)
	}
}

func TestRankLimit(t *testing.T) {
	got := rank(map[string]int{"a": 1, "b": 3, "c": 2}, 2)
	if len(got) != 2 || got[0].Value != "b" || got[1].Value != "c" {
		t.Errorf("rank = %v", got)
	}
}

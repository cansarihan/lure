package event

import (
	"testing"
	"time"
)

func TestRingKeepsNewestFirst(t *testing.T) {
	r := NewRing(3)
	for _, s := range []string{"a", "b", "c", "d"} {
		r.Add(Event{Summary: s})
	}
	got := r.Recent(0)
	if len(got) != 3 || got[0].Summary != "d" || got[2].Summary != "b" {
		t.Fatalf("recent = %+v", got)
	}
	if got[0].ID != 4 || got[0].Time.IsZero() {
		t.Errorf("event not stamped: %+v", got[0])
	}
	if len(r.Recent(2)) != 2 {
		t.Error("Recent(2) should return two events")
	}
}

func TestSubscribe(t *testing.T) {
	r := NewRing(8)
	ch, stop := r.Subscribe()
	r.Add(Event{Summary: "hello"})
	select {
	case e := <-ch:
		if e.Summary != "hello" {
			t.Errorf("got %q", e.Summary)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber received nothing")
	}
	stop()
	if _, open := <-ch; open {
		t.Error("channel should close after stop")
	}
	r.Add(Event{Summary: "after"}) // must not panic on a closed subscriber
}

// Package event defines what a sensor records and keeps the recent record in a
// bounded ring that the panel can follow live.
package event

import (
	"sync"
	"time"
)

// Kind says what happened on a connection.
type Kind string

const (
	KindConnect Kind = "connect"
	KindRequest Kind = "request"
	KindLogin   Kind = "login"
	KindPayload Kind = "payload"
)

// Event is one observation by a sensor.
type Event struct {
	ID       uint64            `json:"id"`
	Time     time.Time         `json:"time"`
	Sensor   string            `json:"sensor"`
	Port     int               `json:"port"`
	Remote   string            `json:"remote"`
	Kind     Kind              `json:"kind"`
	Summary  string            `json:"summary"`
	Fields   map[string]string `json:"fields,omitempty"`
	Tags     []string          `json:"tags,omitempty"`
	Severity int               `json:"severity"`
}

// Ring keeps the most recent events and fans new ones out to subscribers.
type Ring struct {
	mu     sync.Mutex
	buf    []Event
	head   int
	size   int
	nextID uint64
	subs   map[int]chan Event
	seq    int
}

// NewRing returns a ring that remembers capacity events.
func NewRing(capacity int) *Ring {
	if capacity < 1 {
		capacity = 1
	}
	return &Ring{buf: make([]Event, capacity), subs: map[int]chan Event{}}
}

// Add stamps an event with an id (and a time if it has none), stores it, and
// delivers it to every subscriber that is keeping up.
func (r *Ring) Add(e Event) Event {
	r.mu.Lock()
	r.nextID++
	e.ID = r.nextID
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}
	r.buf[r.head] = e
	r.head = (r.head + 1) % len(r.buf)
	if r.size < len(r.buf) {
		r.size++
	}
	targets := make([]chan Event, 0, len(r.subs))
	for _, ch := range r.subs {
		targets = append(targets, ch)
	}
	r.mu.Unlock()
	for _, ch := range targets {
		select {
		case ch <- e:
		default:
		}
	}
	return e
}

// Recent returns up to n events, newest first.
func (r *Ring) Recent(n int) []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n <= 0 || n > r.size {
		n = r.size
	}
	out := make([]Event, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, r.buf[(r.head-i+len(r.buf))%len(r.buf)])
	}
	return out
}

// Subscribe returns a channel of new events and a function that ends the
// subscription.
func (r *Ring) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, 256)
	r.mu.Lock()
	r.seq++
	id := r.seq
	r.subs[id] = ch
	r.mu.Unlock()
	return ch, func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if c, ok := r.subs[id]; ok {
			delete(r.subs, id)
			close(c)
		}
	}
}

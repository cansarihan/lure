// Package recorder is where every sensor's observations end up: each event is
// classified, kept in the live ring and, when a journal is open, written to disk.
package recorder

import (
	"log/slog"

	"github.com/cansarihan/lure/internal/classify"
	"github.com/cansarihan/lure/internal/event"
	"github.com/cansarihan/lure/internal/journal"
)

// Recorder classifies and stores events.
type Recorder struct {
	Ring    *event.Ring
	journal *journal.Writer
	log     *slog.Logger
}

// New builds a recorder over a ring and an optional journal.
func New(ring *event.Ring, j *journal.Writer, log *slog.Logger) *Recorder {
	return &Recorder{Ring: ring, journal: j, log: log}
}

// Record is a sensor.Sink.
func (r *Recorder) Record(e event.Event) {
	e = r.Ring.Add(classify.Apply(e))
	if r.journal != nil {
		if err := r.journal.Write(e); err != nil && r.log != nil {
			r.log.Error("journal write failed", "error", err)
		}
	}
}

// Restore replays events from a journal into the ring, classifying them again
// so rule changes apply to old records too.
func (r *Recorder) Restore(events []event.Event) {
	for _, e := range events {
		t := e.Time
		e = classify.Apply(e)
		e.Time = t
		r.Ring.Add(e)
	}
}

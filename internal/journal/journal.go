// Package journal appends events to a JSON Lines file so a restart keeps the
// history, and reads that file back.
package journal

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/cansarihan/lure/internal/event"
)

// Writer appends events to a file, one JSON object per line.
type Writer struct {
	mu sync.Mutex
	f  *os.File
}

// Open opens path for appending, creating it if needed.
func Open(path string) (*Writer, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600) // #nosec G304 -- operator-chosen journal path
	if err != nil {
		return nil, err
	}
	return &Writer{f: f}, nil
}

// Write appends one event.
func (w *Writer) Write(e event.Event) error {
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = w.f.Write(append(line, '\n'))
	return err
}

// Close closes the file.
func (w *Writer) Close() error { return w.f.Close() }

// Read loads every event in a journal file. A missing file is an empty journal;
// a torn last line, left by a crash mid-write, is skipped.
func Read(path string) ([]event.Event, error) {
	f, err := os.Open(path) // #nosec G304 -- operator-chosen journal path
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return decode(f)
}

func decode(r io.Reader) ([]event.Event, error) {
	var out []event.Event
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var e event.Event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			continue
		}
		out = append(out, e)
	}
	return out, sc.Err()
}

package journal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cansarihan/lure/internal/event"
)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	w, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"one", "two"} {
		if err := w.Write(event.Event{Sensor: "http", Summary: s}); err != nil {
			t.Fatal(err)
		}
	}
	_ = w.Close()
	// a torn trailing line must not break the read
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	_, _ = f.WriteString(`{"sensor":"ht`)
	_ = f.Close()

	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].Summary != "two" {
		t.Fatalf("read %+v", got)
	}
}

func TestReadMissing(t *testing.T) {
	got, err := Read(filepath.Join(t.TempDir(), "absent.jsonl"))
	if err != nil || len(got) != 0 {
		t.Fatalf("missing journal: %v %v", got, err)
	}
}

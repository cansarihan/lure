package server

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cansarihan/lure/internal/event"
	"github.com/cansarihan/lure/internal/insight"
	"github.com/cansarihan/lure/internal/recorder"
	"github.com/cansarihan/lure/internal/seed"
)

func testServer(t *testing.T, token string) (*httptest.Server, *recorder.Recorder) {
	t.Helper()
	ring := event.NewRing(2000)
	rec := recorder.New(ring, nil, nil)
	for _, e := range seed.Events(time.Now(), 400) {
		rec.Record(e)
	}
	srv := httptest.NewServer(New(Options{Ring: ring, Token: token, UI: true}, BuildInfo{Version: "test"}).Handler())
	t.Cleanup(srv.Close)
	return srv, rec
}

func get(t *testing.T, url string, v any) int {
	t.Helper()
	res, err := http.Get(url) // #nosec G107 -- url is the httptest server under test
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if v != nil {
		_ = json.NewDecoder(res.Body).Decode(v)
	}
	return res.StatusCode
}

func TestSummaryAndFilters(t *testing.T) {
	srv, _ := testServer(t, "")
	var sum insight.Summary
	if code := get(t, srv.URL+"/api/v1/summary", &sum); code != http.StatusOK || sum.Events != 400 {
		t.Fatalf("summary %d events=%d", code, sum.Events)
	}
	if len(sum.Sources) == 0 || len(sum.Pairs) == 0 || len(sum.Tags) == 0 {
		t.Errorf("summary lacks rollups: %+v", sum)
	}
	var alerts []event.Event
	get(t, srv.URL+"/api/v1/events?severity=3&limit=50", &alerts)
	if len(alerts) == 0 || len(alerts) > 50 {
		t.Fatalf("alerts = %d", len(alerts))
	}
	for _, e := range alerts {
		if e.Severity < 3 {
			t.Fatalf("severity filter leaked %+v", e)
		}
	}
	var telnet []event.Event
	get(t, srv.URL+"/api/v1/events?sensor=telnet&tag=default-credentials", &telnet)
	for _, e := range telnet {
		if e.Sensor != "telnet" || !hasTag(e, "default-credentials") {
			t.Fatalf("filter leaked %+v", e)
		}
	}
}

func TestSource(t *testing.T) {
	srv, _ := testServer(t, "")
	var body struct {
		Source insight.Source `json:"source"`
		Events []event.Event  `json:"events"`
	}
	if code := get(t, srv.URL+"/api/v1/sources/198.51.100.23", &body); code != http.StatusOK {
		t.Fatalf("source = %d", code)
	}
	if body.Source.Remote != "198.51.100.23" || body.Source.Logins == 0 || len(body.Events) != body.Source.Events {
		t.Errorf("source = %+v (%d events)", body.Source, len(body.Events))
	}
	if code := get(t, srv.URL+"/api/v1/sources/192.0.2.250", nil); code != http.StatusNotFound {
		t.Errorf("unknown source = %d", code)
	}
}

func TestStream(t *testing.T) {
	srv, rec := testServer(t, "")
	res, err := http.Get(srv.URL + "/api/v1/stream") // #nosec G107 -- httptest server
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	r := bufio.NewReader(res.Body)
	_, _ = r.ReadString('\n') // ": connected"
	go rec.Record(event.Event{Sensor: "ssh", Remote: "192.0.2.1", Kind: event.KindConnect, Summary: "live"})
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(line, "data: ") && strings.Contains(line, `"summary":"live"`) {
			return
		}
	}
	t.Fatal("live event never streamed")
}

func TestTokenAndCache(t *testing.T) {
	srv, _ := testServer(t, "s3cret")
	if code := get(t, srv.URL+"/api/v1/rules", nil); code != http.StatusUnauthorized {
		t.Errorf("no token = %d", code)
	}
	var rules []map[string]any
	if code := get(t, srv.URL+"/api/v1/rules?token=s3cret", &rules); code != http.StatusOK || len(rules) < 15 {
		t.Errorf("rules = %d (%d)", code, len(rules))
	}
	res, _ := http.Get(srv.URL + "/") // #nosec G107 -- httptest server
	_ = res.Body.Close()
	if res.Header.Get("Cache-Control") != "no-cache" {
		t.Error("panel should not be cached")
	}
}

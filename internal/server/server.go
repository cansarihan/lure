// Package server exposes what the sensors recorded over an HTTP API and serves
// the embedded panel. It is meant for the operator, so it listens on its own
// address, apart from the decoys, and can require a token.
package server

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cansarihan/lure/internal/classify"
	"github.com/cansarihan/lure/internal/event"
	"github.com/cansarihan/lure/internal/insight"
	"github.com/cansarihan/lure/internal/sensor"
)

// BuildInfo is the version metadata reported on the status endpoint.
type BuildInfo struct {
	Version   string
	Commit    string
	GoVersion string
	Platform  string
}

// Options configures the server.
type Options struct {
	Ring      *event.Ring
	Listeners []sensor.Bound
	Token     string
	UI        bool
}

// Server answers the panel's questions about the recorded events.
type Server struct {
	opts    Options
	info    BuildInfo
	started time.Time
	now     func() time.Time
}

// New builds a server.
func New(opts Options, info BuildInfo) *Server {
	return &Server{opts: opts, info: info, started: time.Now(), now: time.Now}
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("GET /api/v1/status", s.guard(s.handleStatus))
	mux.HandleFunc("GET /api/v1/summary", s.guard(s.handleSummary))
	mux.HandleFunc("GET /api/v1/events", s.guard(s.handleEvents))
	mux.HandleFunc("GET /api/v1/sources/{remote}", s.guard(s.handleSource))
	mux.HandleFunc("GET /api/v1/rules", s.guard(s.handleRules))
	mux.HandleFunc("GET /api/v1/stream", s.guard(s.handleStream))
	if s.opts.UI {
		mux.Handle("GET /", uiHandler())
	}
	return mux
}

func (s *Server) guard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.opts.Token != "" {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if got == "" {
				got = r.URL.Query().Get("token")
			}
			if subtle.ConstantTimeCompare([]byte(got), []byte(s.opts.Token)) != 1 {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version":    s.info.Version,
		"commit":     s.info.Commit,
		"go_version": s.info.GoVersion,
		"platform":   s.info.Platform,
		"uptime":     time.Since(s.started).Round(time.Second).String(),
		"listeners":  s.opts.Listeners,
		"events":     len(s.opts.Ring.Recent(0)),
	})
}

func (s *Server) handleSummary(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, insight.Summarize(s.opts.Ring.Recent(0), s.now(), 15))
}

// handleEvents returns recent events, newest first, optionally narrowed by
// sensor, minimum severity, remote address or tag.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	minSeverity, _ := strconv.Atoi(q.Get("severity"))
	out := make([]event.Event, 0, limit)
	for _, e := range s.opts.Ring.Recent(0) {
		if q.Get("sensor") != "" && e.Sensor != q.Get("sensor") {
			continue
		}
		if e.Severity < minSeverity {
			continue
		}
		if q.Get("remote") != "" && e.Remote != q.Get("remote") {
			continue
		}
		if tag := q.Get("tag"); tag != "" && !hasTag(e, tag) {
			continue
		}
		out = append(out, e)
		if len(out) == limit {
			break
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func hasTag(e event.Event, tag string) bool {
	for _, t := range e.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (s *Server) handleSource(w http.ResponseWriter, r *http.Request) {
	remote := r.PathValue("remote")
	var events []event.Event
	for _, e := range s.opts.Ring.Recent(0) {
		if e.Remote == remote {
			events = append(events, e)
		}
	}
	if len(events) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no events from " + remote})
		return
	}
	sum := insight.Summarize(events, s.now(), 10)
	writeJSON(w, http.StatusOK, map[string]any{"source": sum.Sources[0], "summary": sum, "events": events})
}

func (s *Server) handleRules(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, classify.Rules())
}

// handleStream sends each new event as a server-sent event.
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	events, stop := s.opts.Ring.Subscribe()
	defer stop()
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	beat := time.NewTicker(20 * time.Second)
	defer beat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case e, ok := <-events:
			if !ok {
				return
			}
			data, _ := json.Marshal(e)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-beat.C:
			_, _ = fmt.Fprint(w, ": keep alive\n\n")
			flusher.Flush()
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/cansarihan/lure/internal/event"
	"github.com/cansarihan/lure/internal/journal"
	"github.com/cansarihan/lure/internal/logging"
	"github.com/cansarihan/lure/internal/recorder"
	"github.com/cansarihan/lure/internal/seed"
	"github.com/cansarihan/lure/internal/sensor"
	"github.com/cansarihan/lure/internal/server"
)

var defaultSensors = []string{"http=:8080", "ssh=:2222", "telnet=:2323", "raw:redis=:6379"}

func runSensors(args []string) error {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	var specs multiFlag
	flags.Var(&specs, "sensor", "decoy to run as sensor=addr, repeatable (http, ssh, telnet, raw:label)")
	panel := flags.String("panel", "127.0.0.1:9000", "address for the operator panel; empty disables it")
	journalPath := flags.String("journal", "", "append events to this JSON Lines file and reload it at start")
	keep := flags.Int("keep", 20000, "events kept in memory")
	demo := flags.Int("seed", 0, "preload this many synthetic events for a demo")
	token := flags.String("token", os.Getenv("LURE_TOKEN"), "require this token on the panel")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if len(specs) == 0 {
		specs = defaultSensors
	}
	parsed := make([]sensor.Spec, 0, len(specs))
	for _, s := range specs {
		spec, err := sensor.ParseSpec(s)
		if err != nil {
			return err
		}
		parsed = append(parsed, spec)
	}

	log := logging.New("info", "text", os.Stderr)
	ctx, stop := signalContext()
	defer stop()

	ring := event.NewRing(*keep)
	var j *journal.Writer
	rec := recorder.New(ring, nil, log)
	if *journalPath != "" {
		past, err := journal.Read(*journalPath)
		if err != nil {
			return err
		}
		rec.Restore(past)
		if j, err = journal.Open(*journalPath); err != nil {
			return err
		}
		defer func() { _ = j.Close() }()
		rec = recorder.New(ring, j, log)
		log.Info("journal loaded", "path", *journalPath, "events", len(past))
	}
	if *demo > 0 {
		demoRec := recorder.New(ring, nil, log) // synthetic events stay out of the journal
		for _, e := range seed.Events(time.Now(), *demo) {
			demoRec.Record(e)
		}
		log.Info("seeded synthetic events", "count", *demo)
	}

	bound, wait, err := sensor.Start(ctx, parsed, rec.Record)
	if err != nil {
		return err
	}
	for _, b := range bound {
		log.Info("sensor listening", "sensor", b.Spec.Sensor, "addr", b.Addr)
	}

	if *panel != "" {
		handler := server.New(server.Options{Ring: ring, Listeners: bound, Token: *token, UI: true}, server.BuildInfo{
			Version: version, Commit: commit, GoVersion: runtime.Version(), Platform: runtime.GOOS + "/" + runtime.GOARCH,
		}).Handler()
		srv := &http.Server{Addr: *panel, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
		go func() {
			log.Info("panel listening", "addr", *panel)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error("panel stopped", "error", err)
				stop()
			}
		}()
		defer func() {
			shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdown)
		}()
	}

	<-ctx.Done()
	log.Info("shutting down")
	wait()
	return nil
}

// Command lure runs decoy network services that record what connecting clients
// try, and serves a panel that makes sense of it.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "lure: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return usage()
	}
	args := os.Args[2:]
	switch os.Args[1] {
	case "run":
		return runSensors(args)
	case "report":
		return runReport(args)
	case "rules":
		return runRules(args)
	case "version":
		fmt.Printf("lure %s (commit %s, built %s)\n", version, commit, date)
		return nil
	default:
		return usage()
	}
}

func usage() error {
	fmt.Fprintln(os.Stderr, "usage: lure <run|report|rules|version> [flags]")
	fmt.Fprintln(os.Stderr, "  lure run -sensor http=:8080 -sensor telnet=:2323 -panel 127.0.0.1:9000")
	fmt.Fprintln(os.Stderr, "  lure report -journal events.jsonl")
	return errors.New("no command given")
}

func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}

// multiFlag collects a repeatable flag.
type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

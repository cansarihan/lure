package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/cansarihan/lure/internal/classify"
	"github.com/cansarihan/lure/internal/insight"
	"github.com/cansarihan/lure/internal/journal"
)

func runReport(args []string) error {
	flags := flag.NewFlagSet("report", flag.ContinueOnError)
	path := flags.String("journal", "", "journal file written by lure run")
	top := flags.Int("top", 10, "rows per ranking")
	asJSON := flags.Bool("json", false, "print JSON")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("report needs -journal")
	}
	events, err := journal.Read(*path)
	if err != nil {
		return err
	}
	for i := range events {
		t := events[i].Time
		events[i] = classify.Apply(events[i])
		events[i].Time = t
	}
	sum := insight.Summarize(events, time.Now(), *top)
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(sum)
	}

	fmt.Printf("%d events from %d sources\n\n", sum.Events, len(sum.Sources))
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SOURCE\tEVENTS\tLOGINS\tSENSORS\tLAST SEEN")
	for i, s := range sum.Sources {
		if i == *top {
			break
		}
		fmt.Fprintf(tw, "%s\t%d\t%d\t%v\t%s\n", s.Remote, s.Events, s.Logins, s.Sensors, s.Last.Format(time.RFC3339))
	}
	_ = tw.Flush()
	section := func(title string, rows []insight.Count) {
		if len(rows) == 0 {
			return
		}
		fmt.Printf("\n%s\n", title)
		for _, c := range rows {
			fmt.Printf("  %5d  %s\n", c.N, c.Value)
		}
	}
	section("Behaviours", sum.Tags)
	section("Credentials tried", sum.Pairs)
	section("Paths requested", sum.Paths)
	section("Clients", sum.Clients)
	return nil
}

package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/cansarihan/lure/internal/classify"
)

var severityName = map[int]string{classify.Info: "info", classify.Notice: "notice", classify.Alert: "alert"}

func runRules(_ []string) error {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SEVERITY\tTAG\tMEANING")
	for _, r := range classify.Rules() {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", severityName[r.Severity], r.Tag, r.Description)
	}
	return tw.Flush()
}

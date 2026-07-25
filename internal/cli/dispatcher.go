// dispatcher.go: maps the command word (init, analyze, plan, run,
// onboard) to its handler. Owns the shared lifecycle around every
// command: parse, load config for the commands that declare they need
// it, call the app entry point, hand the returned data to render, and
// map errors to exit codes (0 = clean, 1 = error, 2 = changes pending).
// Handlers stay thin - routing only (analyze's handler switches on
// spec.Mode to pick its app entry point); all real logic lives in
// internal/app, all printing in render.
//
// onboard is a pure generator: converting an unpartitioned table is a
// blocking rebuild, so Horus produces the complete ALTER (boundaries
// resolved, retention collapse applied) plus a size/duration warning and
// a gh-ost/pt-osc alternative - and executes nothing. The operator runs
// it on their own terms; the next cycle detects the now-partitioned table
// and adopts it (journaling an adoption baseline for drift detection).
// analyze/plan/run emit nothing for unpartitioned tables.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/RTolkachev/horus/internal/app"
	"github.com/RTolkachev/horus/internal/config"
	"github.com/RTolkachev/horus/internal/domain"
)

func Run(ctx context.Context, args []string) int {
	spec, err := Parse(args, os.Getenv, os.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // user asked for help and got it
		}
		fmt.Fprintf(os.Stderr, "horus: %v\n", err)
		return 1
	}

	cnf, err := config.Load(spec.ConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}

	switch spec.Command {
	case "init":
		err = app.Init(ctx, spec.DSN)
		if err != nil {
			fmt.Fprintf(os.Stderr, "horus: %v\n", err)
			return 0
		}
		fmt.Fprintf(os.Stdout, "Initialized")
	case "analyze":
		var layouts []domain.PartitionLayout
		switch spec.Mode {
		case "table":
			layouts, err = app.AnalyzeTable(ctx, spec.DSN, spec.Flags["table"].(string), spec.Flags["record"].(bool))
		case "survey":
			layouts, err = app.AnalyzeSurvey(ctx, spec.DSN, spec.Flags["minrows"].(int64), spec.Flags["record"].(bool))
		case "configured":
			layouts, err = app.AnalyzeConfigured(ctx, spec.DSN, cnf, spec.Flags["record"].(bool))
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "analyze: could not get layout")
		}

		err = renderLayouts(os.Stdout, layouts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "analyze: could not render laout")
		}
	default:
		err = fmt.Errorf("%s: not implemented yet", spec.Command)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "horus %s: %v\n", spec.Command, err)
		return 1
	}
	return 0
}

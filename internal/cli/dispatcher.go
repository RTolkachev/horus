// dispatcher.go: maps the command word (init, analyze, plan, run,
// onboard) to its handler. Owns the shared lifecycle around every
// command: parse flags, construct the app, map errors to exit codes
// (0 = clean, 1 = error, 2 = changes pending). Handlers stay thin — all
// real logic lives in the packages internal/app wires together.
//
// onboard is a pure generator: converting an unpartitioned table is a
// blocking rebuild, so Horus produces the complete ALTER (boundaries
// resolved, retention collapse applied) plus a size/duration warning and
// a gh-ost/pt-osc alternative — and executes nothing. The operator runs
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

	switch spec.Command {
	case "init":
		err = app.Init(ctx, spec.DSN)
	default:
		err = fmt.Errorf("%s: not implemented yet", spec.Command)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "horus %s: %v\n", spec.Command, err)
		return 1
	}
	return 0
}

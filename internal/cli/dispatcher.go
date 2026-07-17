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
	"flag"
	"fmt"
	"os"

	"github.com/RTolkachev/horus/internal/app"
)

const usage = `usage: horus <command> [flags]

commands:
  init      create horus's own schema and tables (needs CREATE privilege; run once)
  analyze   show partition state of configured tables (read-only)
  plan      show what horus would change, without changing it
  run       apply pending changes (the cron entry point)
  onboard   generate a partitioning script for an unpartitioned table

flags:
  --dsn     database DSN (default: $HORUS_DSN)
`

// Run dispatches args (without the program name) and returns the process
// exit code.
func Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 1
	}
	cmd, rest := args[0], args[1:]

	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	dsn := fs.String("dsn", os.Getenv("HORUS_DSN"), "database DSN")
	if err := fs.Parse(rest); err != nil {
		return 1
	}
	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "horus: no DSN: pass --dsn or set HORUS_DSN")
		return 1
	}

	var err error
	switch cmd {
	case "init":
		err = app.Init(ctx, *dsn)
		if err == nil {
			fmt.Println("horus meta schema ready (horus.journal, horus.stats)")
		}
	case "analyze", "plan", "run", "onboard":
		err = fmt.Errorf("%s: not implemented yet", cmd)
	default:
		fmt.Fprintf(os.Stderr, "horus: unknown command %q\n\n%s", cmd, usage)
		return 1
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "horus %s: %v\n", cmd, err)
		return 1
	}
	return 0
}

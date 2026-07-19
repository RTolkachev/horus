// Package cli owns the command-line surface: argument parsing, command
// dispatch, and rendering of results.
//
// parser.go: turns argv + env + config-file path into a validated RunSpec.
// The precedence chain (flags > env > config file > defaults) is resolved
// here and ONLY here — no package downstream may read os.Getenv or argv.
//
// Allowed imports: internal/config, internal/domain.
// Forbidden: internal/dbdriver (commands reach the DB only through internal/app).
package cli

import (
	"flag"
	"fmt"
	"io"
)

// RunSpec is the parser's output: every setting resolved and validated,
// so no package downstream reads argv or env.
type RunSpec struct {
	Command    string
	DSN        string
	ConfigPath string

	Record bool   // analyze: also persist the snapshot
	Table  string // onboard: target table
}

// command couples a summary line to its flag definitions — usage text
// is generated from this table, so the two cannot drift.
type command struct {
	summary string
	flags   func(fs *flag.FlagSet, s *RunSpec) // nil = common flags only
}

// order fixes usage listing; the map alone would print randomly.
var order = []string{"init", "analyze", "plan", "run", "onboard"}

var commands = map[string]command{
	"init": {summary: "create horus's own schema and tables (needs CREATE privilege; run once)"},
	"analyze": {
		summary: "show partition state of configured tables (read-only)",
		flags: func(fs *flag.FlagSet, s *RunSpec) {
			fs.BoolVar(&s.Record, "record", false, "persist the snapshot to horus.stats")
		},
	},
	"plan":    {summary: "show what horus would change, without changing it"},
	"run":     {summary: "apply pending changes (the cron entry point)"},
	"onboard": {summary: "generate a partitioning script for an unpartitioned table"},
}

// Parse turns argv + env into a validated RunSpec. stderr receives usage
// and flag errors (tests pass a bytes.Buffer). flag.ErrHelp comes back
// unwrapped so the dispatcher can exit 0 on -h.
func Parse(args []string, getenv func(string) string, stderr io.Writer) (RunSpec, error) {
	if len(args) == 0 {
		usage(stderr)
		return RunSpec{}, fmt.Errorf("no command")
	}
	cmd, rest := args[0], args[1:]
	c, ok := commands[cmd]
	if !ok {
		usage(stderr)
		return RunSpec{}, fmt.Errorf("unknown command %q", cmd)
	}

	spec := RunSpec{Command: cmd}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&spec.DSN, "dsn", getenv("HORUS_DSN"), "database DSN (mysql://user:pass@tcp(host:port)/db)")
	fs.StringVar(&spec.ConfigPath, "config", "horus.yaml", "config file path")
	if c.flags != nil {
		c.flags(fs, &spec)
	}
	if err := fs.Parse(rest); err != nil {
		return RunSpec{}, err // includes flag.ErrHelp
	}

	if spec.DSN == "" {
		return RunSpec{}, fmt.Errorf("no DSN: pass --dsn or set HORUS_DSN")
	}
	if cmd == "onboard" {
		if fs.NArg() != 1 {
			return RunSpec{}, fmt.Errorf("onboard needs exactly one table name (flags first: horus onboard --dsn … events)")
		}
		spec.Table = fs.Arg(0)
	} else if fs.NArg() > 0 {
		return RunSpec{}, fmt.Errorf("%s takes no arguments, got %q", cmd, fs.Args())
	}
	return spec, nil
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: horus <command> [flags]\n\ncommands:")
	for _, name := range order {
		fmt.Fprintf(w, "  %-8s %s\n", name, commands[name].summary)
	}
}

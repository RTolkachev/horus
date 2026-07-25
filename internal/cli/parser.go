// Package cli owns the command-line surface: argument parsing, command
// dispatch, and rendering of results.
//
// parser.go: turns argv + env into a validated RunSpec. The precedence
// chain (flags > env > defaults) is resolved here and ONLY here - no
// package downstream may read os.Getenv or argv. The parser never opens
// the config file; it only carries the path. The dispatcher loads config
// for the commands that need it.
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
	Command    string // the verb: init, analyze, plan, run, onboard
	DSN        string // mysql://… ; engine scheme routes in dbbuilder
	ConfigPath string // --config path; loaded only by commands that need it
	Mode       string
	Flags      map[string]any
}

// command couples a summary line to its flag definitions - usage text
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
			fs.Bool("record", false, "persist the snapshot to horus.stats")
			fs.String("table", "", "table name to analyze")
			fs.Int64("minrows", 0, "all tables with at least that many rows will be analyzed (approximate, catalog estimate)")
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

	// Harvest command-specific flags the user actually set into Flags -
	// an absent key means "flag not passed", so presence stays
	// distinguishable from a zero value. Common flags (dsn, config) live
	// in typed fields and are skipped. Every stdlib flag value implements
	// flag.Getter, so Get returns the parsed, typed value.
	spec.Flags = make(map[string]any)
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "dsn", "config":
			return
		}
		spec.Flags[f.Name] = f.Value.(flag.Getter).Get()
	})

	if spec.DSN == "" {
		return RunSpec{}, fmt.Errorf("no DSN: pass --dsn or set HORUS_DSN")
	}

	switch cmd {
	case "onboard":
		if fs.NArg() != 1 {
			return RunSpec{}, fmt.Errorf("onboard needs exactly one table name (flags first: horus onboard --dsn … events)")
		}
		spec.Flags["table"] = fs.Arg(0)
	case "analyze":
		if _, ok := spec.Flags["record"]; !ok {
			spec.Flags["record"] = false
		}
		_, hasTable := spec.Flags["table"]
		_, hasMin := spec.Flags["minrows"]
		_, hasCfg := spec.Flags["configured"]

		switch {
		case hasTable && !hasMin && !hasCfg:
			spec.Mode = "table"
		case hasMin && !hasTable && !hasCfg:
			spec.Mode = "survey"
		case hasCfg && !hasTable && !hasMin:
			spec.Mode = "configured"
		default:
			return RunSpec{}, fmt.Errorf("analyze needs exactly one of --table, --minrows, --configured")
		}
	default:
		return RunSpec{}, fmt.Errorf("Unknown command")
	}
	return spec, nil
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: horus <command> [flags]\n\ncommands:")
	for _, name := range order {
		fmt.Fprintf(w, "  %-8s %s\n", name, commands[name].summary)
	}
}

// Parser tests exercise the full precedence chain (flag > env >
// default) without touching the process: env is a closure over a test
// map, output goes to a bytes.Buffer. Nothing global is read or
// mutated, so every subtest could run in parallel.
package cli

import (
	"bytes"
	"errors"
	"flag"
	"strings"
	"testing"
)

// env builds a getenv func from a map — the test's entire environment
// is visible in the test case, and the real one is never consulted.
func env(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}

const dsn = "mysql://horus:horus@tcp(127.0.0.1:3307)/horus_test"

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		want    RunSpec // compared with == (RunSpec is all value fields)
		wantErr string  // substring; empty = expect success
	}{
		{
			name: "flag DSN wins over env",
			args: []string{"analyze", "--dsn", dsn},
			env:  map[string]string{"HORUS_DSN": "mysql://env-loses"},
			want: RunSpec{Command: "analyze", DSN: dsn, ConfigPath: "horus.yaml"},
		},
		{
			name: "env supplies DSN when flag absent",
			args: []string{"analyze"},
			env:  map[string]string{"HORUS_DSN": dsn},
			want: RunSpec{Command: "analyze", DSN: dsn, ConfigPath: "horus.yaml"},
		},
		{
			name:    "no DSN anywhere",
			args:    []string{"analyze"},
			wantErr: "no DSN",
		},
		{
			name: "config flag overrides default path",
			args: []string{"plan", "--dsn", dsn, "--config", "prod.yaml"},
			want: RunSpec{Command: "plan", DSN: dsn, ConfigPath: "prod.yaml"},
		},
		{
			name: "analyze accepts record",
			args: []string{"analyze", "--dsn", dsn, "--record"},
			want: RunSpec{Command: "analyze", DSN: dsn, ConfigPath: "horus.yaml", Record: true},
		},
		{
			// Per-command vocabulary: the flag exists on analyze only,
			// so the FlagSet itself rejects it elsewhere.
			name:    "record is not a plan flag",
			args:    []string{"plan", "--dsn", dsn, "--record"},
			wantErr: "not defined",
		},
		{
			name:    "no command",
			args:    []string{},
			wantErr: "no command",
		},
		{
			name:    "unknown command",
			args:    []string{"destroy", "--dsn", dsn},
			wantErr: `unknown command "destroy"`,
		},
		{
			name: "onboard takes the table as positional arg",
			args: []string{"onboard", "--dsn", dsn, "events"},
			want: RunSpec{Command: "onboard", DSN: dsn, ConfigPath: "horus.yaml", Table: "events"},
		},
		{
			name:    "onboard without a table",
			args:    []string{"onboard", "--dsn", dsn},
			wantErr: "one table name",
		},
		{
			name:    "onboard with two tables",
			args:    []string{"onboard", "--dsn", dsn, "events", "audit_log"},
			wantErr: "one table name",
		},
		{
			// stdlib flag stops at the first non-flag token, so a flag
			// after the positional would be silently swallowed into
			// fs.Args() — rejecting extras keeps that mistake loud.
			name:    "stray positional args rejected",
			args:    []string{"analyze", "--dsn", dsn, "events"},
			wantErr: "takes no arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			got, err := Parse(tt.args, env(tt.env), &stderr)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Parse(%q) = %+v, want error containing %q", tt.args, got, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Parse(%q) error = %q, want it to contain %q", tt.args, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.args, err)
			}
			if got != tt.want {
				t.Errorf("Parse(%q)\n got  %+v\n want %+v", tt.args, got, tt.want)
			}
		})
	}
}

// -h is the one parse "error" that is not a failure: flag prints the
// command's defaults and returns flag.ErrHelp, which the dispatcher
// maps to exit 0. The sentinel must survive Parse unwrapped or that
// mapping breaks.
func TestParseHelp(t *testing.T) {
	var stderr bytes.Buffer
	_, err := Parse([]string{"analyze", "-h"}, env(nil), &stderr)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("Parse(-h) error = %v, want flag.ErrHelp", err)
	}
	if !strings.Contains(stderr.String(), "record") {
		t.Errorf("help output does not mention the record flag:\n%s", stderr.String())
	}
}

// Bad input prints the command list; every command must appear, in the
// declared order — a new command that misses the order slice would
// vanish from usage without this.
func TestParseUsageListsAllCommands(t *testing.T) {
	var stderr bytes.Buffer
	_, _ = Parse(nil, env(nil), &stderr)
	out := stderr.String()

	pos := -1
	for _, name := range order {
		// Match the whole listing line ("\n  <name> ..."), not the bare
		// word — command names also occur inside summary prose ("run
		// once"), which a substring search would find first.
		i := strings.Index(out, "\n  "+name+" ")
		if i < 0 {
			t.Errorf("usage output missing command %q", name)
			continue
		}
		if i < pos {
			t.Errorf("usage lists %q out of declared order", name)
		}
		pos = i
	}
	if len(commands) != len(order) {
		t.Errorf("commands map has %d entries, order slice %d — they must cover the same set",
			len(commands), len(order))
	}
}

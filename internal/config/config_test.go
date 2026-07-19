// Test files live next to the code they test and end in _test.go — the
// go tool builds them only for `go test`, never into the binary.
//
// This file declares `package config` (same package = white-box: tests
// can call unexported functions too). The other option is
// `package config_test` (black-box: only the exported API, like a real
// caller). Both are idiomatic; white-box is the simpler starting point.
//
// Run with:
//
//	go test ./internal/config/          # this package
//	go test -v ./internal/config/       # -v shows each test name
//	go test -run TestLoad ./...         # only tests matching a name
//	make test                           # everything
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDatabase is a table-driven test — the standard Go idiom when
// several cases share one shape: define the cases as data, loop, run
// each as a named subtest. Adding a case is one struct literal, not a
// new test function. Note there's no YAML and no files here: Database
// is a pure lookup, so the fixture is a plain struct literal.
func TestDatabase(t *testing.T) {
	cfg := Config{DB: []database{{Name: "analytics"}, {Name: "billing"}}}

	tests := []struct {
		name    string // subtest name, shown by -v and in failures
		lookup  string
		wantErr bool
	}{
		{"first entry", "analytics", false},
		{"last entry", "billing", false},
		{"unknown database", "reporting", true},
		{"empty name", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cfg.Database(tt.lookup)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Database(%q) = %+v, want error", tt.lookup, got)
				}
				// The message must name the missing database — that's
				// what makes the error actionable in a cron log.
				if !strings.Contains(err.Error(), fmt.Sprintf("%q", tt.lookup)) {
					t.Errorf("error %q does not mention %q", err, tt.lookup)
				}
				return
			}
			if err != nil {
				t.Fatalf("Database(%q) unexpected error: %v", tt.lookup, err)
			}
			if got.Name != tt.lookup {
				t.Errorf("Database(%q).Name = %q, want %q", tt.lookup, got.Name, tt.lookup)
			}
		})
	}
}

// Every test is a function named TestXxx taking *testing.T. The go tool
// finds them by that signature — no registration anywhere.
func TestLoad(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "test_horus.yaml")
	yaml := []byte(`
defaults:
  granularity: month
  horizon: 3
  retention: 12m
  strategy:
    type: id
    column: id
    step: 1000000
database:
  - name: horus_test
    defaults:
      granularity: week
      strategy:
        type: id
        column: seq
        step: 500000
    table:
      - name: events
        horizon: 6
        strategy:
          type: id
          column: event_id
          step: 5000000
      - name: audit_log
  - name: billing
    table:
      - name: invoices
`)
	if err := os.WriteFile(path, yaml, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	// Top-level defaults: every field set in the file must arrive.
	if got.Defaults.Granularity == nil || *got.Defaults.Granularity != "month" {
		t.Errorf("Defaults.Granularity = %v, want month", deref(got.Defaults.Granularity))
	}
	if got.Defaults.Horizon == nil || *got.Defaults.Horizon != 3 {
		t.Errorf("Defaults.Horizon = %v, want 3", deref(got.Defaults.Horizon))
	}
	if got.Defaults.Retention == nil || *got.Defaults.Retention != "12m" {
		t.Errorf("Defaults.Retention = %v, want 12m", deref(got.Defaults.Retention))
	}
	// strategy holds only value fields (no pointers), so the whole block
	// compares with == against a struct literal — no field-by-field checks.
	if want := (strategy{Type: "id", Column: "id", Step: 1000000}); got.Defaults.Strategy != want {
		t.Errorf("Defaults.Strategy = %+v, want %+v", got.Defaults.Strategy, want)
	}

	// Two entries: the multi-database file is the core config design —
	// this is the assertion that pins it.
	if len(got.DB) != 2 {
		t.Fatalf("len(DB) = %d, want 2", len(got.DB))
	}
	db := got.DB[0]
	if db.Name != "horus_test" {
		t.Errorf("DB[0].Name = %q, want %q", db.Name, "horus_test")
	}
	if got.DB[1].Name != "billing" {
		t.Errorf("DB[1].Name = %q, want %q", got.DB[1].Name, "billing")
	}
	// Per-database defaults sit beside the global ones; nothing merges them (yet).
	if db.Defaults.Granularity == nil || *db.Defaults.Granularity != "week" {
		t.Errorf("Database[0].Defaults.Granularity = %v, want week", deref(db.Defaults.Granularity))
	}
	if want := (strategy{Type: "id", Column: "seq", Step: 500000}); db.Defaults.Strategy != want {
		t.Errorf("Database[0].Defaults.Strategy = %+v, want %+v", db.Defaults.Strategy, want)
	}

	if len(db.Table) != 2 {
		t.Fatalf("len(Database[0].Table) = %d, want 2", len(db.Table))
	}
	events := db.Table[0]
	if events.Name != "events" {
		t.Errorf("Table[0].Name = %q, want %q", events.Name, "events")
	}
	if events.Horizon == nil || *events.Horizon != 6 {
		t.Errorf("Table[0].Horizon = %v, want 6", deref(events.Horizon))
	}
	if want := (strategy{Type: "id", Column: "event_id", Step: 5000000}); events.Strategy != want {
		t.Errorf("Table[0].Strategy = %+v, want %+v", events.Strategy, want)
	}
	// Load does not merge defaults: a key absent in the file must stay
	// nil, or the merge step could never tell "unset" from "set". When
	// parse grows merging, these assertions are the ones that change.
	if events.Granularity != nil {
		t.Errorf("Table[0].Granularity = %v, want nil (unset)", deref(events.Granularity))
	}
	audit := db.Table[1]
	if audit.Granularity != nil || audit.Horizon != nil || audit.Retention != nil {
		t.Errorf("Table[1] = %v/%v/%v, want all nil (nothing set in file)",
			deref(audit.Granularity), deref(audit.Horizon), deref(audit.Retention))
	}
	// Strategy is a value, not a pointer: absent in YAML means the zero
	// struct — indistinguishable from an explicitly empty block.
	if audit.Strategy != (strategy{}) {
		t.Errorf("Table[1].Strategy = %+v, want zero value (nothing set in file)", audit.Strategy)
	}
}

// deref renders an optional field for failure messages: the value, or
// "<nil>" for unset — printing a *string directly would show an address.
func deref[T any](p *T) any {
	if p == nil {
		return "<nil>"
	}
	return *p
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil {
		t.Fatal("Load() = nil error, want one")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Load() error = %v, want fs.ErrNotExist", err)
	}
}

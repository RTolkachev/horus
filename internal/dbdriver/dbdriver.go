// Package dbdriver defines the engine-agnostic port to the database — the
// ONLY way any component touches the DB. Interfaces sit at the ACTION
// level, not the SQL level: a "split" is one REORGANIZE on MySQL but a
// multi-statement choreography on Postgres, and that difference must stay
// inside the engine package. The SQL text lives behind these interfaces;
// nothing outside internal/dbdriver/* ever sees a statement.
//
// internal/app constructs one Driver per target database and hands each
// component only the facet it is allowed to hold:
//
//	Inspector  inventory
//	Querier    boundary (MaxIDBefore), executor (RowsAbove guard)
//	Applier    executor
//	Locker     the run handler
//	Meta       journal, stats (horus's own tables only)
//
// Allowed imports: internal/domain.
package dbdriver

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/RTolkachev/horus/internal/domain"
)

// ErrLockHeld is returned by Locker.AcquireLock when another horus
// instance already holds the maintenance lock on this database. Callers
// treat it as a clean refusal ("another horus is running"), not a failure.
var ErrLockHeld = errors.New("dbdriver: maintenance lock held by another horus instance")

// Driver is the full engine port. Only internal/app holds one whole;
// every other component receives a single facet.
type Driver interface {
	Inspector
	Querier
	Applier
	Locker
	Meta

	// Close releases all connections, including the dedicated lock session.
	Close() error
}

// Inspector reads catalog state. Inventory is its only caller.
type Inspector interface {
	// Layout returns the observed partition state of table, normalized:
	// partitions in bound order, catch-all identification, approximate
	// rows and bytes, and the ID watermark (AUTO_INCREMENT high-water
	// mark), all captured in one pass. A configured-but-unpartitioned
	// table is a valid observation, not an error
	// (PartitionLayout.Partitioned == false).
	Layout(ctx context.Context, table string) (domain.PartitionLayout, error)
}

// Querier reads table data. Two callers, two methods: boundary resolution
// at plan time, staleness guard at execute time.
type Querier interface {
	// MaxIDBefore resolves a calendar point to an ID point:
	// SELECT MAX(id) WHERE created_at < before. found == false means the
	// table has no rows before that instant (young table); the planner
	// treats that as "boundary not resolvable yet", never as zero.
	MaxIDBefore(ctx context.Context, table string, before time.Time) (id int64, found bool, err error)

	// RowsAbove counts rows in partition with id > pk — the executor's
	// pre-DDL check that a plan-time boundary hasn't drifted past
	// tolerance between planning and execution.
	RowsAbove(ctx context.Context, table, partition string, pk int64) (count int64, err error)
}

// ExecResult is Applier.Apply's outcome. Both values are success; the
// distinction exists so the journal can record that a replay was a
// replay.
type ExecResult int

const (
	// Applied: the DDL ran and changed the table.
	Applied ExecResult = iota
	// AlreadyApplied: the action had already landed (partition exists
	// with exactly this bound) — a crash-resume replay, deliberately not
	// an error. This return is what makes resume safe without atomic
	// DDL+journal.
	AlreadyApplied
)

// Applier executes plan actions. The executor is its only caller.
type Applier interface {
	// Apply compiles action into this engine's DDL and runs it. One
	// action may be any number of statements — that is the engine's
	// business. Apply MUST be idempotent: replaying a landed action
	// returns AlreadyApplied, nil.
	Apply(ctx context.Context, action domain.Action) (ExecResult, error)

	// RenderSQL returns the exact statements Apply would run for action,
	// without executing anything. One source of truth for display
	// (plan --show-sql) and for onboarding script generation.
	RenderSQL(action domain.Action) ([]string, error)
}

// Locker is the single-instance guard. The run handler is its only
// caller; analyze/plan/onboard never lock.
type Locker interface {
	// AcquireLock takes the advisory maintenance lock (GET_LOCK /
	// pg_advisory_lock) on a dedicated session that is never returned to
	// the pool. Fail-fast: if the lock is held, it returns ErrLockHeld
	// immediately rather than waiting — an overrunning run makes the next
	// cron slot a clean no-op. release frees the lock and the session.
	AcquireLock(ctx context.Context) (release func(context.Context) error, err error)
}

// Meta is storage access for horus's OWN tables (horus.journal,
// horus.stats), used by the journal and stats packages only.
//
// This facet IS a generic statement runner — the one place that shape is
// correct: the SQL that flows through it is written by Horus, targets
// tables Horus owns, and is engine-portable, so there is no dialect to
// hide. EnsureMetaSchema is the exception (CREATE TABLE syntax varies)
// and is therefore typed. Never point Meta at a target table.
type Meta interface {
	// EnsureMetaSchema creates the horus schema and its tables if they do
	// not exist. Idempotent; called once at app construction.
	EnsureMetaSchema(ctx context.Context) error

	MetaExec(ctx context.Context, stmt string, args ...any) error
	MetaQuery(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

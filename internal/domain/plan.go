// plan.go: Plan (ordered list of actions — disposable intent, recomputed
// every cycle) and the typed Action variants. v1 ships SplitCatchAll and
// CreateFuture only; Archive and Drop arrive in v1.1, gated on verified
// archive manifests.
package domain

import "time"

// Action is one partition change the executor can apply via
// dbdriver.Applier. Variants are closed: the isAction marker keeps the
// set enumerable so engine drivers can exhaustively switch on them.
type Action interface {
	isAction()
	// ActionTable returns the table the action targets.
	ActionTable() string
}

// SplitCatchAll carves a new bounded partition out of the catch-all:
// p_future becomes (new bounded partition, new catch-all). Metadata-only
// while the catch-all holds no rows above UpperBound.
type SplitCatchAll struct {
	Table        string
	NewPartition string
	// UpperBound is the resolved PK boundary — baked in at plan time by
	// the boundary solver, never re-resolved at execution.
	UpperBound int64
	// ResolvedAt anchors the executor's staleness guard.
	ResolvedAt time.Time
}

func (SplitCatchAll) isAction()             {}
func (a SplitCatchAll) ActionTable() string { return a.Table }

// CreateFuture pre-provisions a bounded partition beyond the current
// edge, per the configured horizon. Same shape as SplitCatchAll; kept a
// distinct type because the planner reasons about them differently
// (overdue split vs. ahead-of-schedule provisioning) and reports should
// too.
type CreateFuture struct {
	Table        string
	NewPartition string
	UpperBound   int64
	ResolvedAt   time.Time
}

func (CreateFuture) isAction()             {}
func (a CreateFuture) ActionTable() string { return a.Table }

// Plan is the planner's output: ordered actions plus the warnings that
// explain what the planner refused or wants a human to see. A Plan is
// intent — if lost, replanning reproduces it.
type Plan struct {
	Table     string
	CreatedAt time.Time
	Actions   []Action
	Warnings  []string
}

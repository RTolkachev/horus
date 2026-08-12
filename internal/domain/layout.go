// Package domain is the shared vocabulary: PartitionLayout, Partition,
// BoundaryMap, Action, Plan, Outcome. It imports NOTHING (stdlib only) and
// is imported by everyone - that property is what makes import cycles
// impossible and the planner testable with plain structs.
//
// layout.go: the normalized snapshot of a table's observed partition
// state, engine-agnostic. Produced by inventory (via dbdriver.Inspector),
// consumed by the planner and stats. Nobody writes a Layout by hand - it
// is derived, disposable, and true only for the instant it was taken.
package domain

// PartitionLayout is one table's observed partitioning facet: the
// partitions themselves. It always travels inside a domain.Table (its
// Layout field, nil when unpartitioned) - identity, watermark and
// snapshot time live on the wrapper, exactly once.
type PartitionLayout struct {
	// Partitions in ascending bound order.
	Partitions []Partition
}

// Partition is one partition's observed state.
type Partition struct {
	Name string

	// UpperBound is the exclusive upper edge (MySQL: VALUES LESS THAN).
	// Lower edges are implicit here - "wherever the previous partition
	// stopped" - which is MySQL RANGE law; an engine with explicit,
	// possibly gapped lower edges (Postgres FROM … TO …) adds a
	// LowerBound field when its driver exists.
	UpperBound Bound

	// ApproxRows and Bytes come from catalog statistics - threshold
	// inputs, not exact counts.
	ApproxRows int64
	Bytes      int64
}

// IsCatchAll reports whether this is the catch-all partition - upper
// edge unbounded - whose size decides whether splits stay metadata-only.
// Derived from the bound; there is deliberately no flag to disagree
// with it.
func (p Partition) IsCatchAll() bool {
	return p.UpperBound.Unbounded()
}

// CatchAll returns the catch-all partition, if the layout has one.
func (l PartitionLayout) CatchAll() (Partition, bool) {
	for _, p := range l.Partitions {
		if p.IsCatchAll() {
			return p, true
		}
	}
	return Partition{}, false
}

// InitResult is app.Init's report: which meta tables it created and
// which already existed (derived from check-first catalog reads - DDL
// itself reports nothing useful).
type InitResult struct {
	Created []string
	Existed []string
}

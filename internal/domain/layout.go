// Package domain is the shared vocabulary: PartitionLayout, Partition,
// BoundaryMap, Action, Plan, Outcome. It imports NOTHING (stdlib only) and
// is imported by everyone — that property is what makes import cycles
// impossible and the planner testable with plain structs.
//
// layout.go: the normalized snapshot of a table's observed partition
// state, engine-agnostic. Produced by inventory (via dbdriver.Inspector),
// consumed by the planner and stats. Nobody writes a Layout by hand — it
// is derived, disposable, and true only for the instant it was taken.
package domain

import "time"

// PartitionLayout is one table's observed state.
type PartitionLayout struct {
	Table string

	// Partitioned is false for a configured table that has not been
	// onboarded yet — a valid observation the planner must never emit
	// actions for.
	Partitioned bool

	// Partitions in ascending bound order. Empty when !Partitioned.
	Partitions []Partition

	// Watermark is the ID high-water mark (AUTO_INCREMENT), captured in
	// the same pass as the catalog read. Stats derives growth from its
	// delta between snapshots.
	Watermark int64

	TakenAt time.Time
}

// Partition is one partition's observed state. Only the upper bound is
// stored — RANGE partitioning speaks in VALUES LESS THAN, so lower edges
// are implicit ("wherever the previous partition stopped") and overlap or
// gaps are unrepresentable.
type Partition struct {
	Name string

	// UpperBound is the VALUES LESS THAN value. Meaningless when
	// IsCatchAll (MAXVALUE).
	UpperBound int64

	// IsCatchAll marks the MAXVALUE partition whose size decides whether
	// splits stay metadata-only.
	IsCatchAll bool

	// ApproxRows and Bytes come from catalog statistics — threshold
	// inputs, not exact counts.
	ApproxRows int64
	Bytes      int64
}

// CatchAll returns the catch-all partition, if the layout has one.
func (l PartitionLayout) CatchAll() (Partition, bool) {
	for _, p := range l.Partitions {
		if p.IsCatchAll {
			return p, true
		}
	}
	return Partition{}, false
}

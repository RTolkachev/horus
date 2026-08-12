// table.go: the observed representation of one table - identity, size
// and growth signals, and (when present) the partition layout and the
// engine's candidacy verdict nested on top. Inspector.Layout (the
// cycle's cheap read) fills identity + watermark + partitions;
// Inspector.Profile (analyze/onboard's deep read) fills everything,
// verdict included.
//
// Domain records neutral, observable FACTS, plus the engine's verdict
// expressed in neutral vocabulary (Finding). The raw candidacy inputs -
// keys, foreign keys, storage traits - deliberately do NOT cross the
// port: the driver reads them from its own catalogs, judges them by its
// own engine's rules, and reports only the verdict. Every field here
// must be fillable (or honestly zero) by any engine driver.
//
// Field discipline: every field has a nameable reader outside the
// drivers - render, step suggestion, the planner, onboard's gate.
// Facts the catalogs offer but nobody outside a driver reads stay out.
package domain

import "time"

// Table is one table's observed state. Commands populate only the
// facets they paid to fetch; zero values mean "not fetched", never an
// error.
type Table struct {
	Name string

	// ApproxRows, DataBytes and IndexBytes come from catalog statistics -
	// step-suggestion and rebuild-cost inputs, not exact counts.
	ApproxRows int64
	DataBytes  int64
	IndexBytes int64

	// Watermark is (approximately) the highest id the table has handed
	// out - the frontier of the data (MySQL: the AUTO_INCREMENT counter;
	// Postgres: the backing sequence). It only ever rises, so the delta
	// between two snapshots is pure insert volume: the growth signal
	// behind step suggestions and "when does the frontier outrun the
	// last partition bound".
	Watermark int64

	// CreatedAt is the table's creation time from the catalog. Age turns
	// a single watermark reading into a lifetime-average growth estimate -
	// the step suggestion's fallback until two recorded snapshots exist.
	CreatedAt time.Time

	// TakenAt is when this observation was captured.
	TakenAt time.Time

	// Layout is the partitioning facet; nil means not partitioned -
	// there is deliberately no separate bool to disagree with it.
	Layout *PartitionLayout

	// Validation is the engine's candidacy verdict, populated by
	// Inspector.Profile only. nil means "not assessed" - distinct from
	// an assessed, finding-free verdict - so the cycle's Layout-fetched
	// tables claim nothing.
	Validation *Validation
}

// Partitioned reports whether the table is partitioned.
func (t Table) Partitioned() bool {
	return t.Layout != nil
}

// Finding is one engine judgment about one rule, in words the operator
// reads.
type Finding struct {
	// Check is a stable slug naming the rule ("engine", "unique-keys",
	// "fk-inbound", …). Grouping and tests key on it; Detail is prose.
	Check string

	// Detail names the offender: "unique key uq_email does not contain
	// id", "fk orders_ibfk_1 → users(id)".
	Detail string
}

// Validation is the engine's partitioning-candidacy verdict for one
// table. The rules and the catalog facts they consume are engine-
// private; only this summary crosses the port.
type Validation struct {
	// Blockers: any entry means the table cannot be partitioned as-is.
	Blockers []Finding

	// Warnings: partitioning is possible, but the operator should know.
	Warnings []Finding

	// Candidate is the suggested partition column (the id-strategy
	// auto-assigned key); empty when blocked.
	Candidate string
}

// OK reports whether nothing blocks partitioning.
func (v Validation) OK() bool {
	return len(v.Blockers) == 0
}

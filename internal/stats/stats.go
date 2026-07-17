// Package stats is pure collection: it receives inventory's snapshot
// (taken_at, ID watermark, per-partition rows/bytes, catch-all fill),
// persists it as one row per table per cycle into horus.stats, and reads
// the series back to derive growth-per-day and days-until-threshold.
//
// Stats performs NO reads against the target server — inventory is the
// sole observer. The only tables stats touches are its own. There is no
// growth reporting unless a series has been collected (≥2 snapshots);
// Horus never synthesizes a trend from server counters.
//
// v1 scope: record, linear growth display in `analyze`, and GrowthReport
// as an advisory planner input (early threshold warnings, horizon
// assurance). Richer projections, skew detection, and alert hooks are
// v1.2.
//
// Growth is measured in ID space (watermark delta between snapshots):
// restart-immune, needs no instrumentation, and tracks boundary pressure
// — the thing partition planning actually consumes.
//
// Written by `run` (always) and `analyze --record` (opt-in, so plain
// analyze keeps its read-only promise). Read by `analyze`.
//
// Allowed imports: internal/dbdriver, internal/domain.
package stats

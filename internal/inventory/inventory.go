// Package inventory answers "what does this table look like right now?"
// — a per-cycle snapshot via dbdriver.Inspector, normalized to
// domain.PartitionLayout: partition bounds, rows, bytes, catch-all state,
// plus the ID watermark (AUTO_INCREMENT), captured in the same pass.
//
// Inventory is the ONLY component that observes the target server
// (information_schema today; sys/performance_schema if a cross-check is
// ever added). Everything downstream — planner, stats — works from its
// snapshot. Read-only, stateless, no history, no judgments: it reports
// facts; the planner raises the alarms.
//
// Allowed imports: internal/dbdriver, internal/domain.
package inventory

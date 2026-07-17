// dispatcher.go: registry mapping RunSpec.Command -> handler (analyze, plan,
// run, onboard). Owns the shared lifecycle around every command: construct the app,
// wire signal handling / context cancellation, map errors to exit codes
// (0 = clean, 1 = error, 2 = changes pending). Handlers stay thin — all
// real logic lives in the packages internal/app wires together.
//
// onboard is a pure generator: converting an unpartitioned table is a
// blocking rebuild, so Horus produces the complete ALTER (boundaries
// resolved, retention collapse applied) plus a size/duration warning and
// a gh-ost/pt-osc alternative — and executes nothing. The operator runs
// it on their own terms; the next cycle detects the now-partitioned table
// and adopts it (journaling an adoption baseline for drift detection).
// analyze/plan/run emit nothing for unpartitioned tables.
package cli

// Package executor walks a Plan in order, stopping on first failure. Per
// action: journal "attempting" -> staleness guard (via dbdriver.Query, for
// splits whose boundary may have drifted since plan time; abort and let
// the next cycle replan rather than silently re-resolving) ->
// dbdriver.Execute -> journal outcome. Treats AlreadyApplied as success.
//
// Sole writer of the journal — one component owns the audit trail, so
// crash-resume has a single source of truth. Checks ctx between actions
// for graceful shutdown.
//
// Allowed imports: internal/dbdriver, internal/journal, internal/domain.
package executor

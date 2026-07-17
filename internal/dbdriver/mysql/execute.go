// execute.go: the Applier facet.
//
// Apply compiles a domain.Action into DDL — SplitCatchAll and
// CreateFuture both render as a single ALTER TABLE ... REORGANIZE
// PARTITION of the catch-all (metadata-only while the catch-all holds no
// rows above the bound). Idempotency: if the partition already exists
// with exactly the action's bound, return dbdriver.AlreadyApplied so
// crash-resume replays are safe.
//
// RenderSQL returns the exact statements Apply would run, unexecuted —
// shared by Apply itself, plan --show-sql, and onboarding script
// generation, so there is one source of truth for the DDL Horus emits.
package mysql

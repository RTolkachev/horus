// cycle.go: composes one maintenance cycle, shared by `horus plan`
// (dry-run) and `horus run` (applies):
//
//	1. inventory.Layout            catalog snapshot
//	2. stats.Record + Series       persist snapshot row (run always;
//	                               analyze only with --record; plan never)
//	                               and read GrowthReport back for planning
//	3. planner.BoundaryNeeds       pure: which dates need boundaries
//	4. boundary.Resolve            date -> pk, skipped when needs are empty
//	5. journal.Load                manifests + any half-finished plan (resume)
//	6. planner.Plan                pure: layout+bounds+history+growth -> actions
//	7. executor.Apply              run mode only
//
// A future daemon/coordinator would loop over this same function; one-shot
// CLI commands call it directly.
package app

// Package planner is the logic core: given inventory snapshot, resolved
// boundaries, journal history, and config, decide what should change.
// Two-pass, both passes pure:
//
//	BoundaryNeeds(layout, cfg)                       -> dates needing resolution
//	Plan(layout, boundaries, history, growth, cfg)   -> ordered domain.Plan
//
// growth (stats.GrowthReport) is ADVISORY: it may move warnings earlier
// (catch-all will hit threshold before the next boundary) or pull a
// CreateFuture forward a cycle — it never decides where a bound goes.
// Every bound in every action traces to layout + config + resolved
// boundary. An empty series (first runs) degrades to pure calendar
// logic.
//
// Safety invariants live here: never emit Drop without a verified archive
// manifest (moot in v1 — Drop isn't emitted at all); refuse to split when
// the catch-all is past the metadata-only threshold (alert instead);
// surface layout drift rather than "fixing" it silently.
//
// Allowed imports: internal/domain ONLY. No dbdriver, no I/O, no clock —
// every decision must be testable with plain structs. If this package
// grows any other import, the design has failed.
package planner

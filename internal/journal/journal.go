// Package journal is the append-only history of what actually happened —
// the receipt, not the shopping list. Lives in a `horus` schema inside the
// target database (survives host loss; visible to any future second
// instance). Write side is used only by the executor; read side feeds the
// planner (history) and the run command (pending half-finished plan to
// resume before replanning).
//
// v1.1 adds archive manifests here as a queryable INDEX — manifest truth
// will live in the archive store next to the data it describes.
//
// Allowed imports: internal/dbdriver, internal/domain.
package journal

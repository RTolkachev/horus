// Package boundary maps wall-clock boundaries to PK values at PLAN time:
// SELECT MAX(id) FROM t WHERE created_at < X, via dbdriver.Query. The
// timestamp column is consulted exactly once, here — resolved values are
// baked into the plan so `horus plan` shows concrete numbers.
//
// Exactly one caller: the cycle, on the planner's behalf (between
// BoundaryNeeds and Plan). If the executor ever wants this package,
// boundaries are being decided at execution time — the thing this design
// exists to avoid.
//
// Allowed imports: internal/dbdriver, internal/domain.
package boundary

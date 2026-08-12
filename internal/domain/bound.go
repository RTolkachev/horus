// bound.go: partition bound values, generalized. A bound is a tuple of
// typed values because engines allow multi-column range partitioning
// (MySQL RANGE COLUMNS, Postgres multi-column ranges) over integers,
// strings (ULID-style keys) and times - the id strategy's single
// integer is just the one-value case. Drivers normalize catalog text
// into these; rendering a bound back into DDL, with each engine's
// quoting and keywords, is the engine's business.
package domain

import (
	"strconv"
	"strings"
	"time"
)

// BoundKind discriminates which BoundValue field is meaningful.
type BoundKind int

const (
	BoundInt BoundKind = iota
	BoundString
	BoundTime

	// BoundMax is the engine's unbounded marker (MAXVALUE on both MySQL
	// and Postgres); no value field applies.
	BoundMax
)

// BoundValue is one typed value in a bound tuple. Exactly the field
// matching Kind is meaningful; the others stay zero.
type BoundValue struct {
	Kind BoundKind
	Int  int64
	Str  string
	Time time.Time
}

// String renders the value for display, not for DDL.
func (v BoundValue) String() string {
	switch v.Kind {
	case BoundInt:
		return strconv.FormatInt(v.Int, 10)
	case BoundString:
		return v.Str
	case BoundTime:
		return v.Time.Format("2006-01-02 15:04:05")
	case BoundMax:
		return "MAXVALUE"
	}
	return "?"
}

// Bound is one partition edge: an ordered tuple, one value per
// partitioning column.
type Bound struct {
	Values []BoundValue
}

// IntBound is the id-strategy constructor: a single integer edge.
func IntBound(v int64) Bound {
	return Bound{Values: []BoundValue{{Kind: BoundInt, Int: v}}}
}

// MaxBound is the fully unbounded edge - the catch-all's upper bound.
func MaxBound() Bound {
	return Bound{Values: []BoundValue{{Kind: BoundMax}}}
}

// Unbounded reports whether every value is the unbounded marker.
// Partial tuples like (10, MAXVALUE) are bounded in their leading
// columns and deliberately NOT unbounded.
func (b Bound) Unbounded() bool {
	if len(b.Values) == 0 {
		return false
	}
	for _, v := range b.Values {
		if v.Kind != BoundMax {
			return false
		}
	}
	return true
}

// Int returns the single integer value of an id-strategy bound;
// ok == false for anything else (tuples, strings, times, MAXVALUE).
func (b Bound) Int() (int64, bool) {
	if len(b.Values) == 1 && b.Values[0].Kind == BoundInt {
		return b.Values[0].Int, true
	}
	return 0, false
}

// String renders the bound for display: a single value plain, tuples
// parenthesized.
func (b Bound) String() string {
	if len(b.Values) == 1 {
		return b.Values[0].String()
	}
	parts := make([]string, len(b.Values))
	for i, v := range b.Values {
		parts[i] = v.String()
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

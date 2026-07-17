// Package mysql implements dbdriver.Driver for MySQL. This package (and
// future sibling engine packages) is the only place SQL text may appear.
// Nothing outside internal/app may import it.
//
// mysql.go: Driver construction — DSN parsing, the work pool, the
// dedicated lock session handed to lock.go — and the private exec helper
// every statement in this package funnels through (logging, timeouts,
// error wrapping). Facets live in their own files: inspector.go,
// execute.go, lock.go, meta.go.
package mysql

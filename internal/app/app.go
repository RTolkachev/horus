// Package app is the DI container — the ONLY package that knows concrete
// implementations. It picks the db driver from the DSN (mysql for v1), wires
// inventory, stats, boundary, planner, executor, and journal, and hands
// composed entry points to the cli dispatcher.
//
// Swapping in the future postgres db driver must touch exactly this package
// and nothing else.
package app

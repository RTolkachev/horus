// Package cli owns the command-line surface: argument parsing, command
// dispatch, and rendering of results.
//
// parser.go: turns argv + env + config-file path into a validated RunSpec.
// The precedence chain (flags > env > config file > defaults) is resolved
// here and ONLY here — no package downstream may read os.Getenv or argv.
//
// Allowed imports: internal/config, internal/domain.
// Forbidden: internal/dbdriver (commands reach the DB only through internal/app).
package cli

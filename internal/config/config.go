// Package config loads and validates the operator's declared intent from
// YAML: per-table granularity (month/week), pre-provision horizon,
// catch-all size threshold, and retention window. Config lives in files
// under version control, never in the database.
//
// Allowed imports: internal/domain.
package config

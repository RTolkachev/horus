// stats.go: StatsSnapshot — one observation per table per cycle, built
// entirely from inventory's snapshot (taken_at, ID watermark,
// per-partition rows and bytes, catch-all fill) — and GrowthReport,
// derived from any two snapshots in the collected series (ID-space delta
// per day, days until catch-all threshold).
package domain

// meta.go: the Meta facet — storage for horus's own tables.
//
// EnsureMetaSchema creates the horus schema, horus.journal, and
// horus.stats if missing (idempotent; the init command is its only
// caller, so elevated CREATE privileges are needed once, not on every
// run); its CREATE TABLE text is the engine-specific part.
// MetaExec/MetaQuery pass portable, Horus-authored SQL through the shared
// exec helper — they must never be pointed at target tables.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
)

// metaSchema is the v1 shape of horus's own storage.
//
// journal: one row per attempted action — the executor writes a row with
// status 'attempting' BEFORE running DDL and finishes it afterwards, so
// an unfinished row is exactly "crashed mid-action". Recorded bounds are
// what drift detection compares against.
//
// stats: one row per table per observation; growth is derived from
// watermark deltas between rows, never stored.
var metaSchema = []string{
	`CREATE DATABASE IF NOT EXISTS horus`,

	`CREATE TABLE IF NOT EXISTS horus.journal (
		id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		table_name     VARCHAR(255)    NOT NULL,
		action         VARCHAR(32)     NOT NULL,
		partition_name VARCHAR(64)     NOT NULL,
		upper_bound    BIGINT          NOT NULL,
		status         ENUM('attempting','applied','already_applied','failed') NOT NULL,
		error          TEXT            NULL,
		started_at     DATETIME(6)     NOT NULL,
		finished_at    DATETIME(6)     NULL,
		PRIMARY KEY (id),
		KEY by_table (table_name, id)
	) ENGINE = InnoDB`,

	`CREATE TABLE IF NOT EXISTS horus.stats (
		id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		table_name  VARCHAR(255)    NOT NULL,
		taken_at    DATETIME(6)     NOT NULL,
		watermark   BIGINT          NOT NULL,
		approx_rows BIGINT          NOT NULL,
		bytes       BIGINT          NOT NULL,
		PRIMARY KEY (id),
		KEY by_table_time (table_name, taken_at)
	) ENGINE = InnoDB`,
}

func (d *Driver) EnsureMetaSchema(ctx context.Context) error {
	for _, stmt := range metaSchema {
		if err := d.exec(ctx, stmt); err != nil {
			return fmt.Errorf("ensure meta schema: %w", err)
		}
	}
	return nil
}

func (d *Driver) MetaExec(ctx context.Context, stmt string, args ...any) error {
	return d.exec(ctx, stmt, args...)
}

func (d *Driver) MetaQuery(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("mysql: meta query: %w", err)
	}
	return rows, nil
}

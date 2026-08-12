// meta.go: the Meta facet - storage for horus's own tables.
//
// EnsureMetaSchema creates horus.journal and horus.stats if missing
// (idempotent; the init command is its only caller). The horus database
// itself is assumed to exist - provisioned out-of-band by a DBA - so the
// app user needs only CREATE TABLE on that schema, never CREATE DATABASE;
// its CREATE TABLE text is the engine-specific part.
// MetaExec/MetaQuery pass portable, Horus-authored SQL through the shared
// exec helper - they must never be pointed at target tables.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
)

// metaSchema is the v1 shape of horus's own storage.
//
// journal: one row per attempted action - the executor writes a row with
// status 'attempting' BEFORE running DDL and finishes it afterwards, so
// an unfinished row is exactly "crashed mid-action". Recorded bounds are
// what drift detection compares against.
//
// stats: one row per table per observation; growth is derived from
// watermark deltas between rows, never stored.
var metaSchema = []string{
	`create table if not exists horus.journal (
		id             bigint unsigned not null auto_increment,
		table_name     varchar(255)    not null,
		action         varchar(32)     not null,
		partition_name varchar(64)     not null,
		upper_bound    bigint          not null,
		status         enum('attempting','applied','already_applied','failed') not null,
		error          text            null,
		started_at     datetime(6)     not null,
		finished_at    datetime(6)     null,
		primary key (id),
		key by_table (table_name, id)
	) engine = innodb`,

	`create table if not exists horus.stats (
		id          bigint unsigned not null auto_increment,
		table_name  varchar(255)    not null,
		taken_at    datetime(6)     not null,
		watermark   bigint          not null,
		approx_rows bigint          not null,
		bytes       bigint          not null,
		primary key (id),
		key by_table_time (table_name, taken_at)
	) engine = innodb`,
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

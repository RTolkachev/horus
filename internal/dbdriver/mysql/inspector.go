// inspector.go: the Inspector and Querier facets.
//
// Inspector.Layout reads information_schema.PARTITIONS (names,
// PARTITION_DESCRIPTION bounds, TABLE_ROWS, sizes) plus the
// AUTO_INCREMENT watermark from information_schema.TABLES in the same
// pass, and normalizes into domain.Table. An unpartitioned table
// returns Layout == nil, not an error.
//
// Querier.MaxIDBefore and Querier.RowsAbove are the two data reads:
// boundary resolution at plan time, staleness guard at execute time.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/RTolkachev/horus/internal/dbdriver"
	"github.com/RTolkachev/horus/internal/domain"
)

var _ dbdriver.Inspector = (*Driver)(nil)

// quoteIdent quotes an identifier for splicing into SQL text - the one
// escape hatch from ? placeholders, which cannot carry identifiers.
// Identifiers only; values keep going through ?.
func quoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// One row per partition; for an unpartitioned table the catalog still
// returns exactly one row with PARTITION_NAME NULL - that is the
// "exists but unpartitioned" signal, distinct from zero rows (no such
// table). DATE(...) bounds and other non-integer descriptions are not
// expected in v1 (id strategy only) and fail the parse loudly.
const layoutQuery = `
	select partition_name, partition_description, table_rows,
	       coalesce(data_length, 0) + coalesce(index_length, 0)
	from information_schema.partitions
	where table_schema = database() and table_name = ?
	order by partition_ordinal_position`

// auto_increment is null when the table has no auto-increment column -
// a real observation, so it becomes watermark 0, not an error.
const watermarkQuery = `
	select auto_increment
	from information_schema.tables
	where table_schema = database() and table_name = ?`

func (d *Driver) RunAnalyzeTable(ctx context.Context, table string) error {
	stmt := fmt.Sprintf("analyze table %s", quoteIdent(table))
	rows, err := d.db.QueryContext(ctx, stmt)
	if err != nil {
		return fmt.Errorf("mysql: analyze table %s: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var tbl, op, msgType, msgText string
		if err := rows.Scan(&tbl, &op, &msgType, &msgText); err != nil {
			return fmt.Errorf("mysql: analyze table %s: scan: %w", table, err)
		}
		if msgType == "Error" {
			return fmt.Errorf("mysql: analyze table %s: %s", table, msgText)
		}
	}
	return rows.Err()
}

// Profile is the deep observation behind analyze/onboard: Layout's
// facets plus sizes/age and the candidacy verdict. The rulebook and
// every raw catalog fact it consumes stay private to this package.
func (d *Driver) Profile(ctx context.Context, table string) (domain.Table, error) {
	// TODO(reggie): implement. Facts first, then the rulebook as pure
	// functions over the scanned rows (unit-test them there).
	//
	// facts (fill Table): information_schema.tables - engine, table_type,
	// table_rows, data_length, index_length, create_time, create_options;
	// plus Layout's partitions + watermark pass.
	//
	// blockers (any → Validation.Blockers, empty Candidate):
	//   - not a base table        tables.table_type != 'BASE TABLE'
	//   - engine not InnoDB       tables.engine
	//   - missing innodb_tables row  where name = concat(database(),'/',?)
	//     (sanity cross-check when engine says InnoDB and layout says
	//     unpartitioned - absence then means catalog confusion, fail loud)
	//   - general tablespace      innodb_tables.space_type = 'General'
	//     (name via innodb_tablespaces only for the message; requires
	//     PROCESS privilege - translate the access-denied error)
	//   - own foreign keys        key_column_usage where referenced_table_name is not null
	//   - inbound foreign keys    key_column_usage where referenced_table_name = ?
	//     (separate query - NOT derivable from this table's definition)
	//   - no candidate column     columns: no integer auto_increment
	//   - unique key w/o candidate  statistics: PK + every non_unique = 0
	//     key must contain the candidate column
	//   - fulltext/spatial index  statistics.index_type
	//
	// warnings (→ Validation.Warnings):
	//   - candidate column nullable   columns.is_nullable (NULLs sort
	//     into the lowest partition)
	//   - compressed row format       tables.row_format (INPLACE ops
	//     demote to table-copy; data_length understates rebuilt size)
	//   - encrypted                   create_options ENCRYPTION="Y"
	//     (transient disk doubling; shadow-table tools must recreate it)
	//   - has triggers                triggers on event_object_table = ?
	//     (pt-osc alternative unavailable; rows only visible with the
	//     TRIGGER privilege - absence of rows is not proof of absence)
	//   - stats stale or zero rows    tables.table_rows/update_time
	//     (size/step suggestions unreliable until analyze table runs)
	return domain.Table{}, fmt.Errorf("mysql: profile %s: not implemented yet", table)
}

func (d *Driver) Layout(ctx context.Context, table string) (domain.Table, error) {
	t := domain.Table{Name: table, TakenAt: time.Now()}

	rows, err := d.db.QueryContext(ctx, layoutQuery, table)
	if err != nil {
		return domain.Table{}, fmt.Errorf("mysql: layout %s: %w", table, err)
	}
	defer rows.Close()

	seen := 0
	for rows.Next() {
		seen++

		var (
			name  sql.NullString
			bound sql.NullString
			nrows sql.NullInt64
			bytes sql.NullInt64
		)
		if err := rows.Scan(&name, &bound, &nrows, &bytes); err != nil {
			return domain.Table{}, fmt.Errorf("mysql: layout %s: scan: %w", table, err)
		}

		if !name.Valid {
			continue
		}

		if t.Layout == nil {
			t.Layout = &domain.PartitionLayout{}
		}
		p := domain.Partition{
			Name:       name.String,
			ApproxRows: nrows.Int64,
			Bytes:      bytes.Int64,
		}
		if bound.String == "MAXVALUE" {
			p.UpperBound = domain.MaxBound()
		} else {
			v, err := strconv.ParseInt(bound.String, 10, 64)
			if err != nil {
				return domain.Table{}, fmt.Errorf(
					"mysql: layout %s: partition %s: unsupported bound %q (id strategy expects an integer): %w",
					table, p.Name, bound.String, err)
			}
			p.UpperBound = domain.IntBound(v)
		}
		t.Layout.Partitions = append(t.Layout.Partitions, p)
	}
	// rows.Err surfaces failures that happen DURING iteration (a
	// connection dying mid-cursor); Next just returns false on them.
	if err := rows.Err(); err != nil {
		return domain.Table{}, fmt.Errorf("mysql: layout %s: %w", table, err)
	}
	if seen == 0 {
		return domain.Table{}, fmt.Errorf("mysql: layout: table %q not found in current database", table)
	}

	var watermark sql.NullInt64
	if err := d.db.QueryRowContext(ctx, watermarkQuery, table).Scan(&watermark); err != nil {
		return domain.Table{}, fmt.Errorf("mysql: layout %s: watermark: %w", table, err)
	}
	t.Watermark = watermark.Int64

	return t, nil
}

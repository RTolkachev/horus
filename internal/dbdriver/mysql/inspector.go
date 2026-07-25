// inspector.go: the Inspector and Querier facets.
//
// Inspector.Layout reads information_schema.PARTITIONS (names,
// PARTITION_DESCRIPTION bounds, TABLE_ROWS, sizes) plus the
// AUTO_INCREMENT watermark from information_schema.TABLES in the same
// pass, and normalizes into domain.PartitionLayout. An unpartitioned
// table returns Partitioned == false, not an error.
//
// Querier.MaxIDBefore and Querier.RowsAbove are the two data reads:
// boundary resolution at plan time, staleness guard at execute time.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/RTolkachev/horus/internal/dbdriver"
	"github.com/RTolkachev/horus/internal/domain"
)

var _ dbdriver.Inspector = (*Driver)(nil)

// One row per partition; for an unpartitioned table the catalog still
// returns exactly one row with PARTITION_NAME NULL - that is the
// "exists but unpartitioned" signal, distinct from zero rows (no such
// table). DATE(...) bounds and other non-integer descriptions are not
// expected in v1 (id strategy only) and fail the parse loudly.
const layoutQuery = `
	SELECT PARTITION_NAME, PARTITION_DESCRIPTION, TABLE_ROWS,
	       COALESCE(DATA_LENGTH, 0) + COALESCE(INDEX_LENGTH, 0)
	FROM information_schema.PARTITIONS
	WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
	ORDER BY PARTITION_ORDINAL_POSITION`

// AUTO_INCREMENT is NULL when the table has no auto-increment column -
// a real observation (the table is not an id-strategy candidate), so it
// scans into a nullable and becomes watermark 0, not an error.
const watermarkQuery = `
	SELECT AUTO_INCREMENT
	FROM information_schema.TABLES
	WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`

func (d *Driver) Layout(ctx context.Context, table string) (domain.PartitionLayout, error) {
	layout := domain.PartitionLayout{Table: table, TakenAt: time.Now()}

	rows, err := d.db.QueryContext(ctx, layoutQuery, table)
	if err != nil {
		return domain.PartitionLayout{}, fmt.Errorf("mysql: layout %s: %w", table, err)
	}
	defer rows.Close()

	seen := 0
	for rows.Next() {
		seen++

		// Nullable catalog columns scan into sql.NullXxx - a plain
		// string/int64 target would error on the unpartitioned row.
		var (
			name  sql.NullString
			bound sql.NullString
			nrows sql.NullInt64
			bytes sql.NullInt64
		)
		if err := rows.Scan(&name, &bound, &nrows, &bytes); err != nil {
			return domain.PartitionLayout{}, fmt.Errorf("mysql: layout %s: scan: %w", table, err)
		}

		if !name.Valid {
			// The single NULL-name row: table exists, no partitioning.
			// Partitioned stays false, Partitions stays nil.
			continue
		}

		layout.Partitioned = true
		p := domain.Partition{
			Name:       name.String,
			ApproxRows: nrows.Int64,
			Bytes:      bytes.Int64,
		}
		if bound.String == "MAXVALUE" {
			p.IsCatchAll = true
		} else {
			p.UpperBound, err = strconv.ParseInt(bound.String, 10, 64)
			if err != nil {
				return domain.PartitionLayout{}, fmt.Errorf(
					"mysql: layout %s: partition %s: unsupported bound %q (id strategy expects an integer): %w",
					table, p.Name, bound.String, err)
			}
		}
		layout.Partitions = append(layout.Partitions, p)
	}
	// rows.Err surfaces failures that happen DURING iteration (a
	// connection dying mid-cursor); Next just returns false on them.
	if err := rows.Err(); err != nil {
		return domain.PartitionLayout{}, fmt.Errorf("mysql: layout %s: %w", table, err)
	}
	if seen == 0 {
		return domain.PartitionLayout{}, fmt.Errorf("mysql: layout: table %q not found in current database", table)
	}

	var watermark sql.NullInt64
	if err := d.db.QueryRowContext(ctx, watermarkQuery, table).Scan(&watermark); err != nil {
		return domain.PartitionLayout{}, fmt.Errorf("mysql: layout %s: watermark: %w", table, err)
	}
	layout.Watermark = watermark.Int64

	return layout, nil
}

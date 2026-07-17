// Package mysql implements dbdriver.Driver for MySQL. This package (and
// future sibling engine packages) is the only place SQL text may appear.
// Nothing outside internal/dbdriver/dbbuilder may import it.
//
// mysql.go: Driver construction — DSN parsing, the work pool — and the
// private exec helper every statement in this package funnels through.
// Facets live in their own files: inspector.go, execute.go, lock.go,
// meta.go.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/RTolkachev/horus/internal/dbdriver"
)

// Facet compliance is asserted per facet as each is implemented; the full
// dbdriver.Driver assertion lands with the last one.
var _ dbdriver.Meta = (*Driver)(nil)

type Driver struct {
	db *sql.DB
}

func New(dsn string) (*Driver, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql: invalid dsn: %w", err)
	}
	cfg.ParseTime = true

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("mysql: open: %w", err)
	}
	// A maintenance tool, not a service: a handful of connections is plenty.
	db.SetMaxOpenConns(4)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("mysql: connect %s/%s: %w", cfg.Addr, cfg.DBName, err)
	}
	return &Driver{db: db}, nil
}

func (d *Driver) Close() error {
	return d.db.Close()
}

// exec is the single funnel for every statement this package runs. Once
// run mode pins a locked session, this is where all work gets routed
// through it.
func (d *Driver) exec(ctx context.Context, stmt string, args ...any) error {
	if _, err := d.db.ExecContext(ctx, stmt, args...); err != nil {
		return fmt.Errorf("mysql: exec: %w", err)
	}
	return nil
}

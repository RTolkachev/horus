// Package dbbuilder is the composition root for db drivers: the ONLY
// package that names concrete engine packages. Everything else —
// including internal/app — sees dbdriver interfaces.
//
// Adding the future postgres engine means one new case in Build and
// nothing else outside internal/dbdriver/postgres.
package dbbuilder

import (
	"fmt"
	"strings"

	"github.com/RTolkachev/horus/internal/dbdriver"
	"github.com/RTolkachev/horus/internal/dbdriver/mysql"
)

// Driver is what Build returns today: the facets the mysql package
// implements so far. It widens to dbdriver.Driver once the remaining
// facets land, then disappears in favor of it.
type Driver interface {
	dbdriver.Meta
	Close() error
}

// DriverBuilder assembles a db driver from configuration.
type DriverBuilder struct {
	dsn    string
	engine string
}

func NewDriver() *DriverBuilder {
	return &DriverBuilder{}
}

// DSN sets the connection string. A scheme prefix ("mysql://") selects
// the engine unless Engine was called explicitly; a bare DSN defaults to
// mysql.
func (b *DriverBuilder) DSN(dsn string) *DriverBuilder {
	b.dsn = dsn
	return b
}

func (b *DriverBuilder) Build() (Driver, error) {
	engine, dsn, ok := strings.Cut(b.dsn, "://")
	if !ok {
		return nil, fmt.Errorf("dbbuilder: DSN must include an engine scheme, e.g. mysql://user:pass@tcp(host:port)/db")
	}
	switch engine {
	case "mysql":
		return mysql.New(dsn)
	default:
		return nil, fmt.Errorf("dbbuilder: unsupported engine %q (supported: mysql)", engine)
	}
}

// Package app wires the pieces into command entry points for the cli
// dispatcher: inventory, stats, boundary, planner, executor, and journal,
// each holding only the dbdriver facet it is allowed to see. Concrete
// engine construction is delegated to internal/dbdriver/dbbuilder - app
// imports interfaces only.
package app

import (
	"context"

	"github.com/RTolkachev/horus/internal/dbdriver/dbbuilder"
)

// Init creates horus's own tables (journal, stats) inside the horus
// database, which must already exist - a DBA provisions it out-of-band,
// so Init needs only CREATE TABLE on that schema. Idempotent; the one
// command that needs any CREATE privilege.
func Init(ctx context.Context, dsn string) error {
	drv, err := dbbuilder.NewDriver().DSN(dsn).Build()
	if err != nil {
		return err
	}
	defer drv.Close()
	return drv.EnsureMetaSchema(ctx)
}

// Package app wires the pieces into command entry points for the cli
// dispatcher: inventory, stats, boundary, planner, executor, and journal,
// each holding only the dbdriver facet it is allowed to see. Concrete
// engine construction is delegated to internal/dbdriver/dbbuilder — app
// imports interfaces only.
package app

import (
	"context"

	"github.com/RTolkachev/horus/internal/dbdriver/dbbuilder"
)

// Init provisions horus's own storage (schema + journal + stats tables).
// The one command that needs CREATE privileges; everything else runs with
// less.
func Init(ctx context.Context, dsn string) error {
	drv, err := dbbuilder.NewDriver().DSN(dsn).Build()
	if err != nil {
		return err
	}
	defer drv.Close()
	return drv.EnsureMetaSchema(ctx)
}

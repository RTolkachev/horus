// analyze.go: entry points for the analyze command - pre-config
// discovery. Point horus at a database BEFORE any config exists, learn
// each table's shape, and (eventually) get a strategy suggestion; the
// config file is written from what analyze reports, not the other way
// around. Only AnalyzeConfigured reads horus.yaml.
//
// Three entry points, one per selection mode (the cli dispatcher routes
// on RunSpec.Mode - mode dispatch is its job, not ours). record is a
// modifier, not a mode: it combines with every selection, so it stays a
// parameter. Everything after "which tables?" is the shared spine.
package app

import (
	"context"
	"fmt"

	"github.com/RTolkachev/horus/internal/config"
	"github.com/RTolkachev/horus/internal/dbdriver"
	"github.com/RTolkachev/horus/internal/dbdriver/dbbuilder"
	"github.com/RTolkachev/horus/internal/domain"
)

// selection answers "which tables?" for one analyze mode. It receives
// the Inspector facet because survey mode must consult the catalog;
// table mode ignores it.
type selection func(ctx context.Context, insp dbdriver.Inspector) ([]string, error)

// AnalyzeTable observes a single named table.
func AnalyzeTable(ctx context.Context, dsn, table string, record bool) ([]domain.Table, error) {
	return analyze(ctx, dsn, record, func(ctx context.Context, insp dbdriver.Inspector) ([]string, error) {
		return []string{table}, nil
	})
}

// AnalyzeSurvey observes every table in the connected database with at
// least minRows rows (approximate - catalog estimate). minRows 0 means
// no filter.
func AnalyzeSurvey(ctx context.Context, dsn string, minRows int64, record bool) ([]domain.Table, error) {
	return analyze(ctx, dsn, record, func(ctx context.Context, insp dbdriver.Inspector) ([]string, error) {
		// TODO(reggie): needs a catalog-listing method on the Inspector
		// facet (information_schema.TABLES, WHERE TABLE_ROWS >= minRows
		// pushed down into the query).
		return nil, fmt.Errorf("analyze survey: table listing not implemented yet")
	})
}

// AnalyzeConfigured observes the tables declared in the config entry
// matching the connected database - the inspection view for tables
// horus already manages.
func AnalyzeConfigured(ctx context.Context, dsn string, config config.Config, record bool) ([]domain.Table, error) {
	return analyze(ctx, dsn, record, func(ctx context.Context, insp dbdriver.Inspector) ([]string, error) {
		// TODO(reggie): config.Load(configPath), then select the entry by
		// the observed database (SELECT DATABASE() - needs a capability
		// on the driver), then return that entry's table names.
		return nil, fmt.Errorf("analyze configured: config selection not implemented yet")
	})
}

// analyze is the shared spine: build the driver, hand the Inspector
// facet to the selection and then to inventory per table, and persist
// the snapshots via stats when record is set.
func analyze(ctx context.Context, dsn string, record bool, sel selection) ([]domain.Table, error) {
	drv, err := dbbuilder.NewDriver().DSN(dsn).Build()
	if err != nil {
		return nil, err
	}
	defer drv.Close()
	list, err := sel(ctx, drv)
	if err != nil {
		return nil, fmt.Errorf("analyze: could not get table selection")
	}
	tables := make([]domain.Table, 0, len(list))
	for _, tbl := range list {
		table, err := drv.Layout(ctx, tbl)
		if err != nil {
			return nil, fmt.Errorf("analyze: could not get partition layout")
		}
		tables = append(tables, table)
	}
	// TODO(reggie): if record: stats.Record(ctx, drv, tables) (Meta
	// facet; translate missing horus.stats into "run horus init first").
	return tables, nil
}

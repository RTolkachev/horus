// meta.go: the Meta facet — storage for horus's own tables.
//
// EnsureMetaSchema creates the horus schema, horus.journal, and
// horus.stats if missing (idempotent, called once at app construction);
// its CREATE TABLE text is the engine-specific part. MetaExec/MetaQuery
// pass portable, Horus-authored SQL through the shared exec helper —
// they must never be pointed at target tables.
package mysql

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

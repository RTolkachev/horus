# Horus

Partition management for MySQL and PostgreSQL that doesn't leak into your
queries: partitions by **primary key**, derives time-aligned boundaries
automatically, and archives expired partitions to S3 — your schema keeps
`PRIMARY KEY (id)` and application code never learns the word "partition."

Based on the design described in
[Designing Partitioning You Don't Have to Babysit](https://explainanalyze.com/p/designing-partitioning-you-dont-have-to-babysit/).

## Status

Early development — not usable yet. Watch/star if the design interests you.

## License

Apache-2.0

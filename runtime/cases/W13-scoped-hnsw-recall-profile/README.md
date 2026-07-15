# W13 Scoped HNSW Recall Profile

W13 isolates the scoped ANN limitation preserved by W12. The W12 schedule made
550 vector requests against 100,000 current vectors. Every request returned the
correct current memory through Vermory's fallback contract, but only 138
remained effective vector results and 412 degraded to lexical under highly
selective tenant and single-continuity filters.

The case reuses the W12 current-authority shape and deterministic exact query
vectors. It qualifies PostgreSQL/pgvector query behavior only. It does not
measure embedding quality, memory formation, benchmark superiority, or a
semantic-default switch.

The target is exact and intentionally strict: all 550 requested vector queries
must return the expected current memory as effective vector results with zero
scope leakage and zero controlled fallback while projection state and the
embedder are healthy. Existing stale-projection, provider-outage, empty-vector,
authority, hash, deletion, and lexical fallback tests remain mandatory.

The implementation may use only query-local pgvector settings. Session-global
state must not leak through the pgx pool.

Frozen case SHA-256:
`d88057317e919941b98382fb9a473389d713224b263a4d8f8aee5a83afb093ba`.

# Scoped HNSW Recall Design

Date: 2026-07-15

Status: frozen for implementation

## Reality Finding

W12 built 100,000 current vectors and issued 550 requests in explicit vector
mode. The projection was current, the deterministic embedder succeeded, every
query had an exact matching vector, and all delivered memories were correct.
Only 138 requests remained effective vector results; 412 returned through the
audited lexical fallback.

This is not a fallback-safety failure. It is a scoped ANN effectiveness failure
that would make semantic mode unreliable under realistic tenant and continuity
filters.

## External Contract

The pgvector 0.8.0 documentation states that HNSW filtering is applied after
the approximate index scan. With default `hnsw.ef_search = 40`, a highly
selective filter may retain too few candidates. Version 0.8.0 introduced
iterative index scans, which continue scanning until enough filtered results
are found or the configured scan bound is reached.

The production query may enable:

```sql
SET LOCAL hnsw.iterative_scan = strict_order;
```

It must do so inside a transaction so the setting is query-local and cannot
leak through the pgx pool. Strict ordering preserves the existing distance and
authority ordering contract. W13 does not pre-authorize a larger
`hnsw.max_scan_tuples`, relaxed ordering, partial indexes, partitioning, or a
new external vector service.

## Implementation Boundary

The smallest acceptable implementation is:

1. begin a short read transaction for one vector search;
2. set `hnsw.iterative_scan` to `strict_order` with `SET LOCAL`;
3. execute the unchanged scoped, active-only, content-hash-guarded query;
4. close rows and commit before returning;
5. roll back on any setting, query, scan, or context error;
6. preserve coordinator fallback to lexical on any vector-query error.

The lexical path, authority model, vector schema, HNSW index, profile registry,
worker, and default retrieval mode remain unchanged.

## Verification

W13 requires three layers:

- a deterministic integration regression proving the query-local setting and
  pool cleanup;
- a medium real-pgvector reproduction that fails on the W12 implementation and
  passes with iterative scanning;
- the full 100,000-vector / 550-query frozen schedule with zero fallback,
  correct targets, zero leakage, and bounded latency.

Existing fallback tests remain required because iterative scanning must not
turn projection lag, provider failure, true empty projection, stale authority,
or hash mismatch into unsafe delivery.

## Non-Claims

- This is not a semantic-quality benchmark.
- This does not rank embedding models.
- This does not switch lexical from the product default.
- This does not qualify relaxed ordering or unbounded HNSW scans.
- This does not qualify one million vectors, HA, PITR, or cross-region use.

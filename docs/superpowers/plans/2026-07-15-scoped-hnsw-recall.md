# Scoped HNSW Recall Implementation Plan

**Goal:** Eliminate the W12 healthy-projection vector fallbacks under the frozen
100,000-vector tenant/continuity scope while preserving all authority, fallback,
isolation, latency, and deployment boundaries.

## Task 1: Freeze And Reproduce

- [x] Freeze the versioned W13 manifest, design, and checklist.
- [x] Preserve the final manifest SHA-256.
- [x] Add an opt-in real-pgvector reproduction using the W12 data shape.
- [ ] Record the failing effective-vector and fallback counts before production changes.

## Task 2: Query-Local Iterative Scan

- [ ] Add a failing integration regression for scoped vector effectiveness.
- [ ] Add a failing regression proving query-local HNSW state does not leak.
- [ ] Use a short transaction and `SET LOCAL hnsw.iterative_scan = strict_order`.
- [ ] Preserve exact ordering, authority/hash guards, and lexical fallback on errors.

## Task 3: Qualification

- [ ] Pass targeted store/coordinator tests and the medium profile.
- [ ] Pass the full 100,000-vector / 550-query profile with 550 effective vector results and zero fallback.
- [ ] Preserve zero leakage, correct targets, latency limits, database-size limit, and current projection state.
- [ ] Preserve stale projection, provider outage, empty projection, authority/hash mismatch, and deletion tests.

## Task 4: Evidence And Delivery

- [ ] Save normalized JSON and raw logs with hashes and zero credential matches.
- [ ] Update the evaluation matrix, hypothesis register, README, and W13 checklist.
- [ ] Run the full PostgreSQL, race, vet, module, OpenClaw, and release gates.
- [ ] Commit, push, update the Draft PR, and close protected CI/artifact verification.
- [ ] Keep the overall Vermory goal active after W13.

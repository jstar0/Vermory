# Vermory Hypothesis Register

Status: review candidate

Date: 2026-07-11

## 1. Purpose

This register contains implementation choices that appear reasonable but have not earned permanent design authority.

Each hypothesis must remain falsifiable. Passing unit tests proves that an implementation matches the hypothesis; it does not prove that the hypothesis matches reality.

Statuses are:

```text
proposed       reasoned candidate without sufficient real evidence
testing        currently exercised by a frozen experiment
supported      evidence supports continued use, but revision remains possible
accepted       frozen for a named release contract
rejected       evidence showed the hypothesis should not continue
```

## 2. Register

### H-001: Go modular monolith

- Status: `proposed`
- Candidate: one Go codebase with HTTP, MCP, worker, and CLI roles.
- Reason: preserves clear domain boundaries without early distributed-system cost.
- Evidence needed: the first three continuity paths can share domain services without client-specific semantics leaking into the core.
- Falsifier: one role requires materially different availability, security, scaling, or deployment semantics that cannot be isolated inside the modular monolith.
- Decision gate: after the first real-client batch and failure-recovery experiment.

### H-002: PostgreSQL-native operational stack

- Status: `supported`
- Candidate: PostgreSQL plus pgvector is sufficient for the native deployment; Redis, Neo4j, Qdrant, and Elasticsearch are not default dependencies.
- Existing evidence: backend lifecycle, B01-B10, 200-record, 1,000-record, deletion, ARM64, and AMD64 tests support the retrieval substrate.
- Existing operational evidence: the opt-in schema-15 scale profile seeded 10,000 governed memories, ran 8 concurrent readers with concurrent deletion, measured search p50/p95 at 17/210 ms on this machine, and recovered after terminating one PostgreSQL backend connection.
- Evidence needed: formation, sealed quality cases, long-running operation, and larger calibrated deployment profiles.
- Falsifier: a required constitutional behavior cannot be implemented reliably or within calibrated profiles without another default service.
- Decision gate: after the second evidence batch and first operational profile.

### H-003: Logical source-observation-memory separation

- Status: `testing`
- Candidate: preserve distinct logical concepts for source, observation, governed memory, history, projection, and delivery.
- Reason: prevents raw input, durable memory, vector state, and generated context from becoming one undifferentiated text pool.
- Experiment 0 signal: `W01-synapseloom-continuity` requires historical conversation observations to remain distinct from current repository authority; `S01-deletion-and-source-injection` requires an untrusted source instruction, governed scope, deleted target, and valid related guidance to remain behaviorally distinct.
- Evidence artifact: `artifacts/experiment-0/experiment-0-v1/report.json` and `report.md` under the same run directory.
- Evidence needed: real cases require independent provenance, formation, deletion, or delivery behavior for these concepts.
- Falsifier: two concepts remain observably identical across all first- and second-batch workflows and their separation creates only maintenance cost.
- Decision gate: after mapping both evidence batches to required operations.

### H-004: Candidate physical schema

- Status: `proposed`
- Candidate entities include tenants, continuities, bindings, sources, versions, observations, memories, revisions, evidence links, relations, events, projections, deliveries, jobs, and audits.
- Reason: this shape can express the current known lifecycle without making embeddings authoritative.
- Evidence needed: every entity must support at least one frozen query, invariant, state transition, or audit requirement.
- Falsifier: an entity has no independent behavior; required operations demand a missing boundary; or transaction consistency becomes unnecessarily complex.
- Decision gate: schema version 0.1 after discovery mapping; schema version 1 only after two materially different batches.

No table name or one-to-one mapping between logical and physical entities is accepted by this hypothesis.

### H-005: Revision-oriented updates

- Status: `testing`
- Candidate: ordinary updates append revisions and explicit change events; privacy deletion can erase protected content while retaining permitted content-free tombstones.
- Reason: supports current-state use, historical explanation, and deletion without silent overwrite.
- Experiment 0 signal: `W01-synapseloom-continuity` freezes a user correction followed by newer repository truth; `C01-device-maintenance-continuity` freezes a failed action followed by a verified correction; `S01-deletion-and-source-injection` freezes explicit deletion without requiring unrelated valid guidance to disappear.
- Evidence artifact: `artifacts/experiment-0/experiment-0-v1/report.json` and `report.md` under the same run directory.
- Evidence needed: progressive correction, conflicting sources, user reversal, archive/reactivation, and hard deletion cases.
- Falsifier: revision identity cannot express common real corrections without duplicating or fragmenting memory unnaturally.
- Decision gate: after update/conflict/deletion experiments.

### H-006: Memory-kind taxonomy

- Status: `proposed`
- Candidate seed kinds: fact, decision, constraint, preference, progress, next action, procedure, experience, risk, identity.
- Reason: these labels may help formation policy and context composition.
- Evidence needed: each retained kind changes a real policy, ranking, presentation, retention, or evaluation decision.
- Falsifier: labels are ambiguous, frequently multi-valued, require constant relabeling, or do not change behavior.
- Decision gate: after labeling the first two evidence batches independently and measuring disagreement.

The system may adopt fewer kinds, multi-label facets, structured attributes, or no fixed taxonomy.

### H-007: Retention classes

- Status: `testing`
- Candidate concepts: working, continuity-durable, and global-default retention.
- Reason: same-session usefulness, continuity reuse, and cross-context defaults have different promotion and expiry risks.
- Experiment 0 signal: `G01-language-default-local-override` requires a local English override to expire without rewriting a stable Chinese default; `S01-deletion-and-source-injection` rejects promotion from an untrusted source into Global Defaults.
- Evidence artifact: `artifacts/experiment-0/experiment-0-v1/report.json` and `report.md` under the same run directory.
- Evidence needed: positive and negative promotion, expiry, and deletion cases across all three continuity lines.
- Falsifier: retention is better expressed by policy, validity, and continuity without a separate class.
- Decision gate: after same-session and global-default experiments.

### H-008: Lifecycle state machine

- Status: `testing`
- Candidate meanings: pending, active, superseded, archived, rejected, and deleted, with conflict represented separately.
- Reason: separates proposal, current use, historical retention, rejection, and forgetting.
- Experiment 0 signal: `C01-device-maintenance-continuity` distinguishes failed, corrected, and verified action state; `S01-deletion-and-source-injection` requires a deleted target to remain unavailable while independent related guidance remains active.
- Evidence artifact: `artifacts/experiment-0/experiment-0-v1/report.json` and `report.md` under the same run directory.
- Evidence needed: every transition must correspond to a real user or source workflow; concurrent transitions must be deterministic.
- Falsifier: common correction, merge, split, temporary validity, contested state, or deletion behavior cannot be represented without exceptions.
- Decision gate: after lifecycle mutation tests and two real evidence batches.

Exact state names and transition edges are not frozen.

### H-009: Hybrid native retrieval

- Status: `testing` (`measured` on W08; `production_path_integrated` on W09)
- Candidate: continuity and lifecycle filtering followed by lexical, exact structured, trigram, and pgvector candidate generation with versioned fusion and optional reranking.
- Reason: pure vector Top-K is weak for technical identifiers and cannot itself encode source authority or lifecycle.
- Existing evidence: W08 ran 24 frozen mixed-language and technical queries over 48 active memories plus proposed, superseded, deleted, cross-continuity, and cross-tenant controls using direct SiliconFlow `BAAI/bge-m3`. Active-only pgvector and exact-guarded RRF both reached Recall@K `1.0000` and MRR `0.9792`, compared with lexical Recall@K `0.6875` and MRR `0.6806`; exact identifiers remained `1.0000`. W09 then connected the active-only vector path to real MCP and Web Chat runtimes with a durable event worker, restricted-role RLS, exact lexical degradation for cursor lag and provider outage, vector reset/rebuild, native dump/restore, and real Grok consumption/writeback. All W09 scope, lifecycle, recovery, and credential hard gates passed.
- Current interpretation: the pgvector candidate path is production-path integrated but remains opt-in. W08 and the independent W10 batch both show a large semantic-retrieval improvement over lexical on their frozen corpora, while the current RRF formula matches vector quality and adds latency rather than demonstrating an independent gain. Lexical remains the default.
- Evidence artifact: `docs/evidence/2026-07-15-independent-retrieval-batch.md` and its report snapshot record a fresh PostgreSQL 18 run over 39 governed records, 18 queries, 102 direct SiliconFlow `BAAI/bge-m3` requests, zero forbidden/ineligible results, and rebuild equivalence.
- Existing source-authority evidence: lexical workspace/conversation and vector retrieval now apply the same explicit origin tie-break; PostgreSQL tests prove an explicit user correction wins an equal-relevance source update without bypassing lifecycle or scope controls.
- Evidence needed: authority behavior on a broader conflict corpus, scale/backlog qualification, and optional rerank comparison on a sealed or externally held corpus. Calibrated profile thresholds, migration rollback, and the first explicit candidate decision are now recorded under H-011.
- Falsifier: a simpler measured strategy matches quality, task success, cost, and failure behavior; or the candidate strategy cannot meet calibrated latency.
- Decision gate: after a second independent retrieval batch, calibrated latency/quality thresholds, and a production outage/fallback slice.

No ranking algorithm or weight is accepted before ablation.

### H-010: Language-aware application analyzer

- Status: `proposed`
- Candidate: Go-side normalization and token extraction preserve Chinese terms and exact technical identifiers before PostgreSQL lexical indexing.
- Reason: default PostgreSQL tokenization alone may not serve mixed Chinese technical text reliably.
- Evidence needed: Chinese and mixed-language retrieval ablation with paths, flags, errors, model names, and code symbols.
- Falsifier: database-native or another established analyzer provides better portable quality with lower maintenance.
- Decision gate: after mixed-language retrieval experiments on ARM64 and AMD64.

### H-011: Versioned semantic projection generations

- Status: `supported` (generation mechanism); v2 remains `candidate`
- Candidate: embeddings are stored by model and projection generation so old and candidate models can coexist during migration.
- Reason: avoids coupling authoritative memory to one embedding model and supports measured cutover.
- Existing evidence: migration 15 registers active v1 and candidate v2 profiles with independent cursors and vector rows. The first rehearsal rebuilt `BAAI/bge-m3` and `BAAI/bge-large-zh-v1.5` side by side with 31 requests each, 30 rows each, zero cursor lag, unchanged v1 row count, and required-fact retrieval through both profiles. The W10 profile comparison then ran both registered profiles through the production worker, coordinator, audit, reset, and rebuild paths. Three corrected-corpus runs produced identical quality values and zero safety/lifecycle/degradation failures. v2 preserved Recall@K `1.0000` but regressed Hit@1 by `0.1111`, MRR by `0.0648`, and nDCG@K by `0.0503`, so the frozen promotion policy retained it as a candidate.
- Evidence artifact: `docs/evidence/2026-07-15-retrieval-profile-migration.md` and `docs/evidence/2026-07-15-retrieval-profile-promotion-decision.md`.
- Current decision: keep `siliconflow-bge-m3-1024-v1` active/default and `siliconflow-bge-large-zh-1024-v2` candidate. A future candidate requires a new corpus and recorded promotion decision.
- Evidence needed: a separate projection class for dimensionality changes and scale/backlog/restart qualification; these are not required to support the current same-dimension generation mechanism.
- Falsifier: a simpler rebuild-and-swap mechanism is operationally sufficient for calibrated deployment profiles.
- Decision gate: passed for the current 1024-dimensional profile class; reopen for a different dimensionality or storage class.

### H-012: PostgreSQL transactional outbox

- Status: `supported` for the current self-hosted profile
- Candidate: authoritative transactions enqueue projection and provider work through PostgreSQL, with idempotent workers and no default Redis dependency.
- Reason: aligns memory state and projection jobs without introducing a second required service.
- Existing evidence: W11 created 1,000 governed facts and projection events in a disposable PostgreSQL 18 cluster, processed a bounded 128-event batch, retained cursor position across provider failure, replayed the full event stream from cursor zero without duplicate vectors, stopped PostgreSQL with `immediate` while embedding was in flight, recovered through the same runtime pool, and proved concurrent deletion wins over late embedding. A separate tenant completed direct SiliconFlow projection and vector retrieval after restart with two real `BAAI/bge-m3` requests.
- Evidence artifact: `docs/evidence/2026-07-15-projection-outbox-fault-profile.md`.
- Current decision: PostgreSQL remains the default authority and transactional outbox; Redis is not a required deployment dependency for the measured developer-local and self-hosted profiles.
- Evidence needed: sustained server-scale backlog, multiple competing workers, retention pressure, and restart during a dimensionality migration.
- Falsifier: queue contention or operational requirements exceed calibrated profiles and an external queue produces a clearly safer design.
- Decision gate: passed for the current self-hosted profile; reopen for server-qualification or cross-region profiles.

### H-013: Row-level security defense in depth

- Status: `proposed`
- Candidate: application authorization, tenant-bearing foreign keys, and PostgreSQL row-level security jointly protect tenant boundaries.
- Reason: query filters alone are an insufficient final barrier for a multi-tenant memory platform.
- Evidence needed: integration tests under tenant-scoped database roles, migration tests, and deliberate filter-omission attacks.
- Falsifier: the initial supported deployment is explicitly single-tenant and RLS creates correctness or operations problems; the multi-tenant profile would still require a separate acceptance decision.
- Decision gate: before any multi-tenant release claim.

### H-014: Prepare and commit client operations

- Status: `proposed`
- Candidate: ordinary clients need one pre-task context operation and one post-task observation/write-back operation, with administrative APIs separate.
- Reason: keeps normal client integration low-friction while preserving governance.
- Evidence needed: one real coder, one Web Chat/API simulator, and one everyday-assistant path can integrate without custom memory semantics.
- Falsifier: streaming, tool-loop, long-running session, or client lifecycle requires a different interaction contract.
- Decision gate: after the first real-client experiment.

Names, payloads, streaming behavior, and transport are not frozen.

### H-015: Optional backend projection adapters

- Status: `supported`
- Candidate: mem0, MemOS, and Supermemory receive only eligible search projections and can be rebuilt from PostgreSQL.
- Existing evidence: all four tested backends implement the lifecycle adapter contract and pass the current scenario set.
- Evidence needed: formation-to-projection synchronization, deletion propagation, adapter outage, and rebuild under real trajectories.
- Falsifier: an adapter cannot preserve constitutional deletion or isolation even when treated as disposable; that adapter is removed rather than weakening the constitution.
- Decision gate: independently for each adapter before supported-release status.

## 3. Decision Records

When a hypothesis changes status, record:

```text
hypothesis id
old and new status
evidence batch and run ids
public and sealed result summary
baseline comparison
failure and limitation summary
decision and rationale
schema or API compatibility effect
rollback path
```

Accepted hypotheses remain revisable through a new versioned product or architecture decision. Existing evidence is never rewritten to match a later design.

## 4. Review Rule

Before implementation adds a new permanent entity, service, state, relation, provider dependency, or release metric, it must either:

- map to an existing hypothesis and its evidence gate; or
- enter this register as a new falsifiable hypothesis.

This rule prevents both speculative complexity and unrecorded simplification.

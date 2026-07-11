# Vermory Reality-First Memory Platform Design

Status: review candidate

Date: 2026-07-11

## 1. Purpose

This document is the index for the Vermory memory-platform design. It replaces the previous monolithic draft that mixed stable product boundaries with unverified schema, algorithm, benchmark, scale, and performance assumptions.

Vermory design is now maintained in four layers:

1. [Product Constitution](2026-07-11-vermory-product-constitution.md)
2. [Reality Program](2026-07-11-vermory-reality-program.md)
3. [Hypothesis Register](2026-07-11-vermory-hypothesis-register.md)
4. [Vertical Experiment Plan](2026-07-11-vermory-vertical-experiment-plan.md)

These documents have different authority. They must not be treated as four equivalent specifications.

## 2. Authority Order

### 2.1 Product Constitution

The constitution defines the product, user-visible behavior, continuity contracts, ownership boundaries, and zero-tolerance invariants. It is stable. Changing it requires an explicit product decision, not an implementation convenience.

### 2.2 Reality Program

The reality program defines how evidence is acquired, frozen, blinded, compared, and reported. It is stable as a method but may expand as better datasets, clients, and benchmarks become available.

### 2.3 Hypothesis Register

The register contains implementation candidates. Every item is provisional until its stated evidence gate is met. A hypothesis may be revised or rejected without changing the product constitution.

### 2.4 Vertical Experiment Plan

The experiment plan controls the next evidence-producing work. It deliberately builds complete paths through real clients instead of completing storage, retrieval, and API layers in isolation. It is expected to change when experiments expose incorrect assumptions.

## 3. Decision Rule

Vermory follows this order:

```text
real failure or real workflow
-> frozen expected and forbidden behavior
-> product invariant check
-> smallest complete design hypothesis
-> end-to-end implementation through a real client
-> external baseline comparison
-> public and sealed evaluation
-> keep, revise, or reject the hypothesis
```

The implementation does not receive authority merely because it is already written. A benchmark name does not receive authority merely because it is public. A schema field does not receive authority merely because it appears comprehensive.

## 4. What Is Fixed Now

The following decisions are fixed for the next implementation cycle:

- Vermory is an AI memory and context platform, not a memo CRUD service.
- It supports same-session and cross-session use.
- Workspace-backed continuity, conversation-backed continuity, and global defaults are the primary continuity contracts.
- PostgreSQL owns authoritative state in the native deployment.
- Search indexes and optional memory backends are disposable and rebuildable.
- mem0, MemOS, and Supermemory are optional adapters, not semantic authorities.
- LLMs may propose memory but do not own truth or lifecycle state.
- wrong binding, forbidden leakage, deleted-fact leakage, and stale-as-current behavior are hard failures.
- real downstream behavior and user correction burden matter more than internal record counts.
- official benchmark runs, translated proxies, and inspired cases are reported separately.

## 5. What Is Not Fixed Yet

The following remain hypotheses:

- final PostgreSQL table layout;
- final memory kinds and retention classes;
- final lifecycle state machine;
- whether observations require a dedicated physical table;
- relation types and whether all require persistence;
- hybrid retrieval fusion algorithm;
- exact context preparation and write-back API shape;
- use of a PostgreSQL outbox or another queue mechanism;
- quality thresholds before baseline calibration;
- reference scale and latency profiles;
- the final set of public benchmarks used for release qualification.

## 6. Supersession

This design package supersedes project-centric, competition-centric, and packet-centric portions of earlier designs when they conflict with the Product Constitution.

Existing code and evidence remain useful inputs:

- source and source-version storage;
- continuity and bridge primitives;
- provider, runner, artifact, and redaction foundations;
- PostgreSQL and pgvector integration;
- mem0, MemOS, and Supermemory adapters;
- backend lifecycle and B01-B10 evidence.

Those assets prove only the behavior they actually exercised. They do not freeze the future memory model and do not prove the full platform complete.

## 7. Implementation Gate

Implementation may begin after these four design layers are reviewed together. The first implementation work is the initial evidence batch and the first cross-line vertical experiment, not a final-schema migration or a package scaffold.

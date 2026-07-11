# Vermory Memory Backend Bake-Off

> Supporting substrate evidence. This bake-off validates rebuildable retrieval adapters; it does not prove the complete Vermory memory platform.

## Decision Scope

This bake-off selects the default memory engine for the practical open-source
Vermory platform. It does not select the system of record. Vermory owns
workspace and conversation identity, source binding, claim lifecycle, bridge
actions, authorization, and audit records in PostgreSQL regardless of the
memory engine selected.

Candidates in the first round:

- `native`: PostgreSQL full-text and vector retrieval without a memory framework
- `mem0`: self-hosted OSS server
- `memos`: self-hosted MemOS service
- `supermemory`: self-hosted local server

Graphiti remains a second-round temporal-graph candidate. It is not compared as
a complete memory service in the first round.

## Reproducible Runtime

All candidates must run on Linux containers or a documented Linux binary. The
reference development environment is an ARM64 Ubuntu VM, but every accepted
candidate must also publish or build for Linux AMD64.

The first quality run uses the same local models for every candidate:

- extraction LLM: `qwen3:4b`
- embedding model: `bge-m3`

The top two candidates are rerun with the same remote production-grade model.
Provider-specific managed extraction models are reported separately and cannot
decide the open-source default.

## Backend Contract

Every adapter must implement the following behavior:

```text
Health
Put
Search
Update
Delete
ResetScope
RebuildScope
Stats
```

Every stored item carries these Vermory-controlled fields:

```text
tenant_id
continuity_id
continuity_line
record_id
source_id
source_version
lifecycle_status
content
metadata
```

Search must always receive an explicit `tenant_id` and `continuity_id`. An
adapter that cannot enforce those filters is not eligible as the default.

## Hard Gates

A candidate is rejected from the default role if any hard gate fails:

1. A query leaks a record from another tenant or continuity.
2. A deleted record remains retrievable after the backend reports deletion.
3. A superseded record is returned as current without an explicit history query.
4. The index cannot be rebuilt from PostgreSQL-owned records.
5. The service has no supported Linux deployment path.
6. The self-hosted license prevents normal open-source distribution or use.
7. A normal API operation requires a vendor-hosted control plane.

## Scenario Set

### B01 Workspace Isolation

Load the two similar repositories from case `101-workspace-parallel-repos` into
separate continuity IDs. Query each repository with ambiguous component names.
No feature flag, owner, timeout, or retry policy may cross the boundary.

### B02 Cross-Tool Continuation

Load the multi-coder relay from case `105-workspace-multi-coder-relay`. A new
client identity must recover the current owner, frozen backend contract, target
file, and next step from the same workspace continuity.

### B03 Conversation Isolation

Load the housing search and job search into different conversation continuity
IDs. Queries must retain current preferences while excluding adjacent threads
and unrelated people.

### B04 Supersede and Historical Query

Store `MongoDB is the current database`, then supersede it with `PostgreSQL is
the current database`. Current-state search must return PostgreSQL and suppress
MongoDB. Historical explanation is assembled from PostgreSQL-owned claim
history; it is a platform-level test rather than a memory-engine requirement.

### B05 Delete and Scope Erasure

Delete one record, then erase an entire continuity. Neither operation may leave
retrievable content. The test includes exact, semantic, and paraphrased queries.

### B06 Source-First Conflict

Store `/v1/orders` as memory, then provide a newer repository source version
containing `/v2/orders`. The engine may return both as candidates, but the
The Vermory result must select `/v2/orders` and mark the older record stale.

### B07 Bridge Noise Filtering

Use case `301-bridge-chat-to-workspace`. Stable product decisions must survive
promotion into the workspace; mascot jokes, candidate names, and the abandoned
Slack bot must not become default project context.

### B08 Rebind and Rebuild

Move a workspace anchor while preserving its continuity ID. Destroy the memory
index, rebuild it from PostgreSQL-owned active records, and verify that recall
and isolation scores return to their pre-destruction values.

### B09 Noise Growth

Ingest temporary guesses, failed debugging paths, repeated facts, jokes, and
unconfirmed alternatives around valid durable facts. Measure retained record
count, duplicate rate, irrelevant retrieval rate, and token cost.

### B10 Chinese and Mixed Technical Text

Use Chinese prompts containing English paths, identifiers, feature flags, and
model names. This prevents an English-only embedding default from winning on a
dataset that does not resemble actual Vermory usage.

## Metrics

Quality metrics:

- current-fact recall
- forbidden cross-scope leakage
- stale-fact leakage
- deletion residue
- duplicate rate
- irrelevant retrieval rate
- source-groundedness
- rebuild equivalence

Operational metrics:

- cold-start time
- idle RSS
- ingest peak RSS
- P50 and P95 write latency
- P50 and P95 search latency
- LLM calls per 100 input messages
- embedding calls per 100 input messages
- disk growth
- number of required services
- backup and restore duration
- ARM64 and AMD64 deployment result

## Decision Rule

Hard gates are evaluated first. Among candidates that pass every hard gate, the
default is selected using this weighting:

| Area | Weight |
|---|---:|
| Recall and current-state correctness | 30% |
| Isolation, deletion, and lifecycle correctness | 30% |
| Operational simplicity and portability | 20% |
| Latency and resource cost | 10% |
| API stability, license, and maintainability | 10% |

No candidate wins from GitHub popularity, vendor benchmark claims, or feature
count alone.

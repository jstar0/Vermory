# Vermory Product Constitution

Status: review candidate

Date: 2026-07-11

## 1. Product

Vermory is a governed AI memory and context platform for multiple models, clients, workspaces, conversations, and everyday assistant workflows.

It is responsible for the complete semantic loop:

```text
real interaction or fact source
-> recognize the current continuity
-> identify information worth retaining
-> preserve source and change history
-> maintain current, historical, and forgotten state
-> retrieve what is relevant now
-> deliver bounded context to an AI or human
-> observe the result
-> update memory under governance
```

Vermory is more than a memo store. It manages what should be remembered, where it belongs, when it is valid, why it is trusted, how it changes, when it must be forgotten, and how it becomes useful context.

The platform must be useful inside one long session and across sessions, clients, models, devices, and time.

## 2. User-Visible Contract

Normal use must remain close to the user's existing AI workflow.

The intended experience is:

- entering a recognized workspace reconnects the relevant working continuity;
- changing AI coding clients does not discard that workspace continuity;
- changing workspaces does not carry unrelated project context;
- an ongoing everyday matter can continue across supported conversation surfaces;
- temporary conversation instructions do not silently become permanent preferences;
- updated facts replace stale current behavior without erasing explainable history;
- deleted facts do not return through exact, paraphrased, semantic, cached, or adapter-backed recall;
- uncertain binding is surfaced or deferred instead of guessed;
- current repository and source reality can correct stale remembered descriptions;
- users can inspect, correct, connect, split, rebind, export, and forget memory without editing database rows.

Client integration should feel automatic in ordinary use. Administrative controls exist for ambiguity, correction, privacy, and diagnostics, not as mandatory steps in every interaction.

## 3. Continuity Contracts

Continuity is the primary boundary. Content classification is secondary.

### 3.1 Workspace-backed continuity

Workspace-backed continuity applies when a stable workspace anchor exists.

Constitutional rules:

- the same recognized workspace continuity is shared across supported clients;
- different workspace continuities are isolated by default;
- wrong automatic binding is worse than a missed binding;
- ambiguous binding must abstain, ask, or operate without durable attachment;
- rename, move, worktree, mirror, and device migration must not inherently create a new continuity;
- live source state and remembered intent are distinguished rather than collapsed into one authority score;
- when descriptive workspace memory conflicts with recognized live source state, the live state governs the description;
- when user intent conflicts with current implementation state, both may be relevant and must not be silently merged.

### 3.2 Conversation-backed continuity

Conversation-backed continuity applies to an ongoing matter without a stable workspace anchor.

Constitutional rules:

- a thread, topic, person, channel, or explicitly named matter may provide an anchor;
- semantic similarity alone cannot merge two matters;
- cross-thread and cross-channel linking is conservative and visible;
- users can connect, separate, rename, or close matters;
- one matter may span channels without pooling unrelated channel history;
- unrelated matters remain isolated even when they share vocabulary, dates, shortlists, or participants.

### 3.3 Global defaults

Global defaults are a thin, strongly governed layer for settings that remain valid across contexts.

Constitutional rules:

- global defaults are not a general personal-memory pool;
- ordinary conversation cannot silently promote information into this layer;
- project state, temporary emotions, active errands, and transient task instructions do not belong here;
- creation and change require an explicit or equally strong policy path;
- users can see, change, and delete every global default.

### 3.4 Bridges

Cross-continuity movement is a governed action, not implicit pooling.

The product supports the outcomes represented by promote, link, export, adopt, rebind, split, and merge. The final API names and persistence model are implementation hypotheses.

Every bridge preserves provenance and remains reversible where the underlying action is logically reversible.

## 4. Memory Authority

### 4.1 Authoritative state

The native deployment uses PostgreSQL as the authoritative state boundary.

Authoritative state includes:

- continuity identity and bindings;
- source identity and source versions;
- governed memory identity and current state;
- evidence and change history required for explanation;
- privacy and deletion state;
- records required to rebuild search projections;
- audit evidence that does not violate deletion policy.

The final physical schema is not constitutional.

### 4.2 Disposable projections

Embeddings, lexical indexes, caches, generated context, and optional memory-backend state are derived projections.

They must be:

- removable without semantic loss;
- rebuildable from authoritative state;
- unable to revive deleted or superseded facts as current;
- versioned sufficiently to support migration and rollback;
- excluded from authority and lifecycle decisions.

### 4.3 Third-party frameworks

mem0, MemOS, Supermemory, and future backends may be used through adapters. None may become a mandatory semantic dependency or a second source of truth.

Removing an adapter must not remove the ability to form, govern, retrieve through the native path, explain, update, or forget memory.

### 4.4 Model providers

LLMs and embedding models are replaceable providers.

LLMs may propose, normalize, classify, compare, or rerank information. They may not silently grant authority to their own output, override explicit user intent, bypass continuity isolation, or prevent deterministic deletion.

Provider failure may degrade automation or retrieval quality. It must not lose authoritative observations, corrupt lifecycle state, or bypass safety policy.

## 5. Semantic Separation

The platform maintains a logical separation between:

```text
source       where information came from
observation  the precise event or span observed
memory       a governed item intended for reuse
history      how that item was proposed, changed, contested, or removed
projection   how current eligible memory is searched
delivery     what a specific consumer received for one task
```

This is a semantic contract, not a requirement for six physical tables. An implementation may combine or split storage only if it preserves the observable contract and evidence requirements.

Raw conversation and raw source content are not automatically durable memory. Generated context is not automatically new memory. Embeddings are never memory truth.

## 6. Formation And Use

### 6.1 Formation

Memory formation must distinguish at least:

- explicit user intent;
- recognized live source facts;
- user-managed documents and records;
- reported observations;
- model inference;
- transient or low-value process noise.

The exact taxonomy and state machine are hypotheses. The invariant is that lower-authority inference cannot silently replace higher-authority evidence or intent.

Same-session working memory is supported. Temporary usefulness does not automatically imply durable retention.

### 6.2 Retrieval

Retrieval must enforce continuity, authorization, lifecycle, privacy, and deletion before relevance ranking.

The system must handle both semantic language and exact technical information such as paths, flags, identifiers, model names, error codes, dates, and numeric constraints. The final retrieval algorithm is a measured hypothesis.

### 6.3 Context delivery

Retrieved items are candidates, not the final prompt.

Delivery must consider task intent, consumer capability, source conflicts, validity, redundancy, privacy, and context budget. Normal model-facing content favors semantic usefulness; provenance and diagnostics remain separately inspectable.

### 6.4 Write-back

AI output, tool results, and user corrections return as observations or candidates. They do not bypass formation and governance merely because they were produced during a successful task.

## 7. Lifecycle And Forgetting

The implementation must support these meanings even if the final state-machine names change:

- proposed but not currently trusted;
- currently eligible for reuse;
- replaced by newer information;
- retained historically but not current;
- rejected;
- forgotten.

Ordinary updates preserve enough history to explain change. Authorized privacy deletion is the exception: deleted sensitive content is erased or irreversibly redacted from authoritative content, projections, caches, artifacts within scope, and optional adapters.

Content-free tombstones and deletion audit may remain only when policy permits. Deleted content itself must not remain recoverable through history or diagnostics.

## 8. Zero-Tolerance Invariants

For all labeled hard-gate cases:

- forbidden cross-tenant fact leakage is zero;
- forbidden cross-continuity fact leakage is zero;
- wrong automatic strong-anchor merge is zero;
- ambiguous strong-anchor resolution does not guess;
- deleted target facts never appear in exact, paraphrased, semantic, related-topic, cached, historical-default, or optional-adapter responses;
- superseded facts are never represented as current;
- model inference never overrides explicit user intent or recognized live authoritative description without governed resolution;
- rebuilding projections preserves all active retrieval assertions;
- duplicate event delivery does not duplicate authoritative memory effects;
- provider failure never bypasses authorization, scope, lifecycle, privacy, or deletion controls.

Zero leakage means the forbidden target fact is absent. A related query may still return unrelated, independently valid information.

## 9. Evidence And Completion

No internal component test, mock provider run, backend pass rate, generated report, or polished interface proves the platform complete by itself.

Vermory qualifies as a usable memory platform only when:

- all three continuity contracts have complete real-client paths from input through reuse and write-back;
- public and sealed hard-gate cases pass;
- native operation does not require a third-party memory framework;
- real downstream tasks demonstrate utility beyond simpler baselines or materially stronger governance at comparable utility;
- user correction burden, stale use, leakage, deletion, failure recovery, and migration have reproducible evidence;
- measured limitations are reported rather than hidden behind aggregate scores.

Quality thresholds, performance SLOs, and scale profiles are calibrated by the Reality Program and do not belong to the immutable constitution before evidence exists.

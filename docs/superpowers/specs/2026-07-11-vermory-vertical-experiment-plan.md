# Vermory Vertical Experiment Plan

Status: review candidate

Date: 2026-07-11

## 1. Purpose

This plan turns the Product Constitution and Reality Program into the next evidence-producing work.

It is not a final implementation checklist. It defines experiments that may revise the Hypothesis Register before schema version 1 or stable client APIs are frozen.

The plan rejects this order:

```text
finish final schema
-> finish memory formation
-> finish retrieval
-> finish APIs
-> connect real clients
```

It uses this order:

```text
freeze a real trajectory
-> implement the smallest complete path through all required layers
-> run it through a real client
-> compare baselines
-> inspect failures
-> revise hypotheses
-> add a materially different trajectory
```

## 2. Complete Path Definition

Every qualifying vertical experiment includes:

```text
real or verified source input
-> continuity resolution
-> source or observation capture
-> candidate formation
-> governance outcome
-> authoritative persistence
-> searchable projection
-> context preparation
-> real model or client consumption
-> downstream artifact or answer
-> post-task observation/write-back
-> at least one update, expiry, supersession, rejection, or deletion action
-> evidence report and baseline comparison
```

An experiment that stops at an internal Go function, generated packet, mock provider, or database row is incomplete.

## 3. Experiment 0: Evidence Bootstrap

### Objective

Freeze the first cross-line discovery batch without designing cases around the current schema.

### Work

- Select varied authorized workspace, conversation, global-default, bridge, and security trajectories according to the Reality Program.
- Preserve raw source snapshots outside model-facing fixtures.
- Anonymize private identifiers while retaining structural difficulty.
- Write expected current facts, forbidden facts, allowed ambiguity, downstream tasks, and artifact checks.
- Record fixture hashes before implementation changes.
- Establish a separate service, account, or execution boundary that the implementation session cannot read.
- Implement an evaluator contract that can score a versioned endpoint or binary without exposing sealed answers. Until that boundary exists, label external local holdouts `withheld_local`, not sealed.
- Run no-context, full-history, summary, and plain-retrieval baselines where the current harness permits.

### Exit evidence

- At least one positive and one negative case for each constitutional invariant used by Experiment 1.
- Public fixture manifests and hashes.
- Sealed evaluator boundary demonstrated with a dummy implementation that cannot read expected outputs, or explicitly reported as not yet available without substituting a weaker local holdout claim.
- Baseline blockers explicitly classified rather than silently skipped.

### Non-goals

- No final memory schema.
- No final taxonomy.
- No release score.
- No claim that seed counts represent production coverage.

## 4. Experiment 1: Three-Line Tripod

### Objective

Prove that one emerging memory engine can support workspace, conversation, and global-default behavior without implementing three unrelated products.

Experiment 1 contains three linked vertical paths. They may share migrations and services, but each path must reach a real consumer.

### 4.1 Workspace path

Select a real repository trajectory containing:

- a stable project decision;
- a changed or conflicting live source;
- a precise path, flag, identifier, or command;
- work performed across at least two real coding clients;
- a post-task update;
- a re-entry or binding check;
- a forget or supersession check.

Run at least no-context, plain summary, plain retrieval, mem0 where applicable, and native Vermory conditions against the same downstream task.

The evidence must include actual client input, output, tool actions, repository diff or artifact, binding result, recalled context, and write-back result.

If a second real coding client is unavailable, the path may test single-client continuity but cannot satisfy or advertise the cross-client result.

### 4.2 Conversation path

Select a real or faithfully replayed long-running matter containing:

- multiple turns and time points;
- a correction or shortlist change;
- a similar but unrelated matter as a distractor;
- one channel, thread, or topic ambiguity;
- a downstream decision or action;
- a deletion tested through exact and paraphrased queries.

Use a Web Chat/API contract simulator backed by a real model provider. The simulator must implement thread identity, turns, context preparation, response persistence, and post-turn write-back rather than issuing isolated prompts.

### 4.3 Global-default path

Use paired cases:

- one explicit stable preference or policy that should become globally reusable;
- one similar temporary instruction or event that must remain local.

Consume the accepted default in both a chat and a workspace task. Verify that the temporary item does not appear outside its continuity. Then update or delete the global default and replay both consumers.

### Shared implementation rule

Implement only the physical schema, transitions, indexes, and APIs required to complete these three paths. Every addition maps to a Hypothesis Register item and frozen case requirement.

Schema migrations are production-shaped and reversible, but schema version 1 is not declared.

### Exit evidence

- All constitutional hard gates pass for Experiment 1 public cases.
- The sealed runner executes without exposing answers; sealed failures may remain but are classified.
- Each line reaches a real model/client and produces a downstream result.
- Baseline comparisons include raw case counts and correction burden.
- Every changed hypothesis records supporting and conflicting evidence.
- The design review identifies unnecessary entities and missing boundaries before Experiment 2.

## 5. Experiment 2: Domain And Wording Transfer

### Objective

Test whether the Experiment 1 design generalized or merely fit its first repositories and conversations.

### Work

- Add materially different workspace domains, repository layouts, languages, and client combinations.
- Add conversation matters with different participants, durations, ambiguities, and privacy pressure.
- Add global-default cases with negation, temporary overrides, and preference change.
- Mutate names, paths, identifiers, ordering, dates, and distractors without changing constitutional expectations.
- Run retrieval ablations: lexical, vector, candidate hybrid strategies, and optional reranking.
- Run candidate-formation ablations: deterministic only, provider-assisted, and policy variants.
- Exercise duplicate delivery, provider timeout, index loss, and PostgreSQL restart within complete paths.

### Exit evidence

- Public and sealed results distinguish implementation defects from hypothesis defects.
- Memory-kind, retention, lifecycle, physical-schema, and API hypotheses are revised or supported based on cross-domain behavior.
- One embedding or index rebuild rehearsal completes without semantic loss.
- Quality and latency profiles are proposed from measured baselines, with denominators and hardware recorded.

### Schema gate

Schema version 1 may be proposed only after Experiment 2 if:

- all physical entities have observed independent behavior;
- common real corrections do not require ad hoc state exceptions;
- deletion and source-history behavior are coherent;
- all three continuity lines use shared semantics;
- migration and rollback have executable evidence.

If these conditions fail, revise the schema hypothesis and continue with another evidence batch.

## 6. Experiment 3: Bridge And Longitudinal Use

### Objective

Exercise continuity over time and across interaction surfaces rather than only replaying short prepared trajectories.

### Work

- Promote a conversation matter into a workspace without importing conversational noise.
- Link multiple conversation anchors and later split an incorrect link.
- Rebind a moved, renamed, worktree, mirror, or device-migrated workspace.
- Export a bounded view to another consumer without merging its continuity.
- Continue one matter through a real OpenClaw or comparable everyday-assistant integration rather than only an isolated API prompt.
- Run a sustained dogfood period with working-memory expiry, durable promotion, correction, and user inspection.
- Measure memory growth, stale-state accumulation, user correction burden, and repeated rediscovery.

### Exit evidence

- Bridge operations preserve provenance and do not create hidden pooling.
- Working state expires or promotes according to measured policy behavior.
- Longitudinal reports include false memories, missed memories, unnecessary context, and user corrections.
- Client friction is measured in normal use rather than inferred from API shape.

## 7. Experiment 4: Operational Qualification

### Objective

Qualify supported deployment profiles after real utility is established.

### Work

- Freeze `developer-local`, `self-hosted-team`, and later `server-qualification` profiles from measured demand.
- Set record, concurrency, latency, recovery, and resource targets per profile.
- Test concurrent retrieval, formation, update, deletion, and rebuild.
- Test schema migration, backup, restore, rollback, and embedding migration.
- Test optional adapter outage, deletion propagation, destruction, and rebuild.
- Test Linux ARM64 and AMD64 packaging.
- Run poisoning, prompt-injection, tenant-isolation, filter-omission, and deletion attacks.

Candidate stress points such as 100,000 active memories, 1,000,000 projections, 10,000 similar records, and 50 clients are adopted only when assigned to a named profile with reference hardware and rationale.

### Exit evidence

- Every supported profile has explicit SLOs and hardware assumptions.
- Hard gates remain zero-tolerance under concurrency and failure.
- Quality, latency, and cost tradeoffs are reported separately.
- Unsupported scale and deployment modes are stated.

## 8. Experiment Review Loop

After each experiment:

1. Verify artifacts and raw case counts.
2. Classify every failure as product, hypothesis, implementation, provider, fixture, evaluator, or infrastructure.
3. Promote appropriate failures into public regression cases only after sealed replacements exist.
4. Update hypothesis status and evidence links.
5. Remove speculative entities and code that no longer serve an accepted or testing hypothesis.
6. Decide the next smallest evidence batch that can discriminate between remaining hypotheses.

No aggregate score may hide a constitutional hard-gate failure.

## 9. Immediate Implementation Boundary

After review approval, implementation begins with Experiment 0 and the smallest shared path required for Experiment 1.

It does not begin by:

- creating every candidate table;
- implementing every memory kind;
- integrating every provider or backend;
- running every public benchmark;
- building the final management UI;
- claiming production scale from synthetic data.

It begins by making real workspace, conversation, and global-default cases executable and then building enough shared system behavior to complete them through real consumers.

# Vermory Experiment 0 Readout

Date: 2026-07-12

Run ID: `experiment-0-v1`

## Result

Experiment 0 passes its evidence-freezing gate.

This result means:

- four first-batch reality cases are structurally valid and byte-frozen;
- every case declares current facts, forbidden facts, source provenance, anchors, pressures, and deterministic downstream checks;
- readable local evidence is identified as `public`, never as sealed;
- the constitutional hard-gate definitions are carried into the next experiment;
- incomplete target coverage is recorded as a limitation rather than converted into an arbitrary failure.

This result does not mean that a production Vermory memory engine has executed or passed the four cases. Experiment 0 deliberately freezes the questions and forbidden behavior before selecting the final schema, lifecycle, retrieval algorithm, API, SLO, or scale profile.

Machine-readable artifacts:

- `artifacts/experiment-0/experiment-0-v1/report.json`
- `artifacts/experiment-0/experiment-0-v1/report.md`
- `artifacts/reality-validation/experiment-0-public-v1/report.json`
- `artifacts/reality-validation/experiment-0-public-v1/report.md`

## Frozen Cases

| Case | Evidence | Continuity | Real pressure | Fixture lock SHA-256 |
|---|---|---|---|---|
| `W01-synapseloom-continuity` | Authorized historical transcript excerpt plus current repository excerpt | Workspace | thread merge, user correction, stale runtime fact, repository authority, cross-session handoff | `9abe7bbaa632454cbab61c76551a4a6e3ade5133104f37bbd12347582b0fcf09` |
| `C01-device-maintenance-continuity` | Authorized and anonymized device-maintenance transcript excerpt | Conversation | diagnostic revision, exclusion boundary, narrow authorization, failed action correction, verified final state | `d63c7b2ab8862f35874b98b7b5eb363d2725842b8f281db013eed2240f0e8328` |
| `G01-language-default-local-override` | Managed global rule excerpt plus a real MCM task language override | Global Defaults | stable preference, explicit local override, scope expiry, default-pollution rejection | `2216c391e6994afd777101fa330e7d9c03023139b569eed6e0783861580af5c3` |
| `S01-deletion-and-source-injection` | Fully synthetic security trajectory | Conversation, Global Defaults, Security | source injection, scope escalation, deletion, exact recall, paraphrased recall, related-topic recall | `e97e58c512f608ee6eddab8cf15891959f4bb453a2eb6de5592cff3faec7133c` |

The first three cases are minimized from authorized real workflows. The safety case is synthetic because real credentials and private deletion targets must not enter a public repository.

## Evidence Infrastructure

The committed evidence contract now enforces:

- strict JSON decoding with unknown-field rejection;
- only `public` and `withheld_local` evidence levels for locally readable cases;
- rejection of a local case claiming `sealed` evidence;
- declared source authorization and anonymization statements;
- unique source and event IDs;
- contiguous event ordering and valid source references;
- clean relative fixture paths that remain under the case directory, including symlink escape rejection;
- fixture SHA-256 verification;
- non-empty current and forbidden behavior with exact conflict detection;
- deterministic `fixture-lock.json` generation;
- post-freeze mutation detection;
- JSON and Markdown validation reports.

External sealed infrastructure has a separate verification boundary:

- `reality/schema/attestation.schema.json` defines the signed payload;
- `reality-attestation-verify` verifies Ed25519 signatures and rejects payload mutation, missing run identity, unknown fields, and future versions;
- no local command signs an attestation or upgrades a readable case to sealed evidence.

No external sealed evaluator was available for this run. The report therefore records `sealed_status=unavailable`, `sealed=0`, and makes no sealed-quality claim.

## Supporting Evidence

The earlier backend bake-off remains valid supporting evidence for retrieval substrates:

- native PostgreSQL plus pgvector, mem0, MemOS, and Supermemory passed the existing lifecycle contract;
- the current B01-B10 backend suite recorded `56/56` assertions with zero forbidden leakage;
- native PostgreSQL and mem0 completed the existing 1,000-record exact-identifier load comparison;
- ARM64 and AMD64 execution evidence exists for the tested backend surfaces.

That evidence supports H-002 and H-015. It does not prove formation quality, continuity resolution, governed updates, task-aware context composition, real-client utility, or the four new reality cases.

The legacy evaluation harness also contains executable conditions for:

- no context;
- stale context;
- plain summary;
- the historical `contextmesh_packet` condition.

Historical provider runs exist for mock, Duojie models, SiliconFlow Qwen models, and selected probes. They remain legacy self-case evidence. They are not results for W01, C01, G01, or S01.

The local MemOS three-client round packs contain scenario definitions but no longer contain the raw run records needed to prove execution. Experiment 0 therefore labels them `translated_proxy`. The historical ContextMesh self-case is labeled `inspired_case`.

## Hypotheses Entering Experiment 1

Four hypotheses move from `proposed` to `testing` because frozen cases now discriminate their behavior:

- `H-003` logical source-observation-memory separation: W01 and S01 require source, historical observation, governed state, deletion, and valid related information to remain distinct.
- `H-005` revision-oriented updates: W01, C01, and S01 require correction, verified replacement, and deletion semantics rather than silent overwrite.
- `H-007` retention classes: G01 and S01 require local instructions to expire and untrusted source text not to promote into Global Defaults.
- `H-008` lifecycle state machine: C01 and S01 require failed, corrected, verified, active, and deleted meanings to remain distinguishable.

No hypothesis becomes `supported` from case creation. The candidate physical schema, memory taxonomy, exact lifecycle names, hybrid ranking, prepare/commit API, outbox, SLOs, and scale profiles remain unfrozen.

## Next Experiment Boundary

Experiment 1 must implement one production-shaped slice against the frozen evidence rather than inventing new success criteria after implementation. At minimum it must execute:

- source ingestion and event replay;
- continuity attachment or explicit abstention;
- candidate memory formation;
- governed current-state and deletion handling;
- retrieval and context composition;
- downstream model consumption;
- post-task write-back;
- deterministic scoring against current and forbidden facts;
- relevant comparisons against no context, full history, plain summary, ordinary vector RAG, mem0, and Vermory.

The first real-client execution should cover one coding client and one Web Chat/API conversation simulator. A dedicated bridge case, a `withheld_local` holdout, and an external sealed evaluator remain separate evidence requirements.

## Limitations

- The first batch has four cases, below the broader discovery target.
- Bridge continuity has no dedicated first-batch case.
- No `withheld_local` holdout is included.
- No genuine external sealed evaluation was run.
- No production memory implementation or real client consumed these cases in Experiment 0.
- The six comparison conditions have not yet run on the new cases.
- Open-ended downstream utility has not yet been judged from real artifacts.
- Load, latency, concurrency, failure recovery, backup, migration, and multi-tenant behavior remain backend or design evidence rather than full-platform evidence.

These limitations are part of the result. They are not hidden by the Experiment 0 pass flag.

# Legacy ContextMesh Evaluation Matrix

> Historical self-case evidence retained for reproducibility. This document does not rank the models for Vermory and does not report results for the frozen Experiment 0 reality cases.

## Purpose

This document records the model and client consumption matrix for the legacy direct-provider harness. Unlike one-off smoke runs, it exercises the same packet contract across multiple compatible model targets and client harnesses.

## Self-Case Task Set

- `self-case-stale-context`
- `self-case-domestic-scope`
- `self-case-architecture-core`
- `self-case-real-case-policy`
- `self-case-artifact-evidence`

## Matrix Command

### Mock matrix

```bash
go run ./cmd/vermory eval-matrix \
  --provider mock \
  --models mock-a,mock-b \
  --run-id matrix-mock-full \
  --artifact-root ./artifacts-provider-smoke \
  --max-tokens 64
```

### Duojie core matrix

```bash
DUOJIE_API_KEY='***' \
go run ./cmd/vermory eval-matrix \
  --provider duojie \
  --models gemini-3-flash,gemini-3.1-pro,glm-5 \
  --run-id duojie-matrix-core \
  --artifact-root ./artifacts-provider-smoke \
  --max-tokens 256
```

## Artifact Layout

- `matrix-runs/<run-id>/report.md`
- `matrix-runs/<run-id>/report.json`
- `matrix-runs/<run-id>__<model>__<task-id>/...`

Each task directory contains a full four-baseline evaluation report with:

- `input.md`
- `packet.md` for packet baseline
- `output.md`
- `raw.json`
- `score.json`
- `report.md`

## Internal Ready Casebook Artifacts

The Internal Ready slice adds executable casebook runs for the three core continuity tracks.

Workspace casebook runs write:

- `casebook-runs/<run-id>/workspace/report.json`
- `casebook-runs/<run-id>/workspace/report.md`
- `casebook-runs/<run-id>/workspace/platform-report.md`
- `casebook-runs/<run-id>/workspace/<baseline>/input.md`
- `casebook-runs/<run-id>/workspace/<baseline>/output.md`
- `casebook-runs/<run-id>/workspace/<baseline>/score.json`
- `casebook-runs/<run-id>/workspace/contextmesh_packet/packet.md`

Conversation casebook runs write:

- `casebook-runs/<run-id>/conversation/report.json`
- `casebook-runs/<run-id>/conversation/report.md`
- `casebook-runs/<run-id>/conversation/conversation/<thread-id>/input.md`
- `casebook-runs/<run-id>/conversation/conversation/<thread-id>/output.md`
- `casebook-runs/<run-id>/conversation/conversation/<thread-id>/score.json`
- `casebook-runs/<run-id>/conversation/conversation/<thread-id>/raw.json` when the provider returns raw response evidence

Bridge casebook runs use the same four-baseline artifact shape as workspace runs and additionally record the governed bridge action metadata in `report.json`:

- `casebook-runs/<run-id>/bridge/report.json`
- `casebook-runs/<run-id>/bridge/report.md`
- `casebook-runs/<run-id>/bridge/platform-report.md`
- `casebook-runs/<run-id>/bridge/<baseline>/input.md`
- `casebook-runs/<run-id>/bridge/<baseline>/output.md`
- `casebook-runs/<run-id>/bridge/<baseline>/score.json`
- `casebook-runs/<run-id>/bridge/contextmesh_packet/packet.md`

Internal-ready acceptance summaries write:

- `acceptance-reports/<run-id>/report.json`
- `acceptance-reports/<run-id>/report.md`

The current minimum acceptance slice has real gates for all three lines:

- workspace: continuation, isolation, and groundedness from the `contextmesh_packet` baseline
- conversation: continuation, isolation, and groundedness from the chat contract runner score
- bridge: signal retention, groundedness, and target fitness from the `contextmesh_packet` baseline

Casebook suite runs write:

- `casebook-suite/<run-id>/report.json`
- `casebook-suite/<run-id>/report.md`

The suite runner executes every case directory under the selected `case-root`, infers the continuity line from the case id prefix, and records per-case status, line, and report URI. The current smoke run against `casebook/cases` executes 16 cases with 0 failures under the mock provider.

Benchmark coverage reports write:

- `benchmark-coverage/<run-id>/report.json`
- `benchmark-coverage/<run-id>/report.md`

The benchmark coverage runner validates that all named public benchmarks are at least `translated_task`, at least 4 reach `executable_evaluation`, and executable benchmarks name concrete case ids. It reports translated proxies, design mappings, and registered original executions as separate counters. Every original evidence path must load a valid execution manifest and qualification; a path string alone is rejected. The current map covers 11 public benchmarks, 8 executable translated evaluations, 3 design mappings, and 1 qualified original-data sample execution.

Internal Ready reports write:

- `internal-ready/<run-id>/report.json`
- `internal-ready/<run-id>/report.md`

The `internal-ready` command runs the casebook suite and benchmark coverage chain, then checks these gates:

- at least 15 main cases are defined
- at least 10 casebook cases execute
- all named benchmarks are translated and at least 4 are executable
- workspace, conversation, and bridge all have executed coverage
- JSON/Markdown artifacts exist for the suite and benchmark reports

The current mock smoke result is `pass=true`, `cases=16`, `executed=16`, `benchmarks=11`, and `executable_benchmarks=8`.

## Internal Ready Commands

Run the full Internal Ready chain:

```bash
go run ./cmd/vermory internal-ready \
  --artifact-root ./artifacts \
  --run-id internal-ready-smoke \
  --provider mock \
  --model mock-model \
  --case-root casebook/cases \
  --benchmark-map casebook/benchmarks/public-benchmark-map.json
```

Run only the executable casebook suite:

```bash
go run ./cmd/vermory eval-casebook-suite \
  --artifact-root ./artifacts \
  --run-id casebook-suite-smoke \
  --provider mock \
  --model mock-model \
  --case-root casebook/cases
```

Run only benchmark coverage:

```bash
go run ./cmd/vermory benchmark-coverage \
  --artifact-root ./artifacts \
  --run-id benchmark-coverage-smoke \
  --map-path casebook/benchmarks/public-benchmark-map.json
```

## Current Status

- Mock matrix: completed
- Duojie core matrix: completed
- Internal Ready mock chain: completed
- LongMemEval original oracle sample with Grok: completed as `dataset_sample`
- Explicit source revision runtime with Grok MCP: completed
- Governed source conflict candidate runtime with Grok MCP: completed
- Provider-assisted unkeyed source target matching with Grok MCP: completed
- Governed multi-fact trusted-document formation with Grok MCP: completed

## Completed Runs

- Mock matrix run ID: `matrix-mock-full`
- Mock regression run ID: `matrix-mock-regression`
- Duojie parallel smoke run ID: `duojie-matrix-parallel-smoke`
- Duojie core matrix run ID: `duojie-matrix-core-v3`
- SiliconFlow Qwen core matrix run ID: `siliconflow-matrix-qwen-core`
- Casebook suite smoke run ID: `casebook-suite-smoke`
- Benchmark coverage smoke run ID: `benchmark-coverage-smoke`
- Internal Ready smoke run ID: `internal-ready-smoke`
- LongMemEval original sample run ID: `longmemeval-original-sample-grok-20260714-attempt-6`
- Source revision Grok session: `955B4CA6-68EB-4D0E-9CB4-96BE91AC1776`
- Source candidate Grok session: `019f5f3c-d836-7d80-a8de-995dcde29ef8`
- Source candidate stale-probe session: `019f5f3e-4b7f-7510-8cda-a26e0ba89725`
- Unkeyed source matching coder session: `019f5fac-feb7-7cf0-a762-15aab01c705a`
- Unkeyed source matching stale-probe session: `019f5fae-1a0e-7fb0-b72d-6a726f5e9815`
- Governed document formation coder session: `019f6019-63fc-78e2-8eb6-41ddfabe62d8`
- Governed document formation stale-probe session: `019f601a-fdfe-7bb0-96ac-376ba8b99515`

## Explicit Source Revision Runtime

The frozen `106-workspace-source-revision` case replaces one named release
command from a newer trusted source while preserving an independent `800 ms`
timeout. A real Grok `grok-4.5` coding task consumed the current workspace
context through MCP, generated and deterministically checked
`release-source-check.md`, and wrote the task result back as `proposed`.

PostgreSQL and real MCP probes established:

- the old command is `superseded` and has no search projection;
- the replacement is `active` and points to the old memory;
- the independent timeout remains `active`;
- the other-workspace distractor is absent from all three deliveries;
- target-task, exact-stale, and paraphrased-stale deliveries contain no stale command;
- projection rebuild preserves three active documents and the same fingerprint.

Official Codex CLI attempts were retained but do not count as a success for
this slice because unsupported model selections and then the account usage
limit stopped each run before MCP. See [the scoped runtime evidence](evidence/2026-07-14-source-revision-runtime.md).

## Governed Source Conflict Candidate Runtime

The frozen `107-workspace-source-conflict-candidate` case uses stable key
`release.signing.mode`. A first source change was proposed and rejected without
changing current retrieval. A second proposal was accepted, atomically
superseding the old keychain fact while preserving the independent `800 ms`
timeout and the rejected candidate as audit history.

A real isolated Grok `grok-4.5` task consumed the accepted workspace through
MCP, created and deterministically checked `release-signing-check.md`, and
wrote its result back as `proposed`. Persisted delivery bodies, not model
self-report, established that:

- proposal alone kept the old current fact and excluded the candidate;
- cross-tenant acceptance was rejected;
- rebuild excluded proposed, rejected, and superseded source states;
- the target task included OIDC and `800 ms` but no keychain or other-tenant fact;
- exact and paraphrased stale probes returned OIDC and no stale fact;
- the client result remained a non-authoritative proposed observation.

This is deterministic keyed formation, not arbitrary-document extraction or
general semantic conflict matching. See
[the scoped runtime evidence](evidence/2026-07-14-source-conflict-candidate-runtime.md).

## Provider-Assisted Unkeyed Source Target Matching

The frozen `108-workspace-unkeyed-source-target-match` case removes the trusted
ingestor's knowledge of `memory_key` while retaining exact source content and
revision identity. A real Grok `grok-4.5` provider received three current
same-workspace keyed facts and selected `release.signing.mode` for the OIDC
signing revision. The same candidate snapshot produced abstentions for one
ambiguous release-flow source and one unrelated maintenance source.

PostgreSQL and real MCP probes established:

- the provider candidate packet excluded the other tenant;
- all three provider decisions share one durable candidate-set fingerprint;
- matching created only a proposed candidate and left pre-accept context on the
  old keychain fact;
- explicit acceptance activated OIDC and projection rebuild excluded the stale
  target and proposed write-backs;
- a real Grok MCP coder created `release-control-policy.md` with OIDC, `800 ms`,
  and SLSA attestation while excluding keychain and static credentials;
- exact and paraphrased stale probes returned OIDC only;
- source-match audit is RLS protected and excluded from model-facing context.

This is closed-set provider matching, not arbitrary-document extraction,
open-vocabulary conflict discovery, or automatic memory activation. See
[the scoped runtime evidence](evidence/2026-07-14-unkeyed-source-target-matching-runtime.md).

## Governed Multi-Fact Document Formation

The frozen `109-workspace-multifact-document-formation` case supplies one
bounded trusted revision with an unchanged region, an updated retry limit, a
new rollback-approval rule, one embedded instruction, and one uncertain policy
sentence. A real Grok `grok-4.5` provider returned structured formation output;
Vermory verified exact byte spans and the unchanged active-fact snapshot before
creating two proposed candidates and one audit-only unchanged item.

PostgreSQL and real MCP probes established:

- malformed fenced output and incorrect unchanged normalization fail durably
  with zero formation items;
- the injection-only and undecided-policy document abstains with zero items;
- pre-review context retains retry 3 and excludes retry 5 plus rollback;
- explicit acceptance activates only the reviewed retry and rollback facts;
- target-tenant projection rebuild preserves four active documents and an
  identical fingerprint;
- a real Grok MCP coder creates `deployment-control-policy.md` with region,
  retry 5, two-maintainer approval, and signed SLSA while excluding stale,
  injected, cross-tenant, and uncertain text;
- exact and paraphrased stale probes return only current facts;
- formation audit tables are RLS protected, and a real provider forget probe
  leaves zero old/new marker residue in linked run and item text.

The real provider suggested `deploy.rollback.maintainers` while the deterministic
fixture names the semantic slot `deploy.rollback.approvals`. The candidate was
operator-reviewed and usable, but this slice does not claim deterministic
provider-generated ontology naming. See
[the scoped runtime evidence](evidence/2026-07-14-governed-document-formation-runtime.md).

## Production Retrieval Ablation

W08 materializes 48 active, four proposed, four superseded, and four deleted
governed memories across four tenants and eight continuities, then executes 24
frozen retrieval queries through the unchanged lexical runtime, direct
SiliconFlow `BAAI/bge-m3` PostgreSQL/pgvector, and exact-guarded RRF.

| Condition | Hit@1 | Recall@K | MRR | nDCG@K | P95 | Forbidden | Ineligible |
|---|---:|---:|---:|---:|---:|---:|---:|
| `lexical_runtime` | 0.6667 | 0.6875 | 0.6806 | 0.6526 | 2.871 ms | 0 | 0 |
| `vector_pg` | 0.9583 | 1.0000 | 0.9792 | 0.9623 | 126.526 ms | 0 | 0 |
| `hybrid_rrf` | 0.9583 | 1.0000 | 0.9792 | 0.9623 | 130.134 ms | 0 | 0 |

The final run used 152 direct embedding requests and passed active-only
projection set equality plus delete/rebuild result-ID equivalence. The first
provider probe and first complete run remain documented: SiliconFlow rejected
the optional `dimensions` request field, and an ANN index containing filtered
non-active rows changed results after active-only rebuild. Both defects were
fixed before the final run.

The result supports an active-only pgvector production integration slice. It
does not establish an RRF benefit: vector and hybrid quality were identical,
and hybrid added latency. H-009 is therefore `testing/measured`, not accepted as
the product default. See
[the scoped evidence](evidence/2026-07-14-production-retrieval-ablation.md).

## Production Retrieval Runtime

W09 connects the frozen direct SiliconFlow `BAAI/bge-m3` profile to the real
workspace MCP, conversation Web Chat, and authenticated runtime construction
paths while retaining lexical as the default. PostgreSQL projection events,
cursors, active-only vectors, and non-sensitive retrieval audits are durable;
the worker runs under a fixed tenant and restricted PostgreSQL role.

| Runtime gate | Result |
|---|---|
| Real Grok MCP vector consumption and proposed writeback | PASS |
| Real Grok Web Chat linked-conversation answer | PASS after recorded cursor catch-up |
| Shadow delivery byte equality with lexical | PASS |
| Projection-lag degradation | `vector -> lexical / projection_lag` |
| Provider-outage degradation | `vector -> lexical / embedding_unavailable` |
| Vector reset/rebuild result IDs and authority | unchanged |
| Restricted-role no-context and cross-tenant probes | PASS |
| Native dump/restore and restore-side rebuild | PASS |
| New credential-shaped artifacts | 0 |

This is `production_path_integrated`, not `accepted_default`. H-009 remains
`testing` pending a second independent retrieval batch, threshold review,
source-authority ranking, embedding migration, and scale/fault qualification.
See [the scoped evidence](evidence/2026-07-14-production-retrieval-runtime.md).

## LongMemEval Original Sample

The committed original-data evidence uses six frozen records from the official
cleaned oracle artifact. It compares `no_context`, `full_oracle_history`,
`plain_lexical_retrieval`, and `vermory_packet` with the same isolated Grok
reader.

| Condition | Completed | Exact | Mean token F1 | Mean answer recall | Abstention |
|---|---:|---:|---:|---:|---:|
| `no_context` | 6/6 | 0/6 | 0.0263 | 0.0385 | 1/1 |
| `full_oracle_history` | 6/6 | 2/6 | 0.5497 | 0.5727 | 1/1 |
| `plain_lexical_retrieval` | 6/6 | 2/6 | 0.4954 | 0.5154 | 1/1 |
| `vermory_packet` | 6/6 | 2/6 | 0.5201 | 0.5214 | 1/1 |

These are deterministic local sample metrics, not official LongMemEval GPT-4o
judge accuracy. The run retained a multi-session counting failure and an
unresolved source-conflict observation rather than upgrading the result to a
full benchmark claim. See [the evidence document](evidence/2026-07-14-longmemeval-original-sample.md).

## Duojie Core Matrix Findings

Tested models:

- `gemini-3-flash`
- `gemini-3.1-pro`
- `glm-5`

Task coverage:

- 5 real self-case tasks per model
- 4 baselines per task
- 15 task reports total

Observed pattern:

- The three models respond differently to the same packet and task, so they remain useful compatibility-test objects.
- `gemini-3.1-pro` produced forbidden phrases like `invented teams` / `fake timelines` on the real-case-policy task; the evidence is retained rather than replaced by a cleaner run.
- The task-level scores identify packet, runner, assertion, or consumer behavior that needs investigation. They do not choose a product-wide default model.

Notable task-level results from `contextmesh_packet` baseline:

- `self-case-artifact-evidence`
  - `gemini-3-flash`: 1.00
  - `gemini-3.1-pro`: 1.00
  - `glm-5`: 0.33
- `self-case-real-case-policy`
  - `gemini-3-flash`: 0.67
  - `gemini-3.1-pro`: 0.67 with forbidden violations
  - `glm-5`: 0.33

Interpretation:

- The matrix proved the earlier low score on `real-case-policy` and `artifact-evidence` was partly a packet-design issue, not only a model issue.
- After adding explicit self-case claims for `real cases`, `source-bound claims`, `audit logs`, `artifacts`, and `WCEF reports`, the packet baseline improved materially on those tasks.

## SiliconFlow Qwen Matrix Findings

Tested models:

- `Qwen/Qwen3-Coder-30B-A3B-Instruct`
- `Qwen/Qwen3-30B-A3B-Instruct-2507`

Task coverage:

- 5 real self-case tasks per model
- 4 baselines per task
- 10 task reports total

Observed pattern:

- Both models are fully integrated into the current matrix workflow and can be treated as covered SiliconFlow domestic-model test objects.
- Their differing outputs expose where a packet, task wording, or assertion contract needs further scrutiny.
- Both remain valid test targets regardless of isolated-task scores.

Important scope note:

- `deepseek-ai/DeepSeek-V4-Flash` is still part of SiliconFlow test coverage, but at this stage as a self-case/probe-covered target rather than a completed matrix-covered target, because repeated direct runs currently hit upstream timeout/busy behavior.

## Interpretation Rule

The matrix is intended to answer:

- whether a model or client can consume the same governed packet and preserve required current facts
- whether the same integration degrades on stale-context correction, scope retention, architecture wording, evidence wording, or real-case policy wording
- which failure belongs to the packet, runner, assertion contract, or consuming client

The matrix does not by itself prove browser, CLI, or MCP tool integration quality.

## Grok CLI Casebook Evidence

- Provider mode: `grok-cli`, using the locally authenticated `grok` executable with model `grok-4.5`.
- Every request starts as an isolated single turn with cross-session memory, web search, plan mode, and subagents disabled. The harness stores the returned CLI JSON as `raw.json`.
- Workspace run `vermory-grok-workspace-v2` executed the first task of `101-workspace-parallel-repos`. Its no-context baseline scored `0.33`; both the ordinary summary and legacy packet baseline scored `1.00` for declared facts and isolation assertions. The corresponding acceptance artifact `vermory-grok-workspace-acceptance-v2` passed.
- Conversation run `vermory-grok-conversation-v4` executed the first chat-contract task of `201-conversation-housing-search` at `1.00`; it retained the active Seattle, budget, pet, one-bedroom, Fremont, and commute facts without workspace framing. The corresponding acceptance artifact `vermory-grok-conversation-acceptance-v2` passed.
- These runs are consumer-compatibility evidence, not a model ranking or proof that Grok completed a real coding-agent task. The harness deliberately disables tools and browser/search behavior.

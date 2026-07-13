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

The benchmark coverage runner validates that all named public benchmarks are at least `translated_task`, at least 4 reach `executable_evaluation`, and executable benchmarks name concrete case ids. The current benchmark map covers 11 public benchmarks and marks 8 as executable translated evaluations.

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

## Completed Runs

- Mock matrix run ID: `matrix-mock-full`
- Mock regression run ID: `matrix-mock-regression`
- Duojie parallel smoke run ID: `duojie-matrix-parallel-smoke`
- Duojie core matrix run ID: `duojie-matrix-core-v3`
- SiliconFlow Qwen core matrix run ID: `siliconflow-matrix-qwen-core`
- Casebook suite smoke run ID: `casebook-suite-smoke`
- Benchmark coverage smoke run ID: `benchmark-coverage-smoke`
- Internal Ready smoke run ID: `internal-ready-smoke`

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

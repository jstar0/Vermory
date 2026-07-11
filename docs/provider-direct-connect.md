# Vermory Direct Provider Connectivity

> Historical provider-harness evidence. These runs validate direct model access for the legacy ContextMesh self-case; they are not Experiment 0 reality-case results or a ranking of models for Vermory.

## Purpose

This document records the direct provider path retained by the Vermory legacy evaluation harness. It is intentionally independent from aggregation gateways and private control-plane services.

## Runtime Rule

- Secrets are provided only through environment variables.
- Provider URLs and model names are runtime flags, not source-controlled configuration.
- The current implementation uses a direct OpenAI-compatible `/v1/chat/completions` adapter.

## Supported Direct Provider Modes

### 1. SiliconFlow

- Provider flag: `--provider siliconflow`
- Default base URL: `https://api.siliconflow.cn/v1`
- Default API key env: `SILICONFLOW_API_KEY`
- Verified real run:
  - Model: `deepseek-ai/DeepSeek-V4-Flash`
  - Command path: `vermory eval-self-case`
  - Artifact run ID: `siliconflow-deepseek-v4-flash-smoke`
- Verified probe run:
  - Artifact run ID: `siliconflow-probe-selected`
  - `Qwen/Qwen3-Coder-30B-A3B-Instruct`: clean `OK`
  - `Qwen/Qwen3-30B-A3B-Instruct-2507`: clean `OK`
  - `deepseek-ai/DeepSeek-V4-Flash`: timed out in direct probe mode under current client timeout, even though the full self-case run had succeeded earlier

### 2. Duojie

- Provider flag: `--provider duojie`
- Default base URL: `https://api.duojie.games/v1`
- Default API key env: `DUOJIE_API_KEY`
- Verified real runs:
  - Model: `gemini-3-flash`
  - Artifact run ID: `duojie-gemini-3-flash-smoke`
  - Model: `gemini-3.1-pro`
  - Artifact run ID: `duojie-gemini-3.1-pro-smoke`
  - Model: `glm-5`
  - Artifact run ID: `duojie-glm-5-smoke`
  - Probe matrix artifact run ID: `duojie-probe-full`

## Duojie Model Probe Notes

The following quick probes were executed through direct `/v1/chat/completions` calls:

- `gemini-3-flash`: usable, returns standard assistant `content`, upstream reported model alias `gemini-3-flash-preview`
- `gemini-3.1-pro`: usable, returns standard assistant `content`
- `glm-5`: usable, returns `content` plus extra `reasoning_content`, and now passes through the current provider adapter
- `glm-5-turbo`: not acceptable as a clean tool-facing default; probe returned large reasoning text instead of stable final answer
- `glm-5.1`: not acceptable as a clean tool-facing default; probe returned polluted output such as `OK</arg_value>`

Current engineering decision:

- These models are retained as test targets, not merely recommended defaults.
- A model may still be a valid test target even if it is slow, noisy, or currently unstable.
- `glm-5-turbo` and `glm-5.1` remain covered as probe targets, but they are not clean enough for routine tool-facing baseline runs right now.
- `deepseek-ai/DeepSeek-V4-Flash` remains an explicit SiliconFlow test target, but it currently shows upstream timeout/busy risk and should be classified separately from the Qwen pair.

## Verified Commands

### Mock baseline

```bash
go run ./cmd/vermory eval-self-case \
  --provider mock \
  --model mock-model \
  --run-id mock-direct-smoke \
  --artifact-root ./artifacts-provider-smoke
```

### SiliconFlow

```bash
SILICONFLOW_API_KEY='***' \
go run ./cmd/vermory eval-self-case \
  --provider siliconflow \
  --model deepseek-ai/DeepSeek-V4-Flash \
  --run-id siliconflow-deepseek-v4-flash-smoke \
  --artifact-root ./artifacts-provider-smoke \
  --max-tokens 256
```

### Duojie

```bash
DUOJIE_API_KEY='***' \
go run ./cmd/vermory eval-self-case \
  --provider duojie \
  --model gemini-3-flash \
  --run-id duojie-gemini-3-flash-smoke \
  --artifact-root ./artifacts-provider-smoke \
  --max-tokens 256
```

```bash
DUOJIE_API_KEY='***' \
go run ./cmd/vermory eval-self-case \
  --provider duojie \
  --model gemini-3.1-pro \
  --run-id duojie-gemini-3.1-pro-smoke \
  --artifact-root ./artifacts-provider-smoke \
  --max-tokens 256
```

```bash
DUOJIE_API_KEY='***' \
go run ./cmd/vermory eval-self-case \
  --provider duojie \
  --model glm-5 \
  --run-id duojie-glm-5-smoke \
  --artifact-root ./artifacts-provider-smoke \
  --max-tokens 256
```

```bash
DUOJIE_API_KEY='***' \
go run ./cmd/vermory probe-provider \
  --provider duojie \
  --models gemini-3-flash,gemini-3.1-pro,glm-5,glm-5-turbo,glm-5.1 \
  --run-id duojie-probe-full \
  --artifact-root ./artifacts-provider-smoke \
  --prompt '不要输出推理过程、不要任何标签或解释，只回复 OK' \
  --max-tokens 128
```

### SiliconFlow probe

```bash
SILICONFLOW_API_KEY='***' \
go run ./cmd/vermory probe-provider \
  --provider siliconflow \
  --models deepseek-ai/DeepSeek-V4-Flash,Qwen/Qwen3-Coder-30B-A3B-Instruct,Qwen/Qwen3-30B-A3B-Instruct-2507 \
  --run-id siliconflow-probe-selected \
  --artifact-root ./artifacts-provider-smoke \
  --prompt '不要输出推理过程、不要任何标签或解释，只回复 OK' \
  --max-tokens 128
```

## Artifact Paths

- `artifacts-provider-smoke/platform-runs/mock-direct-smoke`
- `artifacts-provider-smoke/platform-runs/siliconflow-deepseek-v4-flash-smoke`
- `artifacts-provider-smoke/platform-runs/duojie-gemini-3-flash-smoke`
- `artifacts-provider-smoke/platform-runs/duojie-gemini-3.1-pro-smoke`
- `artifacts-provider-smoke/platform-runs/duojie-glm-5-smoke`
- `artifacts-provider-smoke/provider-probes/duojie-probe-full`
- `artifacts-provider-smoke/provider-probes/siliconflow-probe-selected`

Each run stores:

- `input.md`
- `packet.md` for packet baseline
- `output.md`
- `raw.json`
- `score.json`
- `report.md`

## Scope Boundary

These runs prove:

- the retained harness can call direct providers without an aggregation gateway
- real provider outputs can be captured into repeatable artifacts
- the four-baseline evaluation loop is operational

These runs do not yet prove:

- final WCEF quality
- AI coding tool integration quality
- browser or MCP consumption quality
- strong real-world advantage on difficult project tasks

## Current Test Coverage Status

### Duojie

- Matrix-covered:
  - `gemini-3-flash`
  - `gemini-3.1-pro`
  - `glm-5`
- Probe-covered:
  - `glm-5-turbo`
  - `glm-5.1`

Coverage note:

- `gemini-3-flash`, `gemini-3.1-pro`, and `glm-5` are fully integrated into the current self-case matrix workflow.
- `glm-5-turbo` and `glm-5.1` are still valid compatibility-test objects, but currently only at the probe layer because their outputs are not clean enough for routine matrix use.

### SiliconFlow

- Matrix-covered:
  - `Qwen/Qwen3-Coder-30B-A3B-Instruct`
  - `Qwen/Qwen3-30B-A3B-Instruct-2507`
- Probe-covered:
  - `deepseek-ai/DeepSeek-V4-Flash`
- Self-case covered:
  - `deepseek-ai/DeepSeek-V4-Flash`

Coverage note:

- The two Qwen models have full SiliconFlow matrix coverage.
- `deepseek-ai/DeepSeek-V4-Flash` is explicitly included as a system test target, but current evidence shows upstream timeout/busy risk in repeated direct-run scenarios.

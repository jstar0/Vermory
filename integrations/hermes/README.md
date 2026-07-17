# Hermes Integration

This integration connects the official
[`NousResearch/hermes-agent`](https://github.com/NousResearch/hermes-agent)
memory-provider lifecycle to Vermory conversation continuity.

Hermes calls Vermory before and after each completed agent turn:

```text
Hermes MemoryProvider.prefetch
-> POST /v1/integrations/hermes/turns/prepare
-> governed current context is injected as reference data
-> Hermes produces its answer
-> MemoryProvider.sync_turn
-> POST /v1/integrations/hermes/turns/complete
-> user and assistant observations remain governed in Vermory
```

The provider exposes no model tools. Hermes cannot confirm, correct, forget,
link, promote, or rebind memory through this adapter. Those remain explicit
Vermory governance operations.

## Continuity Identity

Gateway sessions use Hermes' durable `gateway_session_key`. Internal session
rotation and context compression therefore do not split the same messaging
thread. CLI sessions use the Hermes session ID and remain isolated by default.
Hermes and OpenClaw use separate conversation channels even when their raw
session keys are identical; linking them is an explicit bridge action.

## Install On A Hermes Host

On the Mac mini, install the pinned official Hermes checkout as a normal user:

```bash
VERMORY_HERMES_PROXY=http://127.0.0.1:6152 \
  deploy/macos/install-hermes.sh
```

The runner installs under `$HOME/.vermory/hermes`, never invokes `sudo`, uses
the official staged installer to skip Node/browser/desktop dependencies and the
interactive provider wizard, and verifies the exact upstream revision. Then
copy the provider into the active `HERMES_HOME`:

```bash
mkdir -p "${HERMES_HOME:-$HOME/.hermes}/plugins/vermory"
cp integrations/hermes/vermory/__init__.py \
  integrations/hermes/vermory/plugin.yaml \
  "${HERMES_HOME:-$HOME/.hermes}/plugins/vermory/"
```

Start a loopback Vermory service with the external-provider mode, then select
the provider:

```bash
hermes memory setup vermory
hermes memory status
```

The default API URL is `http://127.0.0.1:8787`. The setup flow writes
non-secret settings to `$HERMES_HOME/vermory.json`; an optional client token is
read from `VERMORY_API_TOKEN`. URLs containing embedded credentials, query
parameters, or fragments are rejected.

## Failure Behavior

Prepare failures return no external context and do not block the Hermes turn.
Completion failures do not hide the visible Hermes answer. Responses are
bounded to 256 KiB, and raw HTTP bodies or credentials are never logged.

## Verify

```bash
uv run --project integrations/hermes python -m unittest discover \
  -s integrations/hermes/tests -v
```

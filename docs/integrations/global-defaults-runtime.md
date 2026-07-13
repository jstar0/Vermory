# Global Defaults Runtime

Global Defaults is Vermory's explicit cross-context settings layer. It carries only stable rules that should apply in both workspace-backed and conversation-backed use. It does not learn from ordinary chat, imported documents, model output, or project state.

## User Behavior

When a default exists, Vermory supplies its semantic content to both coder/workspace requests and Web Chat requests. A current task instruction still wins for that task only.

Example:

```text
Global default: reply to user-facing requests in Chinese unless the active task explicitly requests another language.
Current task: produce this table-facing deliverable in English.
```

The current task is handled in English. A later unrelated Chinese request returns to Chinese. The English task does not rewrite the global setting.

## CLI

Create one stable default:

```bash
vermory defaults set \
  --database-url 'postgresql:///vermory?host=/tmp' \
  --tenant-id local \
  --operation-id default-language-v1 \
  --key reply_language \
  --content 'Default user-facing replies to Chinese unless the active task explicitly requests another language.'
```

Inspect the lifecycle:

```bash
vermory defaults inspect \
  --database-url 'postgresql:///vermory?host=/tmp' \
  --tenant-id local
```

Correct a visible active default:

```bash
vermory defaults correct \
  --database-url 'postgresql:///vermory?host=/tmp' \
  --tenant-id local \
  --operation-id default-language-v2 \
  --memory-id '<active-memory-id>' \
  --content 'Default user-facing replies to Chinese with concise Markdown unless the active task explicitly requests another language.'
```

Delete a visible default:

```bash
vermory defaults forget \
  --database-url 'postgresql:///vermory?host=/tmp' \
  --tenant-id local \
  --operation-id default-language-delete \
  --memory-id '<active-memory-id>'
```

## Loopback HTTP

Start the Web Chat runtime with a server-owned tenant:

```bash
vermory web-chat \
  --database-url 'postgresql:///vermory?host=/tmp' \
  --tenant-id local \
  --listen 127.0.0.1:8787 \
  --provider grok-cli
```

The management endpoints are:

- `GET /v1/defaults`
- `POST /v1/defaults/set`
- `POST /v1/defaults/correct`
- `POST /v1/defaults/forget`

Set request:

```json
{
  "operation_id": "default-language-v1",
  "key": "reply_language",
  "content": "Default user-facing replies to Chinese unless the active task explicitly requests another language."
}
```

Correct request:

```json
{
  "operation_id": "default-language-v2",
  "memory_id": "<active-memory-id>",
  "content": "Default user-facing replies to Chinese with concise Markdown unless the active task explicitly requests another language."
}
```

Forget request:

```json
{
  "operation_id": "default-language-delete",
  "memory_id": "<active-memory-id>"
}
```

HTTP clients cannot choose `tenant_id`, `continuity_id`, provider, or model. Unknown authority fields are rejected.

## Precedence And Scope

Runtime precedence is:

1. the active user/task instruction;
2. relevant governed memory in the current workspace or conversation;
3. active Global Defaults.

Global Defaults are supplied as semantic prose only. Keys, memory IDs, lifecycle state, and audit fields are not included in model-facing packets.

Workspace deliveries remain attached to the resolved workspace continuity. Conversation deliveries remain attached to the exact channel/thread continuity. Both paths read the global layer but cannot write to it.

## Governance And Deletion

- `set` fails when the same key already has an active value; use `correct` with the visible active memory ID.
- `operation_id` is an idempotency key and cannot be reused for a different logical mutation.
- `correct` preserves the key and supersedes the targeted active revision.
- `forget` redacts the governed content, its origin observation, search projection, and prior tenant deliveries containing the exact semantic content.
- rebuilding the search projection includes only active revisions and cannot restore deleted content.
- ordinary chat confirmation creates conversation-scoped memory only.
- workspace source import creates workspace-scoped memory only.
- neither path can silently promote data into Global Defaults.

## Acceptance Cases

Automated acceptance covers:

- `G01-language-default-local-override`: stable Chinese default, task-local English override, later Chinese task, workspace and chat replay, correction, deletion, and metadata-free context delivery;
- `S01-deletion-and-source-injection`: untrusted source and ordinary conversation cannot promote into Global Defaults, while deletion remains effective through exact, paraphrased, related-topic, restart, and projection-rebuild probes.

Run the database-backed acceptance suite serially:

```bash
VERMORY_TEST_DATABASE_URL='postgresql:///vermory_test?host=/tmp' \
  go test -p 1 -count=1 ./internal/runtime ./internal/webchat -run 'TestG01|TestS01'
```

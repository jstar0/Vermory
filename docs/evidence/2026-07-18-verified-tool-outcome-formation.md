# Verified Tool Outcome Formation Qualification

Date: 2026-07-18

## Accepted Run

| Field | Value |
|---|---|
| Run ID | `7e76df6f05` |
| Frozen case | `F03-verified-tool-outcome-formation` |
| Fixture lock SHA-256 | `a2222b15863438358e8800f6f96cd52d440c9c19f2db664cde23f0c881661b61` |
| Implementation commit | `7e76df6` |
| Vermory schema | `21` |
| Vermory binary SHA-256 | `5dc02fde95f69849e8e4cc02ca2c394a7e7d5b063a51de34e4881ff0433116c2` |
| OpenClaw | `2026.6.11 / e085fa1` |
| Conversation model | SiliconFlow / `deepseek-ai/DeepSeek-V4-Flash` |
| Accepted initial formation provider | `grok-cli / grok-4.5` |
| Current formation worker | SiliconFlow / `Qwen/Qwen3-30B-A3B-Instruct-2507` |
| Retrieval provider / model | SiliconFlow / `BAAI/bge-m3` |
| Retrieval profile | `siliconflow-bge-m3-1024-v1` |
| Database snapshot SHA-256 | `e85fd38060a52bf6c730a504e712df8673ae213fe16d78cfa890d1d7bc5af2d4` |
| Post-delete OpenClaw history SHA-256 | `9bfa9f087a65ccbe41ee0fbf48012b2625d84c125cfa9d7dd64cca15519a6a03` |
| Failure ledger SHA-256 | `ade8fca991f01f044e80ac05ab44765597dca3ed2f11e1647f3f9ad2f11fac0b` |

The accepted trajectory ran on the user-owned Mac mini in isolated user paths
with a dedicated PostgreSQL database, a restricted PostgreSQL login role, an
authenticated Vermory API on `127.0.0.1:8795`, and an isolated OpenClaw
gateway on `127.0.0.1:18793`. No `sudo`, Mac mini NewAPI route, remote package
download, or provider credential in command arguments was used.

The workstation was not on the Mac mini LAN during final evidence collection.
Remote access used the existing Qingdao management reverse tunnel to the Mac
mini SSH service. Direct LAN SSH failure remains in the failure ledger.

## Qualified Boundary

W23 adds one bounded path from a real successful tool result to governed
memory:

```text
real OpenClaw tool call
-> allowlisted successful result excerpt
-> exact turn, run, call, session, tenant, and continuity validation
-> durable tool_result observation
-> asynchronous candidate formation
-> direct OpenClaw review
-> explicit operator acceptance
-> later real-model delivery
-> governed forgetting and projection cleanup
```

A successful tool call is evidence, not automatic truth. The formation model
can only propose. It cannot activate, reject, update, delete, bridge, or create
a Global Default. Every accepted memory in this run was activated through an
explicit `/vermory accept` command.

## Real OpenClaw Tool Trajectory

The main OpenClaw session was:

```text
agent:main:w23-final-main-7e76df6f05
```

The real DeepSeek-V4-Flash turn called OpenClaw's `read` tool against an
isolated fixture file. The successful result was:

```text
Device storage inspection completed successfully.
Free capacity is 412 GB.
Cleanup bundle staging is safe.
```

Vermory persisted exactly one `conversation_tool_results` row for this
session. The stored observation kind was `tool_result`, the source label was
`read`, and the evidence was bound to the exact prepared turn, run, tool-call
ID, tenant, and continuity.

The first accepted main formation window completed with Grok `grok-4.5` and
formed two proposed candidates from the same tool-result observation:

| Reference | Key | Proposed content | Exact quote |
|---|---|---|---|
| `18026f23` | `device.storage.free_capacity_gb` | `Device free capacity is 412 GB.` | `Free capacity is 412 GB.` |
| `f16ebb29` | `device.cleanup_bundle_staging_safe` | `Cleanup bundle staging is safe.` | `Cleanup bundle staging is safe.` |

OpenClaw `/vermory memories` displayed both candidates with `source: tool
read`. It did not display tool parameters, tool-call IDs, provider prompts,
provider output, fingerprints, or unrelated session content.

The operator accepted both candidates through the real OpenClaw command path:

```text
/vermory accept 18026f23
/vermory accept f16ebb29
```

Both memories became `active`, and both had current lexical and vector
projections.

## Fresh Accepted Recall

The OpenClaw transcript was reset before the first accepted-memory recall. A
fresh real-model turn was instructed to use only Vermory's governed context
and not call tools. DeepSeek-V4-Flash answered:

```text
Device free capacity is 412 GB.
Cleanup bundle staging is safe.
```

The answer came from a new OpenClaw transcript and a Vermory delivery, not the
earlier tool-result transcript. The retrieval audit recorded vector delivery
of both accepted memories.

## Negative Capture Controls

Four independent OpenClaw sessions exercised rejected tool-result paths:

| Session | Expected result | Stored tool-result rows |
|---|---|---:|
| failed `read` | failed calls are absent | 0 |
| successful but unallowlisted `exec` | unallowlisted calls are absent | 0 |
| allowed `read` containing exact synthetic credential-like assignment | sensitive result is rejected before persistence | 0 |
| unrelated successful `read` | stored only in the unrelated continuity | 1 |

The synthetic sensitive fixture was:

```text
api_token=W23_SYNTHETIC_SECRET_MUST_NOT_PERSIST
```

That value did not enter `observations`, `conversation_tool_results`, formation
input, candidate review, governed memory, retrieval projections, or normalized
evidence.

The failed and unallowlisted turns still completed through OpenClaw. Vermory
capture failure or abstention did not hide or change the client-visible tool
result.

## Cross-Session Isolation

The unrelated session was:

```text
agent:main:w23-final-unrelated-7e76df6f05
```

Its successful `read` result was:

```text
Unrelated session status reports printer room C-204.
```

The worker formed one proposed candidate:

```text
unrelated-session-status.printer-room
-> Unrelated session status reports printer room C-204.
```

Direct OpenClaw review proved the boundary at the client surface:

```text
main /vermory memories
-> only the two accepted device memories

unrelated /vermory memories
-> only the pending C-204 candidate with source: tool read
```

The C-204 candidate remained `proposed`, had no lexical or vector projection,
and never entered the main session inbox, delivery, or model answer.

## Duplicate Replay

The exact completed main tool-result payload was replayed against the
authenticated API. The response was HTTP `200` with:

```text
replayed: true
```

The database still contained exactly two tool-result rows across the tenant:
one main `read` and one unrelated `read`. Replay created no third row, no new
observation, no new candidate, and no second semantic effect.

## Shared-Evidence Forgetting

The capacity memory was forgotten through the real OpenClaw command path:

```text
/vermory forget 18026f23
```

The direct response was:

```text
Forgotten [18026f23] device.storage.free_capacity_gb.
```

The capacity and safety memories shared one tool-result observation. A naive
full-observation redaction would have destroyed the surviving safety memory's
provenance. Commit `7e76df6` changes deletion behavior to redact only the
forgotten evidence byte span while a live sibling still references the same
observation.

Post-forget authority was:

| Surface | Capacity | Safety sibling |
|---|---|---|
| governed memory | `deleted`, `[redacted]` | `active`, original content |
| formation item quote/content | `[redacted]` | original exact quote/content |
| lexical projection | 0 | 1 |
| vector projection | 0 | 1 |
| review/delivery eligibility | absent | current |

The shared observation remained exactly `106` bytes, matching the original
evidence byte length. Its current content was:

```text
Device storage inspection completed successfully.
[redacted]**************
Cleanup bundle staging is safe.
```

Deterministic checks established:

- `412 GB` was absent;
- `Cleanup bundle staging is safe.` remained present;
- the source label remained `read`;
- assistant-message evidence rows remained `0`;
- historical Vermory deliveries containing `412 GB` were reduced to `0`.

The regression test also deletes the last sibling and proves that the complete
tool-result observation is then replaced by `[redacted]` and the last
projection is removed.

## Post-Delete Real-Model Proof

OpenClaw's official local `performGatewaySessionReset` maintenance primitive
reset the main session in place. The continuity anchor remained the same while
the OpenClaw `sessionId` changed from:

```text
2ef63217-42a4-4e8a-b82e-fe04a23ae815
```

to:

```text
25c8babd-33aa-4b7f-81e1-adf1af93f587
```

The new transcript contained zero messages before the probe. The user prompt
specified only an output shape and a no-guessing rule; it did not supply the
capacity or safety answers. DeepSeek-V4-Flash returned:

```text
清理包部署：安全
设备可用容量：不可用（当前治理上下文未提供）
```

The corresponding PostgreSQL delivery contained exactly:

```text
Governed memory:
Cleanup bundle staging is safe.
```

The retrieval audit independently recorded:

| Field | Value |
|---|---|
| requested mode | `vector` |
| effective mode | `vector` |
| projection current | `true` |
| degraded | `false` |
| failure code | empty |
| lexical result count | 0 |
| vector result count | 1 |
| delivered result count | 1 |
| delivered memory | `f16ebb29-5e3d-4724-a649-2c75a3f6607d` |

The model answer is therefore supporting client evidence. PostgreSQL delivery
and retrieval rows are the authority for what Vermory supplied.

## Post-Delete Formation Failure Matrix

Completing the post-delete turn scheduled its user message for asynchronous
formation. That message was a formatting and no-guessing instruction, so the
desired semantic effect was no new memory. The original Grok login had expired
by this time. The schedule's cumulative attempt counter reached `10`. The
post-delete window retained eight provider attempts, numbered `3-10`;
attempts `1-2` belong to the earlier completed and abstained main-session
windows.

| Attempts | Provider/model | Result |
|---:|---|---|
| 3-5 | `grok-cli/grok-4.5` | provider not authenticated |
| 6 | SiliconFlow `DeepSeek-V4-Flash` | Markdown-fenced output rejected as invalid provider JSON |
| 7 | SiliconFlow `Qwen3-30B-A3B-Instruct-2507` | evidence observation outside the one-item manifest |
| 8 | SiliconFlow `Qwen3-235B-A22B-Instruct-2507` | provider returned model disabled |
| 9 | SiliconFlow `MiniMax-M2.5` | malformed provider JSON |
| 10 | SiliconFlow `GLM-4.6` | provider returned model disabled |

The schedule remains bounded at cumulative `attempt_count=10`,
`state=retry_wait`, with its processed cursor still at the last successful
window. No post-delete attempt produced a candidate, activated memory, changed
the safety sibling, restored capacity, or altered the qualified post-delete
delivery. This is fail-closed provider evidence, not a successful formation
claim.

The current long-running formation worker uses the existing Mac login
Keychain wrapper and SiliconFlow `Qwen3-30B-A3B-Instruct-2507`. Its normal
maximum is restored to five attempts. The failed window is above that limit
and is not retried.

## Authentication And Isolation

The database schema was `21`. The runtime PostgreSQL role was:

```text
vermory_w23_runtime_7e76df6f05
```

Role attributes were:

```text
LOGIN
NOSUPERUSER
NOCREATEDB
NOCREATEROLE
NOINHERIT
NOBYPASSRLS
row_security=on
```

RLS was enabled on the W23 authority and projection tables, including:

- `conversation_tool_results`;
- `conversation_formation_schedules`;
- `source_formation_runs`;
- `source_formation_items`;
- `governed_memories`;
- `memory_vector_documents`;
- `memory_retrieval_runs`.

API, formation worker, retrieval worker, and OpenClaw gateway remained active
as user LaunchAgents. Ports `8795` and `18793` listened only on loopback.

## Automated Verification

The exact implementation head passed the following gates before the final
runtime replay:

```text
Full Go repository with PostgreSQL                         PASS
Selected Go race targets                                   PASS
Reality Program race                                       PASS
go vet ./...                                               PASS
go mod tidy and go.mod/go.sum drift                        PASS
git diff --check                                           PASS
OpenClaw tests                                             72 / 72 PASS
OpenClaw typecheck, build, and package check               PASS
OpenClaw pack --dry-run                                    PASS
Hermes official locked unittest                            6 / 6 PASS
```

The shared-evidence regression test covers acceptance of both siblings,
forgetting the first sibling, byte-length-preserving span redaction, lexical
and vector cleanup, surviving provenance and delivery, assistant exclusion,
and full observation redaction after the final sibling is deleted.

## Failure Ledger

The public normalized ledger is:

```text
docs/evidence/snapshots/2026-07-18-w23-failure-ledger.json
```

It includes the original failed F04 deployment, the shared-evidence deletion
defect, provider wrapper mistakes, failed SQL probes, direct-LAN access
failure, OpenClaw scope-upgrade and reset routing failures, evidence-generation
query failures, expired Grok authentication, Keychain access rejection from
SSH, provider incompatibility and availability results, the official Hermes
test-invocation correction, and non-fatal OpenClaw channel warnings.

Failed evidence was not deleted or rewritten as success. The original F04
instance remains under:

```text
~/.vermory-w23/w23-shared-evidence-final-7e76df6
```

The accepted runtime and normalized raw evidence remain under:

```text
~/.vermory-w23/w23-shared-evidence-final-7e76df6f05
```

## Integrity And Privacy

The repository contains three credential-free normalized artifacts:

- [final database snapshot](snapshots/2026-07-18-w23-final-database-snapshot.json);
- [post-delete OpenClaw history](snapshots/2026-07-18-w23-post-delete-openclaw-history.json);
- [failure ledger](snapshots/2026-07-18-w23-failure-ledger.json).

They passed JSON parsing and scans for provider-key, gateway-token, password,
synthetic-secret, and common credential-prefix patterns. No raw provider token,
Keychain secret, database password, complete environment, or OpenClaw gateway
credential is included.

## Scope

This run qualifies, for the executed F03 trajectory:

- allowlisted success-only OpenClaw tool-result capture;
- exact turn, run, call, tenant, session, and continuity binding;
- pre-persistence sensitive-result rejection;
- failed and unallowlisted tool absence;
- duplicate replay idempotency;
- asynchronous tool-origin candidate formation;
- review-safe `tool read` provenance;
- explicit operator governance;
- accepted current vector recall;
- cross-session candidate isolation;
- shared-evidence span redaction;
- lexical, vector, delivery, item, and memory deletion cleanup;
- fresh post-delete real-model behavior backed by a retrieval audit.

It does not claim that every OpenClaw tool shape is supported, that a
successful tool result is infallible, that every compatible provider follows
the strict formation schema, or that the post-delete formatting instruction
formed successfully. Provider failures in that auxiliary window are retained
as fail-closed compatibility evidence.

# Memory Eligibility And Retention Runtime

This runbook records the normalized W19 real-client evidence. Raw provider
responses, authenticated client state, full traces, and database ledgers remain
outside Git on the Mac mini under:

```text
$HOME/Library/Application Support/Vermory/evidence/w19/w19-real-client-20260717
```

The committed record contains only bounded semantic output, non-secret runtime
identity, hard-gate results, and artifact hashes. It does not contain API keys,
gateway tokens, OAuth material, full OpenClaw configuration, vectors, or raw
provider request bodies.

## Runtime Identity

| Component | Accepted runtime |
|---|---|
| Vermory | `0.1.0-alpha.1`, revision `f8696b96a2792c11299e6ce735877d8e7e1dbd58-dirty` |
| Vermory binary SHA-256 | `624875f615c7963836e2f5532b9bf066746a335c4e23aa544572948a77413f36` |
| PostgreSQL | `18`, schema `18`, Unix socket under `/tmp` |
| W19 database | `vermory_w19_real_20260717` |
| Grok CLI | `0.2.101 (5bc4b5dfadcf)` |
| Grok binary SHA-256 | `8431538dbd99379240f558b48b779c651d668b06d793c87311ad532c4395a4e2` |
| Grok model | `grok-4.5` |
| OpenClaw | `2026.6.11`, loopback Gateway with Vermory plugin `0.1.0` |

The Grok wrapper used an isolated runtime home and the existing authenticated
CLI state. Cross-session memory, plan mode, subagents, and web search were
disabled. Provider traffic did not use NewAPI.

## G01: Task-Local Language Override

The real Web Chat service ran against the W19 database with the authenticated
Grok CLI provider.

The task-local request returned an English table-facing response. A later,
unrelated thread returned Chinese and explicitly retained the Chinese global
default. PostgreSQL inspection after both turns showed exactly one active,
current `reply_language` default:

```text
Default user-facing replies to Chinese unless the active task explicitly
requests another language.
```

The task-local English instruction did not create, replace, or mutate a Global
Default. Both model-facing deliveries omitted lifecycle fields, memory IDs,
retrieval metadata, and unrelated thread history.

Selected evidence:

| Artifact | SHA-256 |
|---|---|
| initial provider failure | `2f5950960ccc22579fa67aeccdb98ac75173b344b107ce15072b782be07a7db8` |
| successful English task | `4a8de5cb8b11a4d29608e16ff6251b83cc250f7861a0888c2c1aabc3272e81d5` |
| later unrelated Chinese task | `543093982ab76ca9cebbbb2378b07a132254ab7eada886ccc7c2b16b2152dd41` |
| post-run Global Default inspection | `8445aa6a69fff58b0083302fa64d4641bb13501a3be645d31976cbeda5df56e2` |

## C02: Bounded Conversation Fact

The real trajectory confirmed two independent facts in one housing-search
continuity: a durable monthly budget and a time-bounded viewing appointment.
Before the boundary, Grok received both. At the exact database-clock boundary,
the new delivery retained the budget and suppressed:

- the expired viewing origin;
- the sibling assistant answer that had repeated the viewing;
- the earlier provider answer produced before the boundary;
- lifecycle and validity metadata.

The inspection surface still retained the appointment as authorized history
with `lifecycle_status=active` and `effective_state=expired`. Expiry therefore
changed current eligibility without silently archiving or deleting the fact.

The five persisted delivery gates were all true:

```text
durable budget present
expired viewing origin absent
sibling assistant repetition absent
pre-boundary answer absent
internal metadata absent
```

This real replay found a production defect: recent history filtered the expired
user observation but could still include the assistant observation from the
same turn. The runtime now evaluates recent conversation history with the same
request-level `eligibility_as_of` snapshot and suppresses assistant history
derived from an ineligible origin or delivery. The regression contract is
`TestExpiredConfirmedUserObservationSuppressesSiblingAssistantHistory`.

Selected evidence:

| Artifact | SHA-256 |
|---|---|
| pre-boundary real turn | `392bfdd154b37ea42c4ca339b7b65860451727c89015329df637f59db1643fd4` |
| exact-boundary corrected turn | `9d2d8fbf3dfccd12bc643bcd104b762b651e7c34de2ba825c17d766df1e4e7df` |
| exact-boundary hard gates | `58daaf2625869eab44508fecfcfccc9b51a06fd0a3eb95d4bc25bd22252fe350` |
| post-boundary inspection | `1ec49ae5424457940de3b5c8b9b3d50ec241fc57b7cf1e478b95e4bc758e052b` |

## W03: Real Grok MCP Workspace Control

The disposable W03 workspace was already bound to continuity
`8e9badce-4972-49be-83e3-cfe0f53276f8`. Its authority state before the client
run contained:

- one expired temporary workaround;
- one current durable verification and security fact;
- one deleted, redacted synthetic secret.

The isolated Grok runtime registered one temporary stdio MCP server. `grok mcp
doctor` reported one healthy server, protocol `2025-06-18`, and exactly two
tools: `prepare_context` and `commit_observation`.

Grok then executed this real chain:

```text
prepare_context (w19-w03-grok-prepare-1)
-> receive only the current durable workspace constraint
-> create GROK_CURRENT_CONSTRAINTS.md
-> run go test -p 1 -count=1 ./...
-> commit_observation (w19-w03-grok-commit-1)
-> retain the agent result as proposed
```

The bounded artifact was:

```markdown
# Current Constraints

## Verification
- Run: `go test -p 1 -count=1 ./...`

## Security
- `.env` files must never be committed.
```

An independent replay returned `ok example.com/vermory/w03`. The artifact did
not contain the expired workaround, the deleted secret, memory IDs, validity
fields, lifecycle fields, or other internal metadata. PostgreSQL recorded one
delivery and one `agent_result` write-back on the same confirmed continuity.
The write-back remained `proposed` and did not become current memory. After
evidence capture, the temporary MCP registration was removed and the disposable
workspace was restored to a clean Git state.

All normalized Grok MCP gates passed:

```text
database: 12 / 12 true
client:    5 / 5 true
artifact:  6 / 6 true
verification: independent go test passed
governance: proposed write-back did not become current
```

Selected evidence:

| Artifact | SHA-256 |
|---|---|
| artifact snapshot | `ab97a28882afbcbb7bc7b7e78ec1f50b46494fe23c654ea15878cd1f44417a95` |
| exported Grok transcript | `cb5041363fd58c4cdeb606e5cdc454e2a3b85ab52b1c358846c52afed3f9d886` |
| normalized runtime gates | `bbaf5adbc21dc5b6e53692d49ab9d0742b4d3b80c29f51557d06181a0c377416` |
| local trace archive | `c9520c401a9e5f21f261f8527d4541a82b5450658b7ea6b1637e0ec83ad124b6` |

## Official Codex Attempt

Official Codex CLI `0.144.3` was started with an ephemeral configuration and a
temporary SSH-stdio Vermory MCP server. The run failed before the model could
perform the workspace task because the authenticated ChatGPT account had
reached its usage limit.

This failure is not counted as a successful client trajectory:

- no repository artifact was created;
- neither MCP tool completed;
- no delivery was recorded;
- no observation or proposed memory was created.

The retained event stream has SHA-256
`a46b9aa453d5dca3e48c50ca31fc5d4d72d4c68da23b81985f4f5064df366d86`.
The empty final-output file has SHA-256
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
A later Codex retry must use a fresh operation ID and must independently call
both MCP tools; the successful Grok control cannot substitute for it.

## Preserved Failure Ledger

Failures remain chronological and are not replaced by the later successful
artifacts:

1. The first G01 provider call passed `--verbatim` through both the wrapper and
   provider, and Grok rejected the duplicate flag. The runtime wrapper was split
   into an environment-only provider wrapper and a strict one-turn OpenClaw
   wrapper before retrying.
2. The first C02 validity command generated an invalid RFC3339 timestamp. The
   CLI rejected it before a database write; the retry used the PostgreSQL clock.
3. The first exact-boundary C02 replay exposed stale sibling assistant history.
   A RED regression reproduced it, the shared snapshot filter was fixed, and the
   full real turn was rerun.
4. A nonexistent `memory rebuild-projection` CLI command was attempted during
   W03 setup and rejected without changing authority. W19 lexical forget already
   updates authority and projection transactionally.
5. The official Codex attempt stopped at the external usage-limit gate and is
   retained as failed client evidence.
6. Two evidence-only helper attempts failed after the successful Grok run: one
   shell-quoted SQL gate command and one non-login-shell verification that could
   not resolve `go`. Neither touched product authority. The corrected SQL and
   absolute Go-path replay produced the accepted gates above.

## Deterministic Client Gates

The real-client evidence is complemented by deterministic acceptance:

```bash
VERMORY_TEST_DATABASE_URL='postgresql:///vermory_w19_test?host=/tmp' \
  go test -p 1 -count=1 \
  ./internal/runtime ./internal/webchat ./internal/mcpserver

pnpm -C integrations/openclaw test
pnpm -C integrations/openclaw typecheck
pnpm -C integrations/openclaw build
pnpm -C integrations/openclaw pack --dry-run
```

These tests prove the lifecycle and transport contracts. They do not convert
the failed official Codex attempt into a successful Codex trajectory.

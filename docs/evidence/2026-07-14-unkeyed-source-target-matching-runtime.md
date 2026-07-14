# Provider-Assisted Unkeyed Source Target Matching Runtime Evidence

Date: 2026-07-14

Tested implementation revision: `187f47d3fdfa7ab9dfc68181fbde79f56f23c493`

## Scope

This evidence exercises one trusted source fact that has an exact source
revision and exact semantic content but no Vermory `memory_key`. A real Grok
provider receives only the current tenant and confirmed workspace's closed set
of active keyed facts, selects one listed key or abstains, and cannot activate
memory. Vermory persists the provider evidence, creates the existing governed
source candidate only for a valid unique match, and requires an explicit
operator acceptance before normal AI context changes.

This is provider-assisted closed-set target matching. It is not arbitrary
document extraction, open-vocabulary conflict discovery, source authority
ranking, or automatic activation of model output.

## Frozen Scenario

Case `108-workspace-unkeyed-source-target-match` and runtime case `W06` freeze a
software release-control workflow:

| Key or role | Fact |
|---|---|
| `release.signing.mode` | Production releases use a macOS keychain certificate. |
| `deploy.api.timeout` | The deployment API timeout is 800 ms. |
| `release.attestation.format` | Production releases publish a signed SLSA provenance statement. |
| New unkeyed source fact | Production releases now use GitHub Actions OIDC keyless signing. |
| Other-tenant distractor | Production releases use static cloud credentials. |

The committed fixtures have these SHA-256 values:

```text
source.md   88762385b372deb16fd2c8695f6eb3452be6b3bc5093ff7a521133f6905a89e7
claims.json 9163e754a1e0df14ca29e971455c80414abf3f97f67900449fe5b85f487378c5
tasks.json  a6a7dc4e91ae6ec2cdea67f676f1bb3bf2bf4e22e5ece99944cbc7237afcb594
case.json   f4f832690bcf54859083720c825f82cbc9a0219d1562593a58ef9e8baf0f87a8
```

## Runtime

```text
Vermory version: 0.1.0-dev
Vermory revision: 187f47d3fdfa7ab9dfc68181fbde79f56f23c493
Vermory build date: 2026-07-14T15:16:31+08:00
Vermory binary SHA-256: 015448ee4adecd882a3fedef3e4e20009b4bb9c4864fcc9f6640689426dfc305
Go runtime: go1.26.5
Grok CLI: 0.2.101 (5bc4b5dfadcf)
Grok model: grok-4.5
PostgreSQL: 18.4
Schema version: 11
Dedicated database: vermory_w06_20260714071952
Target tenant: w06-local
Distractor tenant: w06-other
```

The Grok runs used a fresh `HOME` with copied login material and one configured
MCP server. Cross-session memory, web search, plan mode, and subagents were
disabled. `grok mcp doctor` reported one healthy server, protocol `2025-06-18`,
and exactly two tools: `prepare_context` and `commit_observation`.

## Real Provider Decisions

Three real `grok-4.5` matching operations ran concurrently against the same
candidate snapshot:

| Operation | Result | Selected key | Provider artifact SHA-256 |
|---|---|---|---|
| `w06-real-match` | matched | `release.signing.mode` | `38cd9b96e66f2b00cb85d3260848cae0908f5f456a1d8a0bcbf60693d6416662` |
| `w06-real-ambiguous` | abstained | none | `8f7773db6bee5899ab369c643afbd7b03c784fa63d08c9c11d0162979611c1b3` |
| `w06-real-unrelated` | abstained | none | `641534a8f699e8eb9828b91408a6f627d0e43db69dcca1cff6d9d7d0fcbd6eab` |

All three audit rows have candidate-set fingerprint
`ec05d49edb4bb2e92b2bfcd31c04c09a8507e7d00c14b1eb75c7a657407cd18f`.
The stored candidate JSON contains zero occurrences of the other tenant's
static credential fact.

The matched decision returned:

```text
source match: d9647442-f76d-4fe5-b185-99b8b0d7221a
selected key: release.signing.mode
target memory: 9c24d779-9294-4264-8044-ebc24554e43c
candidate memory: 52c0d7f6-82b0-48ef-9042-7a3e493b21bf
candidate status before review: proposed
```

The ambiguous source combined an identity-bound release flow with signed
release evidence, so Grok abstained rather than choosing either signing or
attestation. The unrelated maintenance-window source also abstained. Neither
operation created an observation or governed-memory candidate.

## Proposal Isolation And Acceptance

Before operator acceptance, real Grok session
`019f5f81-7638-7143-bf88-856088156fee` called `prepare_context`. PostgreSQL
measured these positions in the persisted delivery:

```text
delivery ID: a0f476bd-1713-44e0-949c-f61a4858a49c
macOS keychain certificate: 146
800 ms: 48
signed SLSA provenance statement: 86
GitHub Actions OIDC keyless signing: 0
static cloud credentials: 0
```

The operator then accepted candidate
`52c0d7f6-82b0-48ef-9042-7a3e493b21bf`. PostgreSQL atomically changed the
target to `superseded`, changed the candidate to `active`, preserved the timeout
and attestation facts as active, and left both abstained match decisions as
audit-only rows.

Projection rebuild retained four active documents across both tenants with
fingerprint `61e769d60beaa3ce142d35dd28b902bd` before and after rebuild. Proposed
client write-backs and the superseded keychain fact were excluded.

## Real MCP Coder Task

Real Grok session `019f5f83-d177-7192-ae20-f7b96ca2a05a` executed:

```text
prepare_context (w06-grok-final-prepare-2)
-> current OIDC signing + 800 ms timeout + SLSA attestation
-> create and deterministically verify release-control-policy.md
-> commit_observation (w06-grok-final-observation-2)
-> agent_result stored as proposed
```

The generated artifact is committed as a
[normalized snapshot](snapshots/2026-07-14-unkeyed-source-target-matching-grok-release-control-policy.md).

```text
delivery ID: bc859f6a-6086-416f-9dce-80490eab0533
observation ID: d750c97e-540c-4807-a37c-525726062118
write-back memory ID: af9b3858-10f6-4215-8dd2-85bfad20a5ae
write-back lifecycle: proposed
artifact SHA-256: 37175009a79c26ce1519d4fd1ddc0b60646582782a26105450233e199b6757d9
transcript SHA-256: 308a7638ae15f48784251922ee67e21e42459f3e199471d07e9903cf03645bac
```

Persisted delivery positions independently establish the consumed context:

```text
GitHub Actions OIDC keyless signing: 84
800 ms: 48
signed SLSA provenance statement: 151
macOS keychain certificate: 0
static cloud credentials: 0
```

## Stale Probes

Real Grok session `019f5f85-3411-7f00-95ac-87bb12acb66b` called
`prepare_context` twice without write-back or file changes.

| Probe | Delivery | OIDC position | Old keychain position | Other-tenant position |
|---|---|---:|---:|---:|
| Exact stale statement | `b023114b-92ff-4dc7-92f3-4a465167292d` | 46 | 0 | 0 |
| Certificate-backed paraphrase | `a8394336-c22e-4e36-bca0-11bb0ca7af41` | 46 | 0 | 0 |

The preserved stale transcript SHA-256 is
`ceacb1136cbfe02e7948f2be13a2969b77b42b4b0686ba931fcca8b0422bccfc`.

## Database And Isolation Evidence

The final `source_match_decisions` inventory contains one `matched` and two
`abstained` rows for `w06-local`. The table has RLS enabled and exactly one
tenant-isolation policy. Application tests also require the restricted runtime
role grant, tenant-aware foreign keys for continuity, target, observation, and
candidate links, and inclusion in the authoritative backup fingerprint.

Final target-tenant governed-memory counts were:

```text
active: 3
superseded: 1
proposed: 2
```

Both proposed rows are real Grok task write-backs. The first coder process
continued after the command wrapper returned early; the official artifact and
ledger values above use the second independently verified operation.

## Preserved Execution Corrections

1. Two pre-accept probes using `dontAsk` were cancelled at MCP authorization
   before a delivery existed. Re-running with an isolated one-server config and
   `--always-approve` produced the recorded read-only delivery.
2. The first final coder wrapper surfaced a telemetry export error and returned
   before the still-running Grok process completed. It later produced a valid
   delivery and proposed write-back. A second operation was run with OTEL export
   disabled and independently verified; both audit rows remain preserved.
3. The stale-probe shell wrapper assigned to zsh's read-only `status` variable
   after Grok had completed. The transcript and both PostgreSQL deliveries were
   already complete; the evidence was recovered without another model call.

## Deterministic Hard Gates

| Gate | Result |
|---|---|
| Provider sees only same-tenant, same-workspace active keyed facts | PASS |
| Clear unkeyed source selects exactly one existing key | PASS |
| Ambiguous source abstains | PASS |
| Unrelated source abstains | PASS |
| Match alone leaves current AI context unchanged | PASS |
| Proposed candidate has no search projection | PASS |
| Explicit acceptance changes current fact | PASS |
| Projection rebuild excludes non-active states | PASS |
| Real artifact contains all three current facts | PASS |
| Real artifact excludes stale and cross-tenant facts | PASS |
| Real MCP write-back remains proposed | PASS |
| Exact and paraphrased stale probes return OIDC only | PASS |
| Match audit table is RLS protected and authoritative | PASS |

## Claim Boundary

This slice proves one closed-set provider-assisted target match, two real
abstentions, durable audit, explicit candidate acceptance, projection rebuild,
same-scope isolation, stale suppression, and one real Grok MCP consumption and
write-back path. It does not prove arbitrary-document claim extraction,
open-ended semantic conflict detection, provider-generated facts, multi-source
authority ranking, conversation formation, hybrid retrieval, formation quality
at scale, sealed benchmark performance, or final release readiness.

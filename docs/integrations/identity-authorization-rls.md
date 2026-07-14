# Identity, Authorization, And PostgreSQL RLS

Vermory has two HTTP deployment profiles:

- `web-chat` is the existing local, server-configured tenant profile. It is loopback-only and keeps `--tenant-id` outside every request.
- `serve` is the authenticated multi-tenant profile. It derives tenant and role from a server-issued bearer token and connects through a non-owner PostgreSQL role protected by RLS.

`serve` never runs migrations, never accepts a tenant selector, and never falls back to admin credentials.

## Prerequisites

- PostgreSQL 16 or newer;
- one admin/migration connection that owns the Vermory schema;
- one separate `LOGIN` runtime role that is not a superuser, does not have `BYPASSRLS`, and owns no served table;
- the Vermory CLI built from this repository.

Keep the two connection strings separate:

```bash
export VERMORY_ADMIN_DATABASE_URL='postgresql:///vermory?host=/tmp'
export VERMORY_RUNTIME_DATABASE_URL='postgresql://vermory_runtime:<password>@/vermory?host=/tmp'
```

Do not pass either connection string to AI prompts, OpenClaw plugin config, model context, logs, or committed files.

## Migrate And Provision

Apply migrations only with the admin connection:

```bash
./bin/vermory database migrate \
  --database-url "$VERMORY_ADMIN_DATABASE_URL"
```

Create a dedicated PostgreSQL login role. Role creation remains an operator/database-administrator action:

```sql
CREATE ROLE vermory_runtime
  LOGIN
  PASSWORD '<runtime-password>'
  NOSUPERUSER
  NOBYPASSRLS;
```

Grant the runtime boundary through the CLI:

```bash
./bin/vermory database grant-runtime \
  --database-url "$VERMORY_ADMIN_DATABASE_URL" \
  --role vermory_runtime
```

The grant command:

- grants CRUD access only to the 12 tenant-bearing continuity tables;
- grants the observation sequence needed by conversation writes;
- grants `EXECUTE` on `vermory_auth.authenticate_token(text, bytea)`;
- does not grant direct reads of `vermory_auth.api_tokens`;
- removes direct access to legacy project/source/capsule/WCEF tables;
- rejects nonexistent, non-login, superuser, `BYPASSRLS`, table-owner, or inherited-forbidden-access roles.

At startup, `serve` revalidates the current database identity and required privileges before binding a listener.

## Issue And Revoke Tokens

Tokens use this external form:

```text
vmt_<public-id>_<secret>
```

PostgreSQL stores only the SHA-256 digest and lifecycle metadata. The raw token is printed only on the first successful issue operation:

```bash
./bin/vermory identity token issue \
  --database-url "$VERMORY_ADMIN_DATABASE_URL" \
  --operation-id issue-openclaw-client-1 \
  --tenant-id example-tenant \
  --subject-id openclaw-client \
  --role client \
  --expires-at 2026-08-14T00:00:00Z
```

Replaying the same issue operation returns metadata but not the secret. If the first secret is lost, issue a new token with a new operation ID.

Inspect metadata by public ID:

```bash
./bin/vermory identity token inspect \
  --database-url "$VERMORY_ADMIN_DATABASE_URL" \
  --public-id '<public-id>'
```

Revoke explicitly:

```bash
./bin/vermory identity token revoke \
  --database-url "$VERMORY_ADMIN_DATABASE_URL" \
  --operation-id revoke-openclaw-client-1 \
  --public-id '<public-id>'
```

Expired, revoked, unknown, malformed, or missing tokens receive `401`. Authentication database failures fail closed and do not fall back to anonymous access.

## Role Matrix

| Route family | `client` | `operator` | `owner` |
|---|---:|---:|---:|
| Chat turn | Allow | Allow | Allow |
| OpenClaw prepare/complete/fail | Allow | Allow | Allow |
| Conversation inspection | Deny | Allow | Allow |
| Confirm/correct/forget memory | Deny | Allow | Allow |
| Global Defaults | Deny | Allow | Allow |
| Bridge create/inspect/reverse | Deny | Allow | Allow |
| Token lifecycle | CLI/admin only | CLI/admin only | CLI/admin only |

No HTTP role can select another tenant. Unknown routes are denied rather than inheriting client access.

## Start The Authenticated Server

Loopback HTTP is accepted for local deployment:

```bash
./bin/vermory serve \
  --database-url "$VERMORY_RUNTIME_DATABASE_URL" \
  --listen 127.0.0.1:8788 \
  --provider external
```

Any non-loopback listener requires both certificate and key:

```bash
./bin/vermory serve \
  --database-url "$VERMORY_RUNTIME_DATABASE_URL" \
  --listen 0.0.0.0:8788 \
  --tls-cert /etc/vermory/tls.crt \
  --tls-key /etc/vermory/tls.key \
  --provider external
```

Provider configuration remains server-owned. `serve` supports the same external, Grok CLI, OpenAI-compatible, SiliconFlow, and Duojie provider constructors as local Web Chat.

## OpenClaw

Put the client token in the OpenClaw process environment:

```bash
export VERMORY_API_TOKEN='<issued-token>'
```

The official plugin reads this variable once during registration. It does not add a token field to OpenClaw config, logs, prompt text, or model context. A malformed value fails plugin registration without echoing the token. An absent value preserves local unauthenticated compatibility.

See [OpenClaw Runtime Integration](openclaw-runtime.md) for plugin installation, hook permissions, isolated Grok replay, governance operations, and failure behavior.

## Verify RLS

The served graph has tenant-aware composite foreign keys and RLS policies on:

```text
continuity_spaces
continuity_bindings
conversation_bindings
observations
governed_memories
memory_deliveries
memory_search_documents
conversation_turns
bridge_operations
bridge_events
bridge_memory_effects
conversation_links
source_match_decisions
```

Using the runtime role, a missing tenant setting sees zero rows:

```sql
SELECT set_config('vermory.tenant_id', '', false);
SELECT count(*) FROM continuity_spaces;
```

Set one tenant and intentionally omit the application filter:

```sql
SELECT set_config('vermory.tenant_id', 'example-tenant', false);
SELECT DISTINCT tenant_id FROM continuity_spaces;
```

Only `example-tenant` may appear. A row carrying tenant B and referencing a tenant A continuity, observation, delivery, memory, or bridge UUID is rejected by the composite foreign key even when the attacker knows the UUID.

Application queries keep explicit tenant predicates. RLS is the second barrier, not the only barrier.

## Backup And Restore

PostgreSQL is authoritative. Back up the complete database, including `vermory_auth`, continuity state, governed memory, delivery history, bridge audit, and goose migration history.

Token digests are authentication material even though raw secrets cannot be recovered from them. Encrypt backups, limit restore access, and treat an exposed backup as a reason to revoke and reissue affected tokens.

Create a native custom-format dump without copying source ownership or source-role ACLs:

```bash
pg_dump \
  --format=custom \
  --no-owner \
  --no-acl \
  --file=/secure/path/vermory.dump \
  "$VERMORY_ADMIN_DATABASE_URL"
```

PostgreSQL custom format is not encryption. Encrypt and access-control the dump during storage and transfer.

Restore into a newly created empty database with an admin identity:

```bash
createdb \
  --maintenance-db="$POSTGRES_ADMIN_MAINTENANCE_URL" \
  "$VERMORY_TARGET_DATABASE_NAME"

pg_restore \
  --exit-on-error \
  --no-owner \
  --no-acl \
  --dbname="$VERMORY_TARGET_ADMIN_DATABASE_URL" \
  /secure/path/vermory.dump
```

PostgreSQL roles and passwords are cluster-level objects and may require separate role/bootstrap automation. After restore:

1. apply any newer migrations with `database migrate` and the admin connection;
2. recreate the restricted runtime login if needed;
3. rerun `database grant-runtime`;
4. rebuild disposable search state with `database rebuild-projections`;
5. run startup role validation through `serve`;
6. verify missing-context and filter-omission RLS queries;
7. verify active/revoked token behavior before accepting traffic.

```bash
./bin/vermory database migrate \
  --database-url "$VERMORY_TARGET_ADMIN_DATABASE_URL"

./bin/vermory database grant-runtime \
  --database-url "$VERMORY_TARGET_ADMIN_DATABASE_URL" \
  --role vermory_runtime

./bin/vermory database rebuild-projections \
  --database-url "$VERMORY_TARGET_ADMIN_DATABASE_URL"
```

The projection rebuild runs in one transaction and inserts only active governed memories. Deleted and superseded content must remain absent from exact and related recall after rebuild. Source-match audit is authoritative rather than a search projection; forgetting a referenced memory redacts its fact text from source fields, candidate snapshots, provider output, and reason text while preserving structural decision metadata.

The reproducible local restore evidence is recorded in [PostgreSQL Operations And Recovery Evidence](../evidence/2026-07-14-postgresql-operations-recovery.md).

## Shutdown And Removal

Stopping or uninstalling a client integration does not delete governed memory or audit history.

For access removal:

1. revoke client and operator tokens;
2. stop `serve`;
3. remove the runtime role's grants with `DROP OWNED BY vermory_runtime` or an equivalent DBA-controlled operation;
4. drop the runtime role only after no process uses it;
5. retain, archive, or delete the Vermory database according to the operator's data-governance policy.

Deleting the database is separate from revoking runtime access and must never be an implicit uninstall action.

# CI Release Gates Evidence

Date: 2026-07-14

Implementation revision: `dc78bd68748374dda313d436b4e9d739bc0fed47`

## Scope

This evidence upgrades the pull-request workflow from a fast source-only Go
check to automatic gates for the runtime surfaces already claimed by the
repository. It does not create a public release or claim final release
readiness.

Before this change, GitHub Actions did not set
`VERMORY_TEST_DATABASE_URL`. PostgreSQL-dependent workspace, conversation,
identity, RLS, recovery, MCP, and operator tests therefore followed their
documented skip path. The workflow also did not install or test the OpenClaw
package and did not build the release binary.

## Enforced Pull-Request Gates

The single `test` job now uses:

```text
Ubuntu GitHub-hosted runner
PostgreSQL service image: postgres:18
Database: vermory_test
Go version: go.mod (1.25.7)
Node.js: 24
pnpm: 11.12.0
Job timeout: 30 minutes
```

| Gate | Command or mechanism |
|---|---|
| PostgreSQL readiness | Container health check with `pg_isready` |
| Full Go suite with database | `go test -p 1 -count=1 ./...` |
| Runtime race coverage | `go test -race -p 1 -count=1` across authn, runtime, webchat, identity CLI, operator CLI, MCP server, command, and provider packages |
| Reality race coverage | `go test -count=1 -race ./internal/reality` |
| Static analysis | `go vet ./...` |
| Dependency drift | `go mod tidy` followed by zero `go.mod` / `go.sum` diff |
| Release build | `go build -trimpath -o /tmp/vermory ./cmd/vermory` |
| OpenClaw dependency integrity | `pnpm -C integrations/openclaw install --frozen-lockfile` |
| OpenClaw behavior and build | `pnpm -C integrations/openclaw check` |
| OpenClaw publish shape | `pnpm -C integrations/openclaw pack --dry-run` |
| Patch hygiene | `git diff --check` |

Package-level Go execution is intentionally serial because the integration
tests share one dedicated PostgreSQL database. This avoids turning database
test races into nondeterministic CI noise while preserving goroutine race
detection inside each tested package.

## Local Verification

The workflow syntax passed `actionlint 1.7.7`. The same full database suite,
runtime race package set, OpenClaw 43-test check, typecheck, build, package
dry-run, `go vet`, tidy check, release build, and diff check passed locally
before push.

## Remote Verification

GitHub Actions run
[`29299572273`](https://github.com/jstar0/Vermory/actions/runs/29299572273)
executed the workflow from revision `dc78bd6`.

```text
job: test
conclusion: success
duration: 2m35s
main steps completed: 16/16
```

The remote job reported success for container initialization, PostgreSQL-backed
tests, runtime race tests, reality race tests, vet, module verification,
release build, OpenClaw install/check/package, and clean-diff verification.
This is stronger evidence than the previous CI result because the database URL
was present and the PostgreSQL service was healthy before tests began.

## Main Branch Enforcement

The public repository previously had no branch protection. After the clean
remote run, `main` was configured with this minimal single-maintainer policy:

```json
{
  "required_check": "test",
  "strict": true,
  "pull_request_required": true,
  "required_approving_reviews": 0,
  "required_conversation_resolution": true,
  "enforce_admins": false,
  "allow_force_pushes": false,
  "allow_deletions": false
}
```

The policy requires a branch to be current with `main` and the expanded `test`
job to pass before a normal merge. It does not require a second maintainer's
approval and does not prevent repository administrators from emergency
recovery. After protection was enabled, Draft PR 1 reported `CLEAN` and
`MERGEABLE` with the required `test` check completed successfully.

## Claim Boundary

This result proves that the current PR automatically executes the repository's
database-backed and OpenClaw release gates on a clean Ubuntu runner and that
normal `main` integration is protected by that check. It does not prove
production scale, a published GitHub Release, artifact signing, container
deployment, macOS/Windows portability, external sealed evaluation, or final
open-source release acceptance.

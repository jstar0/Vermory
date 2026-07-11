# Vermory Experiment 0 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Adopt the Vermory product identity and create a reality-first evidence pipeline that freezes, validates, and reports the first public workspace, conversation, global-default, and safety trajectories without freezing the final memory architecture.

**Architecture:** Keep the existing Go modular monolith and add an `internal/reality` package concerned only with evaluation evidence, not production memory semantics. Public cases live under `reality/cases`; raw authorized sources remain outside the repository, and committed fixtures contain only selected, anonymized excerpts plus hashes. A local case may be `public` or `withheld_local`; `sealed` evidence can enter only as an external evaluator attestation, preventing a readable local directory from being mislabeled as blind.

**Tech Stack:** Go 1.25.7, Cobra, standard-library JSON/JSONL/SHA-256, existing local artifact store, PostgreSQL-independent tests.

## Global Constraints

- Product name: `Vermory`.
- Product slug and CLI name: `vermory`.
- Tagline: `Governed Memory for AI`.
- Keep the repository path unchanged during Experiment 0.
- Keep historical case IDs, historical artifact paths, and previous report contents unchanged unless they are active product output.
- PostgreSQL remains the native authoritative-store decision, but Experiment 0 adds no final memory tables.
- Do not add Redis, Neo4j, Qdrant, Elasticsearch, Kafka, or a new default service.
- Do not call a locally readable fixture `sealed`.
- Do not copy credentials, raw private transcripts, or unredacted private paths into committed fixtures.
- Every manual code edit uses `apply_patch`; mechanical import rewrites and `gofmt` may use tooling.
- Existing unrelated `.DS_Store`, `bluebridge-report/`, and `docs/real-case-showcase.md` assets remain untouched.

---

## Task 0: Preserve The Verified Backend Bake-Off As A Separate Commit

**Files:**
- Modify: `.gitignore`
- Modify: `cmd/contextmesh/main.go`
- Create: `internal/memorybackend/*.go`
- Create: `backend-casebook/bakeoff.json`
- Create: `backend-casebook/scenarios.json`
- Create: `backend-casebook/results/linux-arm64-20260711.json`
- Create: `docs/backend-bakeoff.md`
- Create: `docs/backend-bakeoff-results.md`
- Exclude: `.DS_Store`, `bluebridge-report/`, `docs/real-case-showcase.md`, and all new Vermory design files

**Interfaces:**
- Preserves: the already verified native, mem0, MemOS, and Supermemory adapter contract and evidence.
- Produces: a clean review boundary before the CLI and module rename.

- [x] **Step 1: Re-run the repository and backend-package verification**

```bash
go test ./...
go test -race ./internal/memorybackend
git diff --check
jq empty backend-casebook/bakeoff.json backend-casebook/scenarios.json backend-casebook/results/linux-arm64-20260711.json
```

Expected: all commands exit 0.

- [x] **Step 2: Inspect the exact backend change set**

```bash
git diff -- .gitignore cmd/contextmesh/main.go
find internal/memorybackend backend-casebook -type f -print | sort
```

Verify no credential file, `.supermemory/` runtime file, generated artifact directory, or unrelated report appears.

- [x] **Step 3: Stage exact backend paths**

```bash
git add .gitignore cmd/contextmesh/main.go internal/memorybackend backend-casebook docs/backend-bakeoff.md docs/backend-bakeoff-results.md
git diff --cached --check
git diff --cached --name-only
```

Expected staged paths are limited to the files listed by this task.

- [x] **Step 4: Commit the backend evidence boundary**

```bash
git commit -m "feat: add evidence-backed memory backend adapters"
```

---

## Task 1: Adopt The Vermory Identity Without Rewriting History

**Files:**
- Create: `docs/adr/0001-vermory-product-name.md`
- Create: `internal/brand/brand.go`
- Create: `internal/brand/brand_test.go`
- Move: `cmd/contextmesh/main.go` -> `cmd/vermory/main.go`
- Modify: `go.mod`
- Modify: all Go imports beginning with `contextmesh/`
- Modify: active Go output strings that identify the current product
- Move: the five `2026-07-11-contextmesh-*.md` design files to equivalent `2026-07-11-vermory-*.md` names
- Modify: links and headings inside the moved design files
- Preserve: `casebook/cases/001-contextmesh-bluebridge-preparation/`
- Preserve: existing artifact and historical report contents

**Interfaces:**
- Produces: `brand.Name`, `brand.Slug`, and `brand.Tagline` constants.
- Produces: the `vermory` CLI entry point at `./cmd/vermory`.
- Preserves: historical `contextmesh` fixture IDs as immutable evidence identifiers.

- [x] **Step 1: Write the brand contract test**

Create `internal/brand/brand_test.go`:

```go
package brand

import "testing"

func TestIdentity(t *testing.T) {
	if Name != "Vermory" {
		t.Fatalf("expected product name Vermory, got %q", Name)
	}
	if Slug != "vermory" {
		t.Fatalf("expected product slug vermory, got %q", Slug)
	}
	if Tagline != "Governed Memory for AI" {
		t.Fatalf("unexpected tagline %q", Tagline)
	}
}
```

- [x] **Step 2: Run the brand test and verify the missing-symbol failure**

Run:

```bash
go test ./internal/brand
```

Expected: FAIL because `internal/brand` or the constants do not exist.

- [x] **Step 3: Add the brand constants**

Create `internal/brand/brand.go`:

```go
package brand

const (
	Name    = "Vermory"
	Slug    = "vermory"
	Tagline = "Governed Memory for AI"
)
```

- [x] **Step 4: Rename the module and CLI mechanically**

Run:

```bash
git mv cmd/contextmesh cmd/vermory
go mod edit -module=vermory
rg -l '"contextmesh/' --glob '*.go' | xargs perl -pi -e 's|"contextmesh/|"vermory/|g'
gofmt -w cmd/vermory internal
```

Modify `cmd/vermory/main.go` so the Cobra root command uses `brand.Slug`, and current product-facing output uses `brand.Name`. Do not replace historical fixture IDs or artifact paths merely because they contain `contextmesh`.

- [x] **Step 5: Move the approved design package and update links**

Move the five approved design files to:

```text
docs/superpowers/specs/2026-07-11-vermory-reality-first-memory-platform-design.md
docs/superpowers/specs/2026-07-11-vermory-product-constitution.md
docs/superpowers/specs/2026-07-11-vermory-reality-program.md
docs/superpowers/specs/2026-07-11-vermory-hypothesis-register.md
docs/superpowers/specs/2026-07-11-vermory-vertical-experiment-plan.md
```

Update their titles, relative links, and present-tense product references to `Vermory`. Keep explicit historical references to `ContextMesh` when describing old evidence or migration.

- [x] **Step 6: Record the naming decision and compatibility boundary**

Create `docs/adr/0001-vermory-product-name.md` with:

```markdown
# ADR 0001: Adopt Vermory As The Product Name

Status: accepted

## Decision

The current product name is Vermory. The CLI and Go module use `vermory`.

## Meaning

Vermory combines verifiable or versioned truth with memory. The product tagline is "Governed Memory for AI".

## Compatibility Boundary

Historical ContextMesh case IDs, run IDs, artifact paths, and report contents remain immutable evidence. Current product surfaces, code imports, executable names, and active design documents use Vermory. Database identifiers are migrated only through an explicit schema hypothesis and migration, not a textual rename.
```

- [x] **Step 7: Verify the identity migration**

Run:

```bash
go test ./...
go run ./cmd/vermory --help
rg -n 'Use:[[:space:]]+"contextmesh"|module contextmesh|"contextmesh/internal' go.mod cmd internal
```

Expected:

- all Go tests pass;
- help output starts with `vermory`;
- the final `rg` command returns no matches.

- [x] **Step 8: Commit only the identity change**

Stage only the files named in this task, excluding backend bake-off files and unrelated assets.

```bash
git add -u go.mod cmd internal docs/superpowers/specs
git add internal/brand docs/adr/0001-vermory-product-name.md docs/superpowers/specs/2026-07-11-vermory-*.md
git commit -m "refactor: adopt Vermory product identity"
```

Before committing, verify `git diff --cached --name-only` contains no `bluebridge-report/`, `.DS_Store`, or user-owned showcase files.

---

## Task 2: Define A Reality Evidence Contract

**Files:**
- Create: `internal/reality/types.go`
- Create: `internal/reality/validate.go`
- Create: `internal/reality/validate_test.go`
- Create: `reality/schema/case.schema.json`
- Create: `reality/testdata/valid-public/manifest.json`
- Create: `reality/testdata/valid-public/events.jsonl`
- Create: `reality/testdata/invalid-local-sealed/manifest.json`
- Create: `reality/testdata/invalid-local-sealed/events.jsonl`

**Interfaces:**
- Produces: `reality.LoadCase(dir string) (Case, error)`.
- Produces: `reality.ValidateCase(c Case) []Violation`.
- Produces: `EvidencePublic` and `EvidenceWithheldLocal` for repository-readable cases.
- Excludes: a local `EvidenceSealed` case type; sealed results are external attestations in Task 6.

- [x] **Step 1: Write failing validation tests**

Create tests covering:

```go
func TestLoadAndValidatePublicCase(t *testing.T) {
	c, err := LoadCase("../../reality/testdata/valid-public")
	if err != nil {
		t.Fatal(err)
	}
	if got := ValidateCase(c); len(got) != 0 {
		t.Fatalf("expected valid case, got violations: %#v", got)
	}
}

func TestRejectsLocalCaseClaimingSealedEvidence(t *testing.T) {
	_, err := LoadCase("../../reality/testdata/invalid-local-sealed")
	if err == nil || !strings.Contains(err.Error(), "sealed evidence cannot be loaded from a readable local case") {
		t.Fatalf("expected sealed-evidence rejection, got %v", err)
	}
}

func TestRequiresExpectedAndForbiddenBehavior(t *testing.T) {
	c := validCase()
	c.Expectations.ForbiddenFacts = nil
	violations := ValidateCase(c)
	assertViolationCode(t, violations, "forbidden_facts_required")
}
```

- [x] **Step 2: Run the tests and verify failure**

Run:

```bash
go test ./internal/reality -run 'TestLoadAndValidate|TestRejectsLocal|TestRequires' -v
```

Expected: FAIL because the package and functions do not exist.

- [x] **Step 3: Implement the evidence types**

Define these public shapes in `internal/reality/types.go`:

```go
type EvidenceLevel string

const (
	EvidencePublic        EvidenceLevel = "public"
	EvidenceWithheldLocal EvidenceLevel = "withheld_local"
)

type ContinuityLine string

const (
	LineWorkspace      ContinuityLine = "workspace"
	LineConversation   ContinuityLine = "conversation"
	LineGlobalDefaults ContinuityLine = "global_defaults"
	LineBridge         ContinuityLine = "bridge"
	LineSecurity       ContinuityLine = "security"
)

type SourceRef struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	FixturePath     string `json:"fixture_path"`
	OriginalRef     string `json:"original_ref,omitempty"`
	OriginalRev     string `json:"original_revision,omitempty"`
	SHA256          string `json:"sha256"`
	Authorized      bool   `json:"authorized"`
	Anonymization   string `json:"anonymization"`
}

type Anchor struct {
	Kind       string `json:"kind"`
	Value      string `json:"value"`
	Ambiguous  bool   `json:"ambiguous"`
}

type Expectations struct {
	CurrentFacts    []string `json:"current_facts"`
	ForbiddenFacts  []string `json:"forbidden_facts"`
	AllowedUnknowns []string `json:"allowed_unknowns,omitempty"`
	ExpectedAction  string   `json:"expected_action"`
}

type DownstreamTask struct {
	Prompt             string   `json:"prompt"`
	ArtifactChecks     []string `json:"artifact_checks,omitempty"`
	DeterministicChecks []string `json:"deterministic_checks"`
}

type Manifest struct {
	Version          int              `json:"version"`
	ID               string           `json:"id"`
	Title            string           `json:"title"`
	EvidenceLevel    EvidenceLevel    `json:"evidence_level"`
	ContinuityLines  []ContinuityLine `json:"continuity_lines"`
	Pressures        []string         `json:"pressures"`
	Sources          []SourceRef      `json:"sources"`
	Anchors          []Anchor         `json:"anchors"`
	Expectations     Expectations     `json:"expectations"`
	Task             DownstreamTask   `json:"task"`
}

type Event struct {
	ID        string         `json:"id"`
	Sequence  int            `json:"sequence"`
	Actor     string         `json:"actor"`
	Channel   string         `json:"channel"`
	SourceID  string         `json:"source_id"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type Case struct {
	Directory string
	Manifest  Manifest
	Events    []Event
}

type Violation struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
```

- [x] **Step 4: Implement strict loading and validation**

`LoadCase` must:

- decode `manifest.json` with `json.Decoder.DisallowUnknownFields()`;
- reject `evidence_level: "sealed"` with the exact tested error;
- decode non-empty JSONL from `events.jsonl`;
- call `ValidateCase` and return all validation codes in a deterministic error;
- never read `OriginalRef` while loading a committed case.

`ValidateCase` must enforce:

- version is `1`;
- ID, title, lines, pressures, sources, anchors, task prompt, deterministic checks, current facts, forbidden facts, and expected action are non-empty;
- source IDs and event IDs are unique;
- every event references a declared source;
- event sequence starts at 1 and is strictly contiguous;
- every fixture path is relative, clean, remains below the case directory, and exists;
- every source is authorized and has an anonymization statement;
- declared SHA-256 equals the fixture file's actual SHA-256;
- current and forbidden facts do not contain exact duplicates.

- [x] **Step 5: Add a machine-readable JSON Schema**

Create `reality/schema/case.schema.json` with draft 2020-12, `additionalProperties: false`, the same required manifest fields, and enums for evidence levels and continuity lines. The Go loader remains authoritative for cross-file and hash checks.

- [x] **Step 6: Run focused and full tests**

Run:

```bash
go test ./internal/reality -v
go test ./...
```

Expected: PASS.

- [x] **Step 7: Commit the evidence contract**

```bash
git add internal/reality reality/schema reality/testdata
git commit -m "feat: define reality evidence contract"
```

---

## Task 3: Add Freeze And Validation Commands

**Files:**
- Create: `internal/reality/freeze.go`
- Create: `internal/reality/freeze_test.go`
- Create: `internal/reality/report.go`
- Create: `internal/reality/report_test.go`
- Modify: `cmd/vermory/main.go`

**Interfaces:**
- Produces: `reality.FreezeCase(dir string) (FreezeReport, error)`.
- Produces: `reality.ValidateRoot(root string) ValidationReport`.
- Produces CLI: `vermory reality-freeze --case-dir <dir>`.
- Produces CLI: `vermory reality-validate --case-root <root> --artifact-root <root> --run-id <id>`.

- [x] **Step 1: Write failing freeze tests**

Cover deterministic hashing and mutation detection:

```go
func TestFreezeCaseWritesDeterministicLock(t *testing.T) {
	dir := copyFixture(t, "../../reality/testdata/valid-public")
	first, err := FreezeCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := FreezeCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	if first.LockSHA256 != second.LockSHA256 {
		t.Fatalf("freeze must be deterministic: %s != %s", first.LockSHA256, second.LockSHA256)
	}
}

func TestValidateRootDetectsPostFreezeMutation(t *testing.T) {
	dir := copyFixture(t, "../../reality/testdata/valid-public")
	if _, err := FreezeCase(dir); err != nil {
		t.Fatal(err)
	}
	appendFixture(t, filepath.Join(dir, "fixtures", "source.md"), "mutated")
	report := ValidateRoot(filepath.Dir(dir))
	if report.Pass || !hasViolation(report, "fixture_lock_mismatch") {
		t.Fatalf("expected mutation failure: %#v", report)
	}
}
```

- [x] **Step 2: Run focused tests and verify failure**

```bash
go test ./internal/reality -run 'TestFreezeCase|TestValidateRoot' -v
```

Expected: FAIL with undefined freeze/report functions.

- [x] **Step 3: Implement `fixture-lock.json`**

The lock format is:

```go
type FixtureLock struct {
	Version      int          `json:"version"`
	CaseID       string       `json:"case_id"`
	ManifestHash string       `json:"manifest_sha256"`
	EventsHash   string       `json:"events_sha256"`
	Files        []LockedFile `json:"files"`
}

type LockedFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}
```

Sort `Files` by slash-normalized relative path. Serialize with two-space indentation and a final newline. `FreezeCase` writes atomically and returns the SHA-256 of the serialized lock.

- [x] **Step 4: Implement root validation reports**

`ValidationReport` contains:

```go
type ValidationReport struct {
	RunID       string            `json:"run_id"`
	CaseRoot    string            `json:"case_root"`
	Pass        bool              `json:"pass"`
	Cases       int               `json:"cases"`
	ByLine      map[string]int    `json:"by_line"`
	ByEvidence  map[string]int    `json:"by_evidence"`
	Results     []CaseResult      `json:"results"`
}
```

Results are sorted by case ID. A missing line is reported as coverage information, not automatically a failed arbitrary count. Case validation or lock mismatch fails the report.

- [x] **Step 5: Wire the two CLI commands**

Add Cobra commands with exact flags:

```text
vermory reality-freeze --case-dir reality/cases/W01-example
vermory reality-validate --case-root reality/cases --artifact-root ./artifacts --run-id experiment-0-public
```

Validation writes:

```text
artifacts/reality-validation/<run-id>/report.json
artifacts/reality-validation/<run-id>/report.md
```

The command exits non-zero when `report.Pass` is false.

- [x] **Step 6: Verify CLI behavior**

```bash
go test ./internal/reality ./cmd/vermory
go run ./cmd/vermory reality-freeze --case-dir reality/testdata/valid-public
go run ./cmd/vermory reality-validate --case-root reality/testdata --artifact-root ./artifacts --run-id experiment-0-contract-smoke
```

Expected: freeze succeeds; validation returns non-zero because the test root intentionally contains `invalid-local-sealed`, and the report names that exact case.

- [x] **Step 7: Commit freeze and validation tooling**

```bash
git add internal/reality cmd/vermory/main.go
git commit -m "feat: freeze and validate reality cases"
```

---

## Task 4: Freeze The First Public Three-Line Cases

**Files:**
- Create: `reality/cases/W01-synapseloom-continuity/manifest.json`
- Create: `reality/cases/W01-synapseloom-continuity/events.jsonl`
- Create: `reality/cases/W01-synapseloom-continuity/fixtures/`
- Create: `reality/cases/C01-device-maintenance-continuity/manifest.json`
- Create: `reality/cases/C01-device-maintenance-continuity/events.jsonl`
- Create: `reality/cases/C01-device-maintenance-continuity/fixtures/`
- Create: `reality/cases/G01-language-default-local-override/manifest.json`
- Create: `reality/cases/G01-language-default-local-override/events.jsonl`
- Create: `reality/cases/G01-language-default-local-override/fixtures/`
- Create: `reality/cases/S01-deletion-and-source-injection/manifest.json`
- Create: `reality/cases/S01-deletion-and-source-injection/events.jsonl`
- Create: `reality/cases/S01-deletion-and-source-injection/fixtures/`
- Create: `reality/cases/README.md`

**Interfaces:**
- Consumes: Task 2 manifest/events contract and Task 3 freeze command.
- Produces: one workspace, one conversation, one paired global-default, and one safety trajectory suitable for Experiment 1 implementation.
- Does not produce: a sealed case or a final memory schema.

- [x] **Step 1: Extract and anonymize the SynapseLoom workspace trajectory**

Use these authorized local source aliases:

```text
authorized-workspace:synapseloom
authorized-session:synapseloom-2026-03-31
```

Select only the spans needed to demonstrate parallel plans, changed decisions, repository grounding, and cross-session continuation. Replace personal, credential, network, and unrelated project values with stable fixture aliases. Record the repository commit used as `original_revision`; do not commit the raw rollout.

- [x] **Step 2: Extract and anonymize an everyday conversation trajectory**

Use:

```text
authorized-session:device-maintenance-2026-05-14
```

Build a multi-event device-maintenance matter with a diagnostic correction, an explicit user exclusion, a narrowly authorized cleanup action, one failed attempt, verified final state, forbidden stale values, and a downstream continuation task. Do not include unrelated thread content or raw personal identifiers.

- [x] **Step 3: Build the global-default paired trajectory**

Use the managed global rule source for the stable Chinese-language preference and a real task-scoped language override as the negative case. The expected behavior is that the explicit local override applies only to its task and does not rewrite the global default.

The fixture records semantic excerpts and hashes, not the entire global configuration file.

- [x] **Step 4: Build the safety trajectory**

Use synthetic secrets and synthetic source injection only. The trajectory must include:

- a source passage attempting to instruct the system to ignore continuity policy;
- one target fact eligible for memory before deletion;
- an explicit deletion event;
- exact, paraphrased, and related-topic queries;
- independently valid related information that may still be returned;
- forbidden behavior defined as leakage of the deleted target fact, not an empty result set.

- [x] **Step 5: Scan committed fixtures for accidental sensitive data**

Run deterministic scans for:

```text
sk-
ghp_
github_pat_
postgres://
password=
Authorization:
Bearer
/Users/local-user
10.0.0.
```

The only allowed matches are clearly marked synthetic tokens inside `S01`. The validator report records any exception explicitly.

- [x] **Step 6: Freeze and validate every case**

```bash
for d in reality/cases/*; do
  test -d "$d" || continue
  go run ./cmd/vermory reality-freeze --case-dir "$d"
done
go run ./cmd/vermory reality-validate --case-root reality/cases --artifact-root ./artifacts --run-id experiment-0-public-v1
```

Expected: four cases pass; the report shows workspace, conversation, global-default, and security coverage without claiming the full discovery target is complete.

- [x] **Step 7: Commit only anonymized public evidence**

```bash
git add reality/cases
git commit -m "test: add first reality evidence batch"
```

Inspect the staged diff for secrets and raw local paths before committing.

---

## Task 5: Add External Attestation Without Fake Sealing

**Files:**
- Create: `internal/reality/attestation.go`
- Create: `internal/reality/attestation_test.go`
- Create: `reality/schema/attestation.schema.json`
- Modify: `cmd/vermory/main.go`
- Modify: `docs/superpowers/specs/2026-07-11-vermory-reality-program.md`

**Interfaces:**
- Produces: `reality.VerifyAttestation(data []byte, publicKey ed25519.PublicKey) (Attestation, error)`.
- Produces CLI: `vermory reality-attestation-verify --input <json> --public-key <base64>`.
- Does not produce: a local command that generates a sealed attestation.

- [x] **Step 1: Write failing attestation tests**

Test:

- valid Ed25519 signature passes;
- modified score payload fails;
- missing evaluator ID, suite version, implementation digest, hard-gate status, or run timestamp fails;
- a local case manifest cannot be converted into a sealed attestation by the CLI.

- [x] **Step 2: Implement the attestation shape**

```go
type Attestation struct {
	Version              int            `json:"version"`
	EvaluatorID          string         `json:"evaluator_id"`
	SuiteVersion         string         `json:"suite_version"`
	ImplementationDigest string         `json:"implementation_digest"`
	RunAt                time.Time      `json:"run_at"`
	HardGatesPass        bool           `json:"hard_gates_pass"`
	Counts               map[string]int `json:"counts"`
	FailureCategories    []string       `json:"failure_categories,omitempty"`
	Signature            string         `json:"signature"`
}
```

Canonicalize the unsigned payload through a private struct with fixed field order, then verify Ed25519. Reject unknown JSON fields and future versions.

- [x] **Step 3: Add only verification CLI behavior**

The command verifies an attestation received from an external evaluator and prints its evaluator, suite, implementation digest, gate result, and counts. It never signs and never labels repository-readable cases sealed.

- [x] **Step 4: Verify and document the boundary**

```bash
go test ./internal/reality -run Attestation -v
go test ./...
```

Update the Reality Program to name the attestation schema and state that Experiment 0 may complete with sealed infrastructure unavailable, but must report that limitation.

- [x] **Step 5: Commit attestation verification**

```bash
git add internal/reality reality/schema/attestation.schema.json cmd/vermory/main.go docs/superpowers/specs/2026-07-11-vermory-reality-program.md
git commit -m "feat: verify external sealed attestations"
```

---

## Task 6: Generate The Experiment 0 Readout

**Files:**
- Create: `internal/reality/experiment0.go`
- Create: `internal/reality/experiment0_test.go`
- Modify: `cmd/vermory/main.go`
- Create: `docs/experiment-0-readout.md`
- Modify: `docs/superpowers/specs/2026-07-11-vermory-hypothesis-register.md`

**Interfaces:**
- Produces CLI: `vermory experiment-0 --case-root reality/cases --artifact-root ./artifacts --run-id <id>`.
- Produces: JSON and Markdown reports under `artifacts/experiment-0/<run-id>/`.
- Produces: measured evidence only; does not declare schema version 1.

- [x] **Step 1: Write failing Experiment 0 report tests**

Test that the report:

- lists every case and fixture-lock hash;
- reports continuity and pressure coverage;
- distinguishes `public`, `withheld_local`, and externally attested sealed evidence;
- reports sealed infrastructure as `unavailable` when no attestation is supplied;
- carries forward every validation failure;
- never converts target seed counts into automatic pass/fail;
- names legacy scenario packs as `inspired_case` or `translated_proxy`, not real trajectories.

- [x] **Step 2: Implement the readout**

The report shape includes:

```go
type Experiment0Report struct {
	RunID              string                    `json:"run_id"`
	Pass               bool                      `json:"pass"`
	PublicValidation   ValidationReport          `json:"public_validation"`
	ContinuityCoverage map[string][]string        `json:"continuity_coverage"`
	PressureCoverage   map[string][]string        `json:"pressure_coverage"`
	EvidenceLevels     map[string]int             `json:"evidence_levels"`
	SealedStatus       string                     `json:"sealed_status"`
	Baselines          map[string]string          `json:"baselines"`
	HypothesisSignals  map[string][]string        `json:"hypothesis_signals"`
	Limitations        []string                   `json:"limitations"`
}
```

Experiment 0 passes when public cases are valid and frozen and no constitutional hard-gate definition is missing for the next experiment. Missing sealed infrastructure and incomplete target coverage remain explicit limitations, not hidden failures or fake passes.

- [x] **Step 3: Run the complete readout**

```bash
go run ./cmd/vermory experiment-0 --case-root reality/cases --artifact-root ./artifacts --run-id experiment-0-v1
jq . artifacts/experiment-0/experiment-0-v1/report.json
```

Expected:

- public validation passes;
- four initial cases are listed;
- sealed status is `unavailable` unless a real external attestation was supplied;
- target discovery coverage is reported as incomplete without making false claims;
- old round-1 scenario definitions are not counted as real executed trajectories.

- [x] **Step 4: Update the hypothesis register from evidence**

Only change a hypothesis from `proposed` to `testing` when at least one frozen case actively discriminates it. Record case IDs and Experiment 0 artifact URIs. Do not mark hypotheses `supported` based only on case creation.

- [x] **Step 5: Write the human readout**

`docs/experiment-0-readout.md` summarizes:

- the four frozen public cases and their real/synthetic provenance;
- evidence that exists versus evidence still missing;
- baseline runs available from existing harnesses;
- why old self-case and backend results remain supporting evidence only;
- exact hypotheses entering Experiment 1;
- explicit blockers for a genuine sealed evaluator or unavailable real client.

- [x] **Step 6: Run final verification**

```bash
go test ./...
go test -race ./internal/reality
go vet ./...
git diff --check
```

Expected: all commands exit 0.

- [x] **Step 7: Commit the Experiment 0 readout**

```bash
git add internal/reality cmd/vermory/main.go docs/experiment-0-readout.md docs/superpowers/specs/2026-07-11-vermory-hypothesis-register.md
git commit -m "docs: complete Vermory experiment 0 readout"
```

---

## Execution Notes

- Execute inline in this session because the user previously requested fewer subagents and asked the primary agent to own the work.
- Review `git status --short` before every commit and stage exact paths only.
- The repository currently has no Git remote. Do not invent or publish a remote during Experiment 0.
- Existing uncommitted backend bake-off work is verified but separate. Preserve it and either commit it in an isolated evidence commit before Task 1 or leave it unstaged; never mix it with brand or Experiment 0 commits.
- Local aliases or symlinks may point to the repository; generated evidence and path checks use the canonical Git root resolved at runtime.

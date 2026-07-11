# Contributing to Vermory

Vermory accepts code, documentation, evaluation cases, and reproducible failure reports.

## Before Opening a Pull Request

Run:

```bash
go test ./...
go test -race ./internal/reality
go vet ./...
```

Keep changes narrow and explain which real failure, frozen case, or falsifiable hypothesis the change addresses.

## Reality Evidence Rules

- Never commit credentials, private raw transcripts, personal absolute paths, or unrelated user content.
- Minimize and anonymize authorized source excerpts before committing them.
- A readable repository fixture may be `public` or `withheld_local`; it may not be described as sealed.
- Freeze expectations and forbidden behavior before implementing a mechanism intended to pass the case.
- Preserve failed cases and baseline outputs. Do not delete evidence merely to improve a score.
- Synthetic data is appropriate for security, privacy, mutation, and load testing, but it must be identified as synthetic.

## Architecture Changes

Before adding a permanent entity, service, lifecycle state, provider dependency, or release metric:

1. Map it to an existing hypothesis in the hypothesis register, or add a new falsifiable hypothesis.
2. Name the real case or operational failure that requires it.
3. Define how a simpler baseline will be compared.
4. Define rollback or migration behavior.

## Commit Style

Use concise imperative commit subjects, for example:

```text
feat: add conversation continuity candidate formation
test: freeze workspace rebind trajectory
docs: record retrieval ablation result
```

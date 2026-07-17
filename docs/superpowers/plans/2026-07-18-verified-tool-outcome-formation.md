# Verified Tool Outcome Formation Implementation Plan

Design: [Verified Tool Outcome Formation Design](../specs/2026-07-18-verified-tool-outcome-formation-design.md)

## Task 1: Freeze Reality Case

- [x] Add and freeze `F03-verified-tool-outcome-formation` before implementation.
- [x] Validate all public cases and update the authoritative coverage counts.

## Task 2: Tool Observation Authority

- [ ] Add the `tool_result` observation kind and tenant-scoped metadata migration.
- [ ] Implement exact turn, run, session, tool, call-ID, content, size, and replay validation.
- [ ] Reject sensitive result content before persistence and keep reset/deletion complete.

## Task 3: OpenClaw Capture

- [ ] Add an explicit tool allowlist and bounded documented result extractors.
- [ ] Register `after_tool_call` without registering another model tool.
- [ ] Persist only successful exact-identity results and remain fail-open.

## Task 4: Mixed Formation And Review

- [ ] Schedule completed turns through eligible user and tool observations.
- [ ] Form from labeled `user_message` and `tool_result` evidence while rejecting assistant input.
- [ ] Expose bounded source kind and tool label in HTTP and direct OpenClaw review.
- [ ] Keep every candidate proposed until explicit operator governance.

## Task 5: Automated Qualification

- [ ] Pass migration, RLS, FK, idempotency, drift, replay, size, sensitive-data,
  cross-continuity, deletion, reset, scheduler, worker, and review tests.
- [ ] Pass OpenClaw extraction, allowlist, fail-open, identity, request-bound, and package tests.
- [ ] Pass full PostgreSQL, race, Reality, vet, module drift, build, and packaging gates.
- [ ] Preserve every rejected input and failed attempt in the evidence ledger.

## Task 6: Mac Mini Real-Client Evidence

- [ ] Deploy W23 to isolated user-owned Mac mini paths without `sudo`.
- [ ] Produce real OpenClaw `after_tool_call` events from allowed successful and failed tools.
- [ ] Review and accept supported C01-derived outcomes and consume them in a fresh real-model turn.
- [ ] Prove assistant-only, failed, unallowed, sensitive, duplicate, and cross-session input remains absent.
- [ ] Forget one accepted tool-origin memory and prove covered deletion surfaces are clean.

## Task 7: Protected Delivery

- [ ] Write evidence and update public documentation with only proven claims.
- [ ] Commit without amending earlier commits and push the exact head.
- [ ] Verify protected CI, artifacts, packages, signatures, and Mac mini evidence.
- [ ] Update the existing Draft PR with exactly one W23 section.
- [ ] Keep the overall Vermory platform goal active.

# Protected Artifact Signing Implementation Plan

Design: [Protected Artifact Signing Design](../specs/2026-07-18-protected-artifact-signing-design.md)

## Task 1: Freeze Reality Case

- [x] Add and freeze `I04-protected-artifact-signing` before workflow implementation.
- [x] Validate all public reality cases and update authoritative case counts.

## Task 2: Complete Release Manifest

- [ ] Add one portable deterministic manifest generator and verifier.
- [ ] Require exactly four Go archives plus checksums, OpenClaw, Hermes, and Hermes sidecar.
- [ ] Test deterministic output, missing payload rejection, and modified payload rejection.

## Task 3: Protected OIDC Signing

- [ ] Keep the ordinary test job without `id-token: write`.
- [ ] Add a trusted same-repo post-test signing job with minimal permissions.
- [ ] Sign the complete manifest with pinned Cosign and upload the Sigstore bundle.
- [ ] Verify exact workflow identity and issuer plus modified-manifest and wrong-identity rejection.

## Task 4: Manual And Tag Workflow Contract

- [ ] Apply the same complete manifest and signature contract to manual snapshots.
- [ ] Attach the manifest and bundle to draft tagged releases.
- [ ] Keep publication, tags, and releases absent during W24 qualification.

## Task 5: Automated Qualification

- [ ] Pass full PostgreSQL, race, Reality, vet, module drift, OpenClaw, Hermes, and packaging gates.
- [ ] Pass actionlint and static permission/identity assertions for both workflows.
- [ ] Preserve every failed signing, verification, workflow, and evidence attempt.

## Task 6: Mac Mini Protected Evidence

- [ ] Stream the exact signed artifact through the Qingdao reverse-management tunnel without local persistence.
- [ ] Verify transport digest, payload manifest, Sigstore identity/issuer, negative controls, packages, and privacy.
- [ ] Save a complete evidence manifest under the isolated W24 Mac mini root.

## Task 7: Protected Delivery

- [ ] Write public evidence and update only proven documentation claims.
- [ ] Commit without amending earlier commits and push exact heads.
- [ ] Require and verify protected `test` and `sign-snapshot` checks.
- [ ] Update Draft PR 1 with exactly one W24 section.
- [ ] Keep the overall Vermory platform goal active.

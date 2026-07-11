# Security Policy

## Reporting a Vulnerability

Do not open a public issue for a vulnerability that could expose memory content, credentials, tenant data, deleted facts, or continuity bindings.

Use GitHub's private vulnerability reporting or Security Advisory flow for `jstar0/Vermory`. Include:

- affected commit or version;
- reproduction steps with synthetic data;
- expected and observed isolation or deletion behavior;
- whether optional adapters, caches, artifacts, or logs retain the target;
- any known mitigation.

Never include real credentials, private transcripts, or personal memory exports in the report.

## High-Priority Security Boundaries

- cross-tenant and cross-continuity isolation;
- wrong workspace binding;
- deleted-fact residue;
- source prompt injection and unauthorized promotion;
- optional adapter deletion propagation;
- artifact, cache, and audit redaction;
- attestation signature verification.

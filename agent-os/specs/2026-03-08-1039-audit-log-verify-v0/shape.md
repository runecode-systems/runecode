# Audit Log v0 + Verify — Shaping Notes

## Scope

Create a tamper-evident audit log with signed, hash-chained events and a verifier.

## Decisions

- Audit events are append-only and hash-chained.
- Isolate-attributed events must be signed by isolate keys; writers must verify signatures.
- Audit log storage is encrypted at rest by default (no silent plaintext mode).
- Audit logs are segmented for retention/archival without breaking verifiability.

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-crypto-key-mgmt-v0/`, `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`
- Product alignment: Enables cryptographic provenance and auditable evidence.

## Standards Applied

- None yet.

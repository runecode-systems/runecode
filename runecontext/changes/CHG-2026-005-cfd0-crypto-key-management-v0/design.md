# Design

## Overview
Establish the minimum cryptographic root-of-trust for signing manifests, approvals, audit evidence, and authoritative trusted-state integrity in MVP while preserving future remote and distributed deployment optionality.

## Key Decisions
- Signed protocol surfaces use a semantic payload plus detached attestation/signature blocks over the payload's RFC 8785 JCS bytes; RuneCode must not rely on ambiguous "sign the object minus `signatures`" rules.
- `ApprovalRequest` and `ApprovalDecision` are signed objects alongside manifests, audit evidence, and provenance receipts.
- `ApprovalDecision` is signed by a trusted approval authority, not directly by the user's authenticator; any user-held cryptographic assertion is verified by the approval authority and then bound into the signed decision.
- Logical signing authorities are scope-aware and topology-neutral (`deployment`, `node`, `user`, `session`, `publisher`, `store`) and may collapse onto one machine in MVP while remaining logically distinct.
- `secretsd` is the logical custody API for long-lived private keys and authoritative state-integrity keys; custody frontends may be ephemeral.
- Isolates sign their own audit events; the control plane must not be able to forge isolate-attributed events.
- Isolate session private keys are generated and stored inside the isolate boundary, are per-session ephemeral, and are never possessed by the launcher or broker.
- If secure key storage is unavailable (hardware/OS keystore), the system must fail closed by default (no silent plaintext fallback).
- If passphrase-derived encryption is explicitly opted into (dev/portable mode), the KDF and passphrase policy are specified and audited (Argon2id; minimum strength requirements).
- Isolate key provisioning is TOFU for MVP; provisioning mode and handshake binding context are recorded and surfaced as a degraded posture, and later attestation must upgrade the same session-key model rather than replace it.
- Signature metadata include `{alg, key_id}` and resolve through first-class verifier records so algorithm agility and key discovery remain explicit.
- The user-involvement slider may change approval frequency and minimum assurance for ordinary actions, but it must not lower the fixed hard floor for a small set of high-blast-radius operations defined by policy.
- Trusted Go verification is authoritative at narrow daemon boundaries; runner-side TypeScript verification is parity/supporting only.
- Authoritative backup and state integrity under this change applies to canonical RuneCode trusted state, not runner-internal non-canonical orchestration state such as LangGraph checkpoints unless that state is exported into canonical protocol objects.

## Main Workstreams
- Signed Object + Verifier Contract
- Scoped Key Authorities + Posture Recording
- Approval Authority + User-Assurance Hooks
- Sign/Verify Primitives + Verification Placement
- Rotation + Revocation (Minimal)
- Authoritative State Integrity

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.

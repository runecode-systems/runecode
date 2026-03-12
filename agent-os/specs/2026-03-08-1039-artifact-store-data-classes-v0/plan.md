# Artifact Store + Data Classes v0

User-visible outcome: all cross-boundary handoffs are explicit, hash-addressed artifacts with immutable data classification; the system enforces allowed flows between roles.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-artifact-store-data-classes-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Define MVP Data Classes

- Define the MVP starter taxonomy (names + meaning + sensitivity):
  - `spec_text`
  - `unapproved_file_excerpts` (unapproved; viewable locally, not eligible for third-party egress)
  - `approved_file_excerpts`
  - `diffs`
  - `build_logs`
  - `audit_events`
  - `audit_verification_report` (machine-readable audit verification output)
- Include reserved-but-not-used-yet classes for later roles (e.g., `web_query`, `web_citations`) without enabling those roles in MVP.

Approval semantics (MVP):
- `approved_file_excerpts` are created only via an explicit, recorded human approval.
- Promotion must not mutate an existing artifact in place. Implementation must represent approval as minting a new artifact reference (and audit event) that attests the excerpt is approved.

Approval hardening (MVP):
- Promotion requests are size-bounded and rate-limited; bulk promotion requires an explicit, separate approval.
- Approval UI must present the full excerpt content (or an explicit “view full content” affordance) plus origin metadata (repo path, commit, extraction tool/version) before approval.
- Revocation model (policy-level): support a denylist of previously-approved excerpt artifact hashes.
  - Revocation does not delete bytes or rewrite history; it prevents future flows/egress and is recorded as an audit event.

Parallelization: can be implemented in parallel with policy engine and TUI work once the excerpt artifact schema + approval decision schema are stable.

## Task 3: Content-Addressed Artifact Store (CAS)

- Implement a local artifact store interface:
  - `put(stream) -> {hash, size, metadata}`
  - `get(hash) -> stream`
  - `head(hash) -> metadata`
- Ensure hashing is deterministic and uses the canonicalization rules from the schema spec.
- At-rest protection (MVP):
  - store artifacts under encrypted-at-rest storage by default (e.g., inside the encrypted workspace volume)
  - record storage protection posture in audit metadata
  - do not silently fall back to plaintext; require an explicit dev-only override if ever allowed

Parallelization: can be implemented in parallel with audit log storage; coordinate on shared hashing/canonicalization rules.

## Task 4: Flow Matrix Enforcement

- Define a manifest-driven flow matrix: which roles can produce/consume which data classes.
- Enforce at the broker/policy layer (fail-closed).
- Ensure artifacts are immutable: `data_class` cannot change after creation.
- Ensure `unapproved_file_excerpts` never flow to egress roles; only `approved_file_excerpts` may be eligible for model egress when explicitly opted in by the signed manifest.

Parallelization: can be implemented in parallel with broker artifact routing and policy evaluation; it depends on stable role manifests + data-class taxonomy.

## Task 5: Quotas + Limits (Minimal)

- Add per-step and per-role limits for:
  - max artifact count
  - max total bytes
  - max single artifact size
- Record violations as audit events.

Parallelization: can be implemented in parallel with broker rate limits; align quotas with audit metadata for observability.

## Task 6: Garbage Collection + Retention (Minimal)

- Define an MVP retention model to prevent unbounded growth:
  - artifacts referenced by active/retained runs are kept
  - unreferenced artifacts are eligible for deletion based on TTL and/or quota pressure
- Record GC actions (and resulting freed bytes) as audit events.

Operational note (MVP): backup/restore is a first-class concern.
- Define a minimal, deterministic export/backup format (hash manifest + metadata) and restore rules.
- Backup/restore operations must not leak secret-class data across boundaries and should be recorded as audit events.

Parallelization: can be implemented in parallel with audit log retention; coordinate on “retained run” semantics.

## Acceptance Criteria

- Every cross-role handoff references artifacts by hash only.
- Disallowed data flows are blocked deterministically and audited.
- Artifacts can be listed and inspected (metadata) via CLI/TUI.
- Artifact store does not grow without bound; GC can reclaim unreferenced artifacts deterministically.
- `approved_file_excerpts` cannot be produced without a recorded human approval; `unapproved_file_excerpts` are blocked from third-party egress deterministically.

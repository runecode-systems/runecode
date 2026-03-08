# Artifact Store + Data Classes v0

User-visible outcome: all cross-boundary handoffs are explicit, hash-addressed artifacts with immutable data classification; the system enforces allowed flows between roles.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-artifact-store-data-classes-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Define MVP Data Classes

- Define the MVP starter taxonomy (names + meaning + sensitivity):
  - `spec_text`
  - `approved_file_excerpts`
  - `diffs`
  - `build_logs`
  - `audit_events`
- Include reserved-but-not-used-yet classes for later roles (e.g., `web_query`, `web_citations`) without enabling those roles in MVP.

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

## Task 4: Flow Matrix Enforcement

- Define a manifest-driven flow matrix: which roles can produce/consume which data classes.
- Enforce at the broker/policy layer (fail-closed).
- Ensure artifacts are immutable: `data_class` cannot change after creation.

## Task 5: Quotas + Limits (Minimal)

- Add per-step and per-role limits for:
  - max artifact count
  - max total bytes
  - max single artifact size
- Record violations as audit events.

## Task 6: Garbage Collection + Retention (Minimal)

- Define an MVP retention model to prevent unbounded growth:
  - artifacts referenced by active/retained runs are kept
  - unreferenced artifacts are eligible for deletion based on TTL and/or quota pressure
- Record GC actions (and resulting freed bytes) as audit events.

## Acceptance Criteria

- Every cross-role handoff references artifacts by hash only.
- Disallowed data flows are blocked deterministically and audited.
- Artifacts can be listed and inspected (metadata) via CLI/TUI.
- Artifact store does not grow without bound; GC can reclaim unreferenced artifacts deterministically.

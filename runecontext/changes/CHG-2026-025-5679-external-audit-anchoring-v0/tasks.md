# Tasks

## Later Anchor Target Model

- [ ] Define later non-MVP anchor targets, including the planned `anchor_kind` families:
  - transparency-log style anchors
  - timestamp-authority style anchors
  - public-chain style anchors
- [ ] Freeze one concrete first runtime adapter, with transparency-log style anchoring as the recommended initial target family, while keeping the target contracts additive for timestamp-authority and public-chain follow-on work.
- [ ] Keep each target kind on a typed target descriptor and typed anchor receipt payload contract instead of a freeform blob.
- [ ] Make the typed target descriptor digest the authoritative target identity used by policy, approval, lease binding, audit, and verification rather than raw URL strings.
- [ ] Keep transport-specific endpoints as derived execution details below the typed target descriptor rather than the canonical policy surface.
- [ ] Preserve the shared `AuditReceipt(kind=anchor)` envelope and `AuditSegmentSeal` subject model from `runecontext/changes/CHG-2026-006-84f0-audit-anchoring-v0/` while adding target-specific typed fields.
- [ ] Keep external anchor payloads additive to the shared receipt model rather than introducing a second external-only receipt family or duplicating top-level subject fields.
- [ ] Keep the shared anchor receipt minimal and move larger target-specific proof bytes into typed authoritative sidecar evidence referenced by digest.

## Egress + Trust Boundary Model

- [ ] External anchoring requires explicit signed-manifest opt-in and must never silently enable network access.
- [ ] External anchor traffic must use an explicit allowlist and a non-workspace execution pathway.
- [ ] Align authenticated external anchor submissions with the shared remote-state-mutation gateway class rather than an ad hoc external-only egress category.
- [ ] Add a provider-neutral typed external anchor request family on the shared gateway path rather than reusing a local-only ad hoc anchor action hash.
- [ ] Define canonical target identity and matching rules rather than target-local raw URL policy.
- [ ] Define how policy and audit distinguish:
  - local-only anchoring
  - configured-but-not-run external anchoring
  - completed external anchoring
- [ ] Secret material for target authentication, if any, must follow the same no-env-var/no-raw-log posture as other gateway-style integrations.
- [ ] Keep `v0` external anchor submission as an exact-action approval boundary per outbound submission when remote target state is mutated.
- [ ] Shape later signed-manifest automation as an additive posture that reuses the same typed prepared and execute path, request hashes, policy bindings, and lease bindings rather than creating an automation-only route.
- [ ] Bind exact-action approval to the canonical typed external anchor request hash and canonical target descriptor identity.
- [ ] Keep later external anchor execution compatible with `CHG-2026-059-7b31-cross-machine-evidence-replication-restore-v0` so any future durability barrier or remote-state-mutation recovery requirement reuses the shared trusted prepare, execute, and reconcile model rather than inventing an anchor-local exception path.

## Execution Lifecycle And Performance Foundation

- [ ] Introduce a durable prepared and execute lifecycle for external anchor submission with `prepare`, `get`, and `execute` control-plane semantics.
- [ ] Allow `execute` to complete inline when possible while making `deferred` a normal first-class result.
- [ ] Bind durable attempt idempotency to immutable request inputs, including seal digest, canonical target descriptor identity, and typed request hash.
- [ ] Keep network I/O, remote polling, and external waits outside the audit ledger mutex.
- [ ] Snapshot immutable segment and seal inputs under lock, perform outbound work without the ledger lock, then reacquire only for final compare-and-persist work.
- [ ] Use bounded worker concurrency and explicit retry or backoff so the same architecture works on constrained local machines and larger deployments.
- [ ] Avoid making full segment replay the only hot-path receipt admission mechanism when only external anchor evidence changed.

## Receipt, Audit, and Verification Integration

- [ ] Store external anchor receipts as sidecar audit evidence and optional exported artifacts while keeping `AuditSegmentSeal` as the anchoring subject.
- [ ] Keep authoritative storage sidecar-first and treat exported artifacts as copies of the authoritative receipt rather than a second trust source.
- [ ] Verification output must distinguish:
  - valid external anchors
  - deferred or unavailable anchors
  - invalid anchors
- [ ] Verification remains fail closed on invalid receipts, never rewrites existing audit history, and stays aligned to the shared `AuditVerificationReport` dimension model.
- [ ] Record canonical target identity, anchoring subject identity, outbound payload or subject hash, bytes, timing, outcome, and any relevant lease or policy bindings in audit evidence.
- [ ] Preserve attestation evidence and verification references when the anchored audit subject depends on attested runtime evidence rather than flattening them into launch-only or target-local summaries.
- [ ] Reuse validated project-substrate snapshot identity in anchored evidence when the anchored audit subject depends on project context.
- [ ] Preserve raw target proof bytes, provider receipts, and verification transcripts as typed sidecar evidence referenced by digest rather than embedding large provider-specific blobs in the shared receipt envelope.
- [ ] Support incremental receipt and proof verification over an already-verified seal so repeated external anchor submissions do not require full verifier replay as the only normal path.
- [ ] Keep full verification recomputation available as the authoritative recovery and audit check path.

## Target-Set Semantics

- [ ] Support target-set foundations without requiring multi-target runtime scope in the first adapter.
- [ ] Define aggregate required-target semantics as:
  - `ok` when all required targets are satisfied with valid evidence
  - `degraded` when no invalid evidence exists but one or more required targets remain deferred, unavailable, or unsatisfied
  - `failed` when any required target has invalid evidence or any authoritative persisted receipt for the anchored subject is invalid
- [ ] Keep optional supplemental targets visible in per-target results and findings without blocking aggregate `ok` once the required target set is satisfied.
- [ ] Reserve quorum-style policies such as `min_valid_required` for later additive work rather than overloading `required` semantics in `v0`.

## Fixtures + Adapter Conformance

- [ ] Add checked-in fixtures for representative external anchor receipts, target-specific sidecar proof objects, and invalid cases for each supported target kind.
- [ ] Keep fixture updates explicit and reviewable; CI verifies but does not regenerate them implicitly.
- [ ] Add fixtures that cover deferred execution, unavailable targets, invalid target proof, target identity mismatch, and exact-action binding mismatch.

## Acceptance Criteria

- [ ] External anchoring targets are defined in a dedicated later spec rather than remaining as a note in the MVP anchoring spec.
- [ ] External anchoring never silently enables network access.
- [ ] Receipt verification is typed, auditable, and fail closed on invalid data.
- [ ] External anchoring inherits shared gateway identity, lease, approval, and audit-evidence discipline rather than inventing an external-only outbound model.
- [ ] `v0` external anchor submission uses exact-action approval and a durable prepared and execute lifecycle that supports deferred completion without introducing a second trust path.
- [ ] The first runtime adapter is one concrete target family, with transparency-log style anchoring recommended first, while the contracts remain additive for later target kinds.
- [ ] The performance foundation avoids network I/O under audit ledger lock and supports incremental receipt admission so the same architecture remains viable on constrained and scaled environments.
- [ ] Aggregate required-target posture follows `all required targets satisfied`, with quorum-style policies deferred to later additive work.

# Tasks

## Later Anchor Target Model

- [ ] Define later non-MVP anchor targets, including the planned `anchor_kind` families:
  - transparency-log style anchors
  - timestamp-authority style anchors
  - public-chain style anchors
- [ ] Keep each target kind on a typed target descriptor and typed anchor receipt payload contract instead of a freeform blob.
- [ ] Preserve the shared `AuditReceipt(kind=anchor)` envelope and `AuditSegmentSeal` subject model from `runecontext/changes/CHG-2026-006-84f0-audit-anchoring-v0/` while adding target-specific typed fields.
- [ ] Keep external anchor payloads additive to the shared receipt model rather than introducing a second external-only receipt family or duplicating top-level subject fields.

## Egress + Trust Boundary Model

- [ ] External anchoring requires explicit signed-manifest opt-in and must never silently enable network access.
- [ ] External anchor traffic must use an explicit allowlist and a non-workspace execution pathway.
- [ ] Define how policy and audit distinguish:
  - local-only anchoring
  - configured-but-not-run external anchoring
  - completed external anchoring
- [ ] Secret material for target authentication, if any, must follow the same no-env-var/no-raw-log posture as other gateway-style integrations.

## Receipt, Audit, and Verification Integration

- [ ] Store external anchor receipts as sidecar audit evidence and optional exported artifacts while keeping `AuditSegmentSeal` as the anchoring subject.
- [ ] Keep authoritative storage sidecar-first and treat exported artifacts as copies of the authoritative receipt rather than a second trust source.
- [ ] Verification output must distinguish:
  - valid external anchors
  - deferred or unavailable anchors
  - invalid anchors
- [ ] Verification remains fail closed on invalid receipts, never rewrites existing audit history, and stays aligned to the shared `AuditVerificationReport` dimension model.

## Fixtures + Adapter Conformance

- [ ] Add checked-in fixtures for representative external anchor receipts and invalid cases for each supported target kind.
- [ ] Keep fixture updates explicit and reviewable; CI verifies but does not regenerate them implicitly.

## Acceptance Criteria

- [ ] External anchoring targets are defined in a dedicated later spec rather than remaining as a note in the MVP anchoring spec.
- [ ] External anchoring never silently enables network access.
- [ ] Receipt verification is typed, auditable, and fail closed on invalid data.

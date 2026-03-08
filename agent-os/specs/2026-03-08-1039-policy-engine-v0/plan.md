# Policy Engine v0

User-visible outcome: RuneCode deterministically allows/denies actions based on signed manifests and role rules, with explicit human approvals for elevated risk.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-policy-engine-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Role + Run/Stage Policy Model

- Define how role manifests and run/stage capability manifests combine into an effective policy.
- Bind all decisions to the manifest hash (no “implicit” policy).
- Define the MVP policy language and evaluation semantics:
  - declarative, schema-validated policy documents (no embedded scripting)
  - explicit precedence rules (e.g., explicit deny > require_human_approval > allow)
  - stable reason codes and structured decision details
- Define policy loading/tamper resistance:
  - policy inputs are content-addressed and bound to signed manifests
  - policy evaluation rejects inputs that do not validate against the schema version

## Task 3: Invariants (Fail Closed)

- Enforce MVP invariants:
  - no escalation-in-place
  - deny-by-default for network/filesystem/shell/secrets
  - no single role combines workspace RW + public egress + long-lived secrets

## Task 4: Backend Selection Rules

- MicroVM is the default backend when available.
- Container backend is only allowed with an explicit opt-in recorded as an approval + audit event.
- The system must not automatically fall back from microVM to containers.

## Task 5: Decision Outputs

- Standardize policy decisions:
  - `allow | deny | require_human_approval`
  - stable reason codes
  - structured “required approvals” payloads
- Decision artifacts must include hashes of all evaluated inputs (manifest hash, request hash, relevant artifact hashes).

## Acceptance Criteria

- Every action request is evaluated deterministically and produces a policy decision artifact.
- Policy evaluation does not execute arbitrary code and is deterministic for identical inputs.
- Container usage is blocked unless explicitly opted in and recorded.
- Violations are auditable and do not partially execute.

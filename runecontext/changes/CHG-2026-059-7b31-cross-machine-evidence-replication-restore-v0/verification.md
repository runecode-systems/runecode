# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the change preserves the current verification-plane distinction between canonical evidence, derived surfaces, and derivative evidence bundles.
- Confirm signed replication checkpoints are introduced as a new object family rather than overloading `AuditEvidenceBundleManifest` into federation authority.
- Confirm remote S3-compatible targets are treated as durable storage substrates and not as semantic truth surfaces.
- Confirm tenant and project namespace layout remains storage-only and does not replace canonical typed identity.
- Confirm historical local GC is allowed only after trusted remote durability confirmation and leaves enough compact local skeleton state for new ordinary work.
- Confirm the change defines at least `healthy`, `remote_durability_degraded`, and `local_capture_unhealthy` durability posture with the intended action gates.
- Confirm one remote target is explicitly degraded posture and healthy self-healing requires two independent remote targets.
- Confirm publication-sensitive actions require a pre-action durability barrier and durable prepare, execute, and reconcile semantics rather than a best-effort flush.
- Confirm degraded-state changes have no permanent lower-assurance publication lane and are only eligible for re-creation through a new healthy audited run.
- Confirm fetch-on-miss, restore, and anti-entropy are checkpoint-driven and fail closed on ambiguous or unverifiable remote content.
- Confirm any optional helper remains in the trusted domain and does not become a second public authority or restore-admission surface.

## Close Gate
Use the repository's standard verification flow before closing this change.

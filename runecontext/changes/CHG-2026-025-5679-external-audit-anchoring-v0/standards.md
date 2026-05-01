## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/approval-binding-and-verifier-identity.md`
- `standards/security/secret-lease-lifecycle-and-binding.md`
- `standards/security/audit-anchoring-receipt-and-verification.md`
- `standards/security/audit-verification-scope-and-evidence-binding.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/deterministic-check-write-tools.md`

## Resolution Notes
Migrated from the legacy spec standards list and refreshed to canonical RuneContext standard paths, then expanded to anchor external-target work to shared audit receipt, verification posture, and typed control-plane contract discipline.

This now also includes shared trust-boundary, approval-binding, and lease-binding rules so authenticated external anchoring inherits the same reviewed remote-mutation discipline used by other high-risk outbound lanes.

The captured decisions also rely on the existing control-plane requirement that prepared, get, and execute style request lifecycles remain typed and topology-neutral, and on the shared policy rule that exact-action approvals continue to bind canonical request hashes rather than ambient operator posture.

## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/trusted-local-artifact-persistence.md`
- `standards/security/model-provider-substrate-and-secret-ingress.md`
- `standards/security/secret-lease-lifecycle-and-binding.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/local-first-future-optionality.md`

## Resolution Notes
Expanded to include the shared policy-foundation, trust-boundary, trusted-persistence, secret-ingress, and control-plane contract standards for typed package-registry destinations, explicit offline-vs-egress behavior, and broker-owned cache authority.

This now explicitly includes reuse of the shared gateway operation taxonomy and shared gateway audit evidence model where no dependency-specific exception is required.

This also now explicitly captures:
- checkpoint-style approval semantics for dependency scope enablement or expansion rather than per-fetch approval
- topology-neutral cache identity and artifact contracts suitable for local-first and later larger-scale deployments
- secret-lease and auth-material rules needed to keep private-registry support additive rather than foundationally disruptive

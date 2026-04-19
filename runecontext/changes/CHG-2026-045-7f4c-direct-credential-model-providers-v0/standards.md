## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/trust-boundary-change-checklist.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/audit-verification-scope-and-evidence-binding.md`
- `standards/global/control-plane-api-contract-shape.md`

## Resolution Notes
This change exists to make pre-beta direct-credential model access land on the same durable provider substrate that later OAuth and bridge-runtime features will reuse.

That includes preserving one shared provider-profile and auth-material model, one canonical typed model boundary, one secrets-custody posture, and one broker-projected readiness and compatibility surface.

Provider profiles and auth material must remain distinct so auth-mode changes, credential rotation, and later OAuth or bridge adoption do not rewrite provider identity or readiness semantics.

Direct credential setup must use broker-owned setup and secret-ingress flows that CLI and TUI both consume. Secret values must stay out of CLI args, environment variables, and ordinary typed broker request and response bodies.

Popular SDKs and public provider APIs may inform adapter implementation, but they are not authoritative control-plane contracts. Provider-specific wire payloads remain below the canonical typed model boundary unless a later typed extension is explicitly reviewed.

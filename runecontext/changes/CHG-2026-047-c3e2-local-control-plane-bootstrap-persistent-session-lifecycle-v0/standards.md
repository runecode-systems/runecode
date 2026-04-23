## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/runner-durable-state-and-replay.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/local-first-future-optionality.md`
- `standards/global/project-substrate-contract-and-lifecycle.md`

## Resolution Notes
This change exists to make RuneCode feel like one attachable local product without introducing daemon-private lifecycle truth or a second control plane.

That includes keeping attach, detach, and reconnect semantics broker-owned and topology-neutral so later platform-specific service realization stays additive rather than architectural.

This change also freezes the following implementation posture so later features build on one reviewed foundation instead of accreting lifecycle semantics opportunistically:
- one local RuneCode product instance per authoritative repository root
- a dedicated broker-owned typed product lifecycle posture surface distinct from readiness, version, and project-substrate posture
- a canonical top-level `runecode` user command for product lifecycle and attach flows
- private local bootstrap and supervision mechanics that remain non-authoritative and topology-neutral from the client contract perspective
- explicit separation between session object lifecycle, projected session work posture, and client attachment state
- diagnostics/remediation-only attach when services are healthy but current repository project-substrate posture blocks normal managed operation

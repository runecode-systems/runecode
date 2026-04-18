## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/runner-durable-state-and-replay.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/local-first-future-optionality.md`

## Resolution Notes
This change exists to make RuneCode feel like one attachable local product without introducing daemon-private lifecycle truth or a second control plane.

That includes keeping attach, detach, and reconnect semantics broker-owned and topology-neutral so later platform-specific service realization stays additive rather than architectural.

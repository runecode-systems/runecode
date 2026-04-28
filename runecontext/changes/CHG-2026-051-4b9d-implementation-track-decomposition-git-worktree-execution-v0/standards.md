## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/runner-durable-state-and-replay.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/approval-binding-and-verifier-identity.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/local-product-lifecycle-and-attach-contract.md`
- `standards/global/project-substrate-contract-and-lifecycle.md`

## Resolution Notes
This change exists to give approved-change implementation work one reviewed path for explicit or inferred track decomposition and isolated git-worktree execution without drifting into hidden scheduler heuristics or client-local coordination truth.

That includes freezing the following clarifications for this future foundation:
- explicit track declarations override inferred grouping
- inferred decomposition becomes a broker-owned proposed execution-plan artifact
- pending operator input or approval remains dependency-aware partial blocking rather than a whole-system stop signal
- unrelated eligible tracks may continue only when plan, dependency graph, policy, coordination state, and project-substrate posture all allow it
- git worktree mechanics remain implementation-private while broker-owned track, session, run, approval, artifact, audit, and project-context identities remain canonical

This change builds on session execution orchestration, workflow definition binding, and first-party workflow-pack foundations rather than redefining those authority surfaces locally.

That now also includes the `CHG-049` clarifications that:
- approved implementation work is already bound to reviewed implementation-input sets and exact digests before this change starts decomposing it
- the initial `v0` baseline remains at most one mutation-bearing shared-workspace run per authoritative repository root unless and until later reviewed concurrency or worktree execution rules explicitly extend it

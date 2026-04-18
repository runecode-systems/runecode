## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/approval-binding-and-verifier-identity.md`
- `standards/security/audit-verification-scope-and-evidence-binding.md`
- `standards/security/secret-lease-lifecycle-and-binding.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/local-first-future-optionality.md`
- `standards/global/deterministic-check-write-tools.md`

## Resolution Notes
Expanded to include trust-boundary interfaces, audit-evidence binding, and local-first control-plane standards because this change now freezes more than a simple gateway lane.

The resulting git-gateway foundation must preserve all of the following:
- one shared gateway and approval model rather than a git-only exception path
- typed git request objects as sole authority, with summary projections as derived-only UX data
- migration away from `GitRemoteMutationSummary` as any authority surface for git-lane behavior
- logical repository identity and exact approval binding rather than raw URL policy
- artifact-managed repository policy, including optional repository-specific commit rules such as DCO
- broker `prepare/get/execute` request-union contract with CLI and TUI as friendly thin adapters
- trusted Go orchestration with native git for local mutation/ref push and Go provider adapters for provider APIs
- GitHub-first provider delivery under provider-neutral contracts
- broker-owned setup and configuration semantics with TUI and CLI as thin clients
- auditable, lease-bound, fail-closed remote mutation with no second secrets store or second policy authority

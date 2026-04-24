# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the roadmap and change text both describe LangGraph as optional and post-MVP rather than required.
- Confirm the change explicitly states that implementation should be determined later based on whether it is still needed.
- Confirm the change keeps LangGraph internal-only and non-canonical.
- Confirm the change preserves broker-owned run truth, approval truth, lifecycle state, and immutable `RunPlan` authority.
- Confirm the change does not weaken trust-boundary, approval-binding, or fail-closed recovery expectations.
- Confirm the change preserves exact-action wait semantics for `git_remote_ops` and similar hard-floor remote-state-mutation approvals.
- Confirm wait, replay, and resume preserve canonical action hashes, relevant artifact hashes, and expected result tree identity where those bindings exist.
- Confirm remote-drift handling remains fail closed under any LangGraph-backed resume path.
- Confirm validated project-substrate snapshot binding and repository substrate drift handling remain fail closed for project-context-sensitive execution under any LangGraph-backed resume path.
- Confirm the change preserves distinct `waiting_operator_input` and `waiting_approval` semantics under any LangGraph-backed runtime path.
- Confirm the change requires support for multiple simultaneous scoped waits and dependency-aware partial blocking rather than a whole-run paused flag.

## Close Gate
Use the repository's standard verification flow before closing this change.

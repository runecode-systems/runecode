# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm this change captures the contract-first workflow substrate split out from `CHG-2026-017`.
- Confirm the change reuses shared identity, executor, gate, approval, audit, and runner-to-broker contracts rather than inventing process-local variants.
- Confirm workflow definitions reuse the shared wait vocabulary, including distinct `waiting_operator_input` and `waiting_approval`, rather than defining process-local wait kinds.
- Confirm workflow definitions encode enough dependency and continuation structure for scoped blocking instead of forcing a whole-workflow stop model.
- Confirm `approval_profile` and `autonomy_posture` remain separate shared inputs to workflow selection and execution.
- Confirm workflow-composed git remote mutation still routes through shared typed git request, patch artifact, repository identity, and exact-action approval semantics.
- Confirm workflow-composed dependency fetch still routes through shared typed dependency-fetch request identity, broker-owned cache authority, and shared gateway approval/audit semantics.
- Confirm workflow definitions do not treat raw lockfile bytes, tool-private cache paths, or unpacked dependency trees as authoritative dependency identity.
- Confirm offline cached dependency use inside workflow execution is modeled as broker-mediated internal artifact handoff rather than egress.
- Confirm the workflow contract remains compatible with the public-registry-first first slice and does not require private-registry credential semantics in the foundational contract.
- Confirm project-context-sensitive workflow selection or execution reuses the shared validated project-substrate snapshot-binding model.
- Confirm project-context-sensitive workflows fail closed on blocked repository substrate posture.
- Confirm workflow definitions do not embed alternate project discovery, init, adopt, or upgrade semantics.
- Confirm generic authoring UX and shared-memory accelerators remain out of scope here.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.8`.

## Close Gate
Use the repository's standard verification flow before closing this change.

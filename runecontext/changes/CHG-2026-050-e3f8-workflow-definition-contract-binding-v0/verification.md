# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm this change captures the contract-first workflow substrate split out from `CHG-2026-017`.
- Confirm the change explicitly separates `ProcessDefinition` as executable graph authority, `WorkflowDefinition` as reviewed selection/binding authority, and `RunPlan` as runtime execution authority.
- Confirm the change keeps one topology-neutral architecture across constrained local and scaled deployments rather than introducing environment-specific contract or trust paths.
- Confirm the change reuses shared identity, executor, gate, approval, audit, and runner-to-broker contracts rather than inventing process-local variants.
- Confirm the change freezes a normalized DAG-based workflow/process graph shape rather than leaving executable control flow implicit, nested-DSL-shaped, or runner-inferred.
- Confirm `v0` forbids general loops/cycles and relies on separate attempt identities for retries and reruns.
- Confirm workflow definitions reuse the shared wait vocabulary, including distinct `waiting_operator_input` and `waiting_approval`, rather than defining process-local wait kinds.
- Confirm workflow definitions encode enough dependency and continuation structure for scoped blocking instead of forcing a whole-workflow stop model.
- Confirm the change encodes dependency-aware independence without promising new runtime parallel-execution semantics in `v0`.
- Confirm `approval_profile` and `autonomy_posture` remain separate shared inputs to workflow selection and execution.
- Confirm canonical definition hashing is based on RFC 8785 JCS bytes and that selected workflow/process definitions bind into broker-owned signed compilation/selection evidence.
- Confirm broker-owned compiled-plan authority is the intended runtime source of truth and that trusted-artifact rescans are not preserved as the long-term execution-authority model.
- Confirm workflow-composed git remote mutation still routes through shared typed git request, patch artifact, repository identity, and exact-action approval semantics.
- Confirm workflow-composed dependency fetch still routes through shared typed dependency-fetch request identity, broker-owned cache authority, and shared gateway approval/audit semantics.
- Confirm workflow definitions do not treat raw lockfile bytes, tool-private cache paths, or unpacked dependency trees as authoritative dependency identity.
- Confirm workflow/process definitions prefer reference-oriented dependency requirement binding over embedding workflow-local raw resolver authority.
- Confirm offline cached dependency use inside workflow execution is modeled as broker-mediated internal artifact handoff rather than egress.
- Confirm the workflow contract remains compatible with the public-registry-first first slice and does not require private-registry credential semantics in the foundational contract.
- Confirm project-context-sensitive workflow selection or execution reuses the shared validated project-substrate snapshot-binding model.
- Confirm project-context-sensitive compiled plans bind the exact validated project-substrate digest and fail closed on incompatible drift during resume/continuation.
- Confirm project-context-sensitive workflows fail closed on blocked repository substrate posture.
- Confirm workflow definitions do not embed alternate project discovery, init, adopt, or upgrade semantics.
- Confirm schema-version evolution is explicit where the richer workflow/process/run-plan substrate materially changes object shape or authority semantics.
- Confirm generic authoring UX and shared-memory accelerators remain out of scope here.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.8`.

## Close Gate
Use the repository's standard verification flow before closing this change.

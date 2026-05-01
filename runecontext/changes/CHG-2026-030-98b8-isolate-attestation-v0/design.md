# Design

## Overview
Define isolate attestation evidence and verification that upgrades MVP TOFU compatibility metadata to required measured attestation for the normal supported runtime path, while inheriting the signed runtime-asset identity foundation established by `CHG-2026-026-98be-image-toolchain-signing-pipeline`.

## Key Decisions
- All supported production and user-facing runtime paths require valid attestation and fail closed when evidence is unavailable, invalid, replayed, or freshness-deficient.
- TOFU remains only as a non-production compatibility mechanism for tests, fixtures, fake backends, and implementation scaffolding; it is not a supported operator-visible posture, runtime option, or fallback path.
- Attestation upgrades the existing `IsolateSessionBinding` model from `CHG-2026-009-1672-launcher-microvm-backend-v0`; it does not replace the existing session-binding contract with a different identity model.
- Attestation uses additive immutable evidence records rather than mutating the original binding object in place.
- Attestation adds stronger evidence; it does not replace the need for explicit binding to session identity, image identity, and provisioning evidence.
- Verifier, policy, broker projection, and TUI surfaces must expose provisioning posture explicitly and must not collapse valid, unavailable, invalid, or replayed attestation outcomes into one ambiguous status field.
- Invalid or replayed attestation evidence fails closed when an attested posture is required.
- Attestation evidence binds to the same `session_nonce`, `handshake_transcript_hash`, signed runtime-image descriptor identity, boot-profile identity, and concrete boot-component digests established by the reviewed runtime-asset admission and launch flow rather than by ambient platform-specific launch assumptions.
- Attestation references persisted launch/runtime evidence from the signed runtime-asset pipeline where needed, but it must not redefine runtime identity independently of that pipeline.
- The attestation upgrade path preserves compatibility with the `isolate_session_started` and `isolate_session_bound` audit event families.
- Attested runtime identity remains distinct from validated project-substrate snapshot identity; later evidence may bind both when relevant, but attestation must not redefine project-context identity.
- Platform-specific attestation sources may differ across Linux, macOS, Windows, microVM, or container realizations, but the shared verifier contract and operator-facing runtime identity semantics remain backend-neutral and platform-neutral.
- Performance optimizations may cache verification results, prewarm runtime assets, or add warm-pool mechanics, but they must not weaken trust semantics, launch semantics, or audit semantics and must not introduce separate small-device and large-deployment architectures.

## Goals
- Replace TOFU-only normal provisioning posture with required attestation for the supported runtime path.
- Forbid TOFU trust decisions in all production and user-facing runtime flows.
- Preserve the per-session isolate key as the durable isolate identity root.
- Bind attestation to reviewed signed runtime identity rather than to ambient backend assumptions.
- Preserve the existing audit event families and runtime posture model.
- Keep the contract offline-capable and topology-neutral.
- Establish a verification and caching model that remains efficient from constrained devices to larger deployments.

## Non-Goals
- Defining public operator-visible identity around backend-specific attestation vendor claims.
- Moving authority for runtime attestation into the runner or any untrusted component.
- Introducing a second runtime truth surface separate from the signed runtime-asset pipeline.
- Using project-substrate identity as a substitute for runtime identity.
- Normalizing TOFU as a supported steady-state posture for production-like runtime operation.
- Providing any production, user-facing, or supported operator override that permits TOFU trust decisions.

## Attestation Architecture

### One Runtime Trust Path
The required runtime trust path is:

`publish -> sign -> trusted admission -> verified local cache -> launch -> secure session -> collect attestation -> trusted verification -> persisted evidence -> broker projection -> audit and verification`

This path remains the same on constrained local devices and on larger scaled deployments.

Scaling may change:
- how verified runtime assets reach the node
- whether assets are prewarmed
- whether warm pools are used
- whether verification results are reused from local cache

Scaling must not change:
- what is trusted
- what is verified
- when launch is allowed
- what evidence is persisted
- what posture is shown to operators

### Supported Runtime Policy
- Supported production and user-facing runtime operation requires valid attestation.
- Missing attestation source, invalid evidence, replayed evidence, or unverifiable freshness fails closed.
- No automatic TOFU fallback is permitted.
- No manual production or user-facing override to TOFU is permitted.
- No release-mode policy, documented operator flow, default configuration, CLI flag, TUI action, or constrained-device exception may permit TOFU trust decisions.
- TOFU may remain in tests, fixtures, fake backends, and implementation scaffolding so compatibility paths and failure cases stay reviewable, but it is not part of the supported product security posture.
- If TOFU is observed outside those non-production contexts, that is a defect rather than a degraded supported posture.

## Evidence Model

### Baseline Binding
The existing per-session isolate key binding remains the baseline identity root.

That baseline still binds at least:
- `{run_id, isolate_id, session_id}`
- `session_nonce`
- `handshake_transcript_hash`
- per-session isolate key identity
- launch/runtime identity context

Attestation upgrades that same logical binding. It does not replace it.

### New Immutable Records
Add two new logical evidence families.

#### `IsolateAttestationEvidence`
This is the immutable record of collected attestation claims and the normalized trusted input to verification.

It should bind at least:
- `run_id`
- `isolate_id`
- `session_id`
- `session_nonce`
- `handshake_transcript_hash`
- `isolate_session_key_id_value`
- `launch_runtime_evidence_digest`
- `runtime_image_descriptor_digest`
- boot-profile identity
- concrete launched boot-component digests
- `measurement_profile`
- normalized attestation source kind
- source-issued freshness material and freshness binding claims
- normalized measurement claims or a digest-addressed normalized measurement payload
- `evidence_digest`

The evidence object may retain backend- or platform-specific raw source details in implementation-private or detailed evidence surfaces, but the shared verifier input must be normalized enough that operator-facing semantics remain stable across backends.

#### `IsolateAttestationVerificationRecord`
This is the immutable trusted verification result for one `IsolateAttestationEvidence` record evaluated against one verifier-policy state.

It should bind at least:
- `attestation_evidence_digest`
- verifier authority state identity
- verification rules/profile version used
- verification timestamp
- verification result
- stable machine reason codes
- replay verdict
- any derived normalized measurement digest or digest set used for comparison

This split is deliberate:
- raw collected evidence stays reviewable
- verification can be rerun under new authority state or policy
- broker projections can cache results by immutable identity
- audit and TUI can explain whether evidence existed but was rejected

### Why Additive Immutable Records
This adds complexity, but it is the best foundation because it preserves the difference between:
- what was observed at session establishment
- what was later proven about that session
- how trusted policy evaluated that proof

That fits the repo's existing evidence-first model and avoids mutating the meaning of prior records.

## Runtime Identity Binding

### Primary Binding Target
Attestation binds primarily to persisted launch/runtime evidence from the signed runtime-asset pipeline, not to ambient host state and not to free-floating descriptor fields alone.

The preferred primary join key is the persisted launch/runtime evidence digest from the launcher-produced runtime evidence model, because that record already captures:
- admitted runtime-image descriptor identity
- boot profile identity
- concrete boot-component digests
- runtime signer and verifier identity
- authority state linkage

### Secondary Explicit Bindings
Attestation evidence should also carry explicit copies or digest references for:
- `runtime_image_descriptor_digest`
- boot profile identity
- concrete launched boot-component digests

Those fields make verification and operator explanation easier without creating a second runtime identity path.

## Measurement Profile Contract

### Closed Reviewed Vocabulary
`measurement_profile` must be a closed, versioned reviewed vocabulary owned by trusted Go code.

Each profile defines:
- required claim inputs
- canonicalization rules
- freshness rules
- accepted attestation source classes
- normalized output shape
- final digest derivation rules

### `expected_measurement_digests` Semantics
`expected_measurement_digests` should mean allowed final normalized measurement digests for the declared `measurement_profile`.

They should not mean:
- ambiguous per-component partial matches
- informal backend-specific claim bundles
- heuristic matching against whichever fields happened to be present

Trusted verification must reduce raw platform-specific evidence to one deterministic normalized digest or one deterministic normalized digest set before matching.

## Replay And Freshness Model

### Replay Identity
Replay protection must bind to the exact live session, not to time alone.

Replay identity should include at least:
- `{run_id, isolate_id, session_id}`
- `session_nonce`
- `handshake_transcript_hash`
- `isolate_session_key_id_value`
- `launch_runtime_evidence_digest`
- `measurement_profile`
- normalized attestation evidence identity

### Freshness Rule
If a backend-specific attestation source cannot prove freshness bound to the current live session, it does not qualify for `attested` posture.

### Replay Outcome
Replay is not a benign degraded state.
Replay is a verification failure class that must surface as invalid evidence with a dedicated machine reason such as `replay_detected`.

## Posture Model

### Coarse Public Posture
Keep the existing operator-facing coarse posture model:
- `tofu`
- `attested`
- `not_applicable`
- `unknown`

This preserves the current run-surface vocabulary.

### Detailed Trusted Posture
Add a second explicit detailed attestation posture axis for trusted evidence, broker projection, TUI, and verification:
- `tofu_only`
- `valid`
- `unavailable`
- `invalid`
- `not_applicable`
- `unknown`

Replay is represented as `invalid` with a dedicated machine reason code rather than as a separate top-level posture.

This keeps the public vocabulary stable while still allowing policy and UX to distinguish:
- evidence was never available
- evidence existed but failed verification
- evidence was replayed
- evidence was valid and upgraded the session

## Verification And Caching

### Verification Location
Trusted Go code remains the authoritative verification boundary.
Verification must not depend on runner-owned logic or online third-party verifier discovery during normal launch.

### Verification Result Caching
Cache attestation verification results by immutable identity, at least:
- `attestation_evidence_digest`
- verifier authority state digest
- `measurement_profile`

This is the main performance foundation for the change.

It allows:
- repeated broker read-model projection without redoing expensive verification work
- deterministic restart reconstruction
- the same architecture on small and large nodes

It must not allow:
- bypass of actual verification on first use
- cache reuse across changed verifier authority state
- cache reuse across changed normalized evidence identity

### Offline-Capable Operation
Once signed runtime assets and verifier policy are admitted locally, normal launch and attestation verification should remain offline-capable.

## Audit And Operator Surfaces

### Existing Event Families Stay
`isolate_session_started` and `isolate_session_bound` remain the event families.

`attestation_evidence_digest` stays the forward-compatible digest reference that links those events to attestation evidence.

### Emission Rule
Audit emission must treat attestation as a material posture change, not as metadata that can be hidden behind the original TOFU emission marker.

That means:
- a baseline session-started event can still exist for session establishment evidence
- a session-bound event must be emitted when authoritative posture changes materially for the same session identity
- deduplication identity must include attestation identity when attestation changes the authoritative posture

### Operator Explanation
TUI and verification views must be able to explain separately:
- a session binding exists
- attestation evidence exists
- attestation verification succeeded or failed

This avoids confusing operators with one overloaded status label.

## Project Identity Separation
Attested runtime identity remains distinct from validated project-substrate snapshot identity.

Later features may bind both through one explicit typed reference seam, but this change must not:
- put project-context identity inside runtime-image fields
- treat runtime attestation as project validation
- create multiple competing project truth surfaces

## Cross-Platform Contract
Linux, macOS, Windows, microVM, and container backends may gather different raw source evidence, but they must share:
- the same logical posture model
- the same trusted verifier contract shape
- the same binding requirements to session and runtime identity
- the same replay and freshness expectations
- the same operator-visible semantics

Backend-specific realization details belong in detailed evidence only.

## Performance And Scaling Posture

### One Architecture Everywhere
RuneCode must use the same reviewed attestation architecture on constrained and scaled deployments.

Different deployments may vary in:
- cache population strategy
- mirror or prefetch strategy
- warm-pool behavior
- local storage sizing

They must not vary in:
- trust roots
- verification steps
- fallback behavior
- audit semantics

There is no constrained-device exception that permits TOFU for production or user-facing runtime use.

### Performance Constraints
- No optimization may bypass signature verification, verifier admissibility, measurement-profile evaluation, replay checks, or evidence persistence.
- Performance work must improve warm paths through caching and reuse of immutable verified state, not by weakening correctness.
- Restart-time reconstruction must derive authoritative posture from persisted evidence and cached verification results, not from stale in-memory state.

## Main Workstreams
- Attestation Evidence Model
- Trusted Verification Record Model
- Replay And Freshness Enforcement
- Launch, Verification, and Policy Integration
- Broker Projection, Audit, and TUI Posture
- Fixtures, Caching, and Cross-Platform Contract Stability

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.

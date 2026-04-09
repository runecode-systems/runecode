# Tasks

## Signed Object + Verifier Contract

- [ ] Standardize the signed-object contract around a semantic payload plus detached attestation/signature blocks over RFC 8785 JCS canonical payload bytes.
- [ ] Define how signed protocol families that currently carry `signatures` converge on that contract rather than relying on ambiguous omission rules.
- [ ] Make `ApprovalRequest` and `ApprovalDecision` signed protocol objects alongside manifests, receipts, provenance receipts, and audit events.
- [ ] Reserve a stable hook for step-up assurance evidence in approvals (`approval_assertion_hash` or equivalent) so a trusted approval authority can bind verified user cryptographic assertions into signed approval decisions without making the delivery channel authoritative.
- [ ] Define a first-class verifier record/descriptor model carrying at least:
  - `key_id`
  - `alg`
  - public key encoding and value
  - logical purpose
  - logical scope
  - owner principal reference
  - protection posture
  - creation metadata
  - current status
- [ ] Define deterministic public-key-derived `key_id` rules; key IDs must be topology-neutral and must not encode local filesystem paths, local usernames, keystore handles, or mutable trust state.
- [ ] Keep the signed-object and verifier model compatible with future multi-signature and quorum-approval expansion.

Parallelization: can be implemented in parallel with schema work; depends on stable signed object envelopes and key id formats.

## Scoped Key Authorities

- [ ] Replace generic "per-machine signing keypair" language with scope-aware logical authorities:
  - deployment-scoped `manifest_authority`
  - node-scoped `host_audit`
  - node- or deployment-scoped `audit_anchor`
  - store- or deployment-scoped `backup_integrity`
  - user-scoped `approval_authority`
  - session-scoped `isolate_session_identity`
  - publisher-scoped `image_signing`
- [ ] Allow multiple logical authorities to collapse onto one machine in MVP without collapsing their logical scope, audit identity, or policy rules.
- [ ] Keep boundary-visible authority and verifier identities topology-neutral so future remote and horizontally scaled deployments do not depend on local host identity.
- [ ] Per-isolate identity signing keypair:
  - used by isolates to sign isolate-attributed audit events
  - generated inside the isolate boundary
  - per-session ephemeral
  - bound to a specific run/session at handshake time and recorded in durable state
  - never generated or held by the launcher or broker
- [ ] Optional data-at-rest encryption key hierarchy (MVP posture-aware):
  - keys used for app-level encryption of sensitive local state when not relying solely on full-disk/workspace encryption
  - must fail closed by default if no secure storage is available
- [ ] Separate key namespace for image/toolchain signing (can be stubbed for MVP if image signing is not yet enforced).

Parallelization: can be implemented in parallel with schema work; depends on stable verifier records and authority-scope names.

## Key Storage + Posture Recording

- [ ] `secretsd` exposes the logical custody API for long-lived private keys and authoritative state-integrity keys; serving processes may be ephemeral while authority state remains durable.
- [ ] Prefer hardware-backed keys when available (TPM/Secure Enclave).
- [ ] Otherwise use OS key storage where possible.
- [ ] If neither hardware-backed nor OS key storage is available:
  - default behavior is fail closed (no silent plaintext-on-disk fallback)
  - allow an explicit, audited opt-in to passphrase-derived encryption for developer/portable setups
- [ ] Passphrase-derived encryption requirements (MVP):
  - KDF: Argon2id (RFC 9106) with stored parameters per ciphertext.
  - Default parameters (baseline): memory=64 MiB, iterations=3, parallelism=1, salt=16 bytes, key=32 bytes.
  - Passphrase policy: reject passphrases shorter than 16 characters; warn on 16-19; recommend 20+ (multi-word).
  - Never persist the passphrase; derived keys are kept in memory only for the minimum required window (best-effort zeroization).
- [ ] Define and record closed posture enums:
  - `key_protection_posture = hardware_backed | os_keystore | passphrase_wrapped | ephemeral_memory`
  - `identity_binding_posture = attested | tofu`
  - `presence_mode = none | os_confirmation | passphrase | hardware_touch`
- [ ] Unknown posture values fail closed.
- [ ] Degraded posture values must be surfaced in audit events and UI rather than remaining silent details.

Parallelization: can be implemented in parallel with secretsd work; coordinate on a shared KDF policy so “dev/portable mode” is consistent across the system.

## Approval Authority + User-Assurance Hooks

- [ ] Separate the approval model into:
  - policy approval
  - approval assurance level
  - presence mode
  - delivery channel
- [ ] Define `approval_assurance_level = none | session_authenticated | reauthenticated | hardware_backed`.
- [ ] Delivery channel (`local_tui`, `remote_tui`, messaging bridge, later integrations) is routing/advisory metadata only and is not the trust primitive.
- [ ] `ApprovalDecision` is signed by a trusted approval authority; when step-up assurance is required, the user authenticator signs a challenge/assertion that the approval authority verifies and binds into the decision.
- [ ] Replace broker-generated placeholder `ApprovalRequest` envelopes with real detached signatures from a broker-scoped authority or a distinct unsigned-pending contract; fail closed if a path would otherwise emit an `ed25519` envelope with stub signature bytes.
- [ ] Highest-assurance operations require a user-controlled cryptographic factor regardless of delivery channel; exact hard-floor categories are coordinated with `runecontext/changes/CHG-2026-007-2315-policy-engine-v0/`.
- [ ] Typed approval payloads must provide enough context for safe review, including:
  - scope
  - why approval is required
  - exact effect if approved
  - exact effect if denied or deferred
  - security-posture impact
  - blocked work
  - expiry
  - related hashes and artifacts
- [ ] Ensure the same approval contract supports local TUI, remote TUI, and message-driven delivery without changing trust semantics.
- [ ] Keep the cryptographic approval identity and assurance fields aligned with the broker local API approval list/get/resolve contract so delivery surfaces do not fork approval semantics.

Parallelization: can be implemented in parallel with audit anchoring; both rely on user-presence-gated signing.

## Sign/Verify Primitives + Verification Placement

- [ ] Standardize algorithms and hash/sign inputs:
  - signatures: Ed25519
  - hashing: SHA-256 over canonical bytes
  - canonicalization: RFC 8785 JCS (see schema spec)
- [ ] Define a minimal isolate key provisioning + pinning protocol (MVP):
  - isolate generates a per-session keypair at session start
  - isolate proves possession of the private key during a mutually authenticated session handshake
  - broker pins the public key to `{run_id, isolate_id, session_id}` and records it durably
- [ ] TOFU risk posture (MVP):
  - Treat the first provisioning handshake as TOFU and record it explicitly.
  - Record additional binding context in durable state + audit metadata:
    - `provisioning_mode = tofu`
    - isolate image digest (and signer, if available)
    - active manifest hash
    - a launcher-generated `session_nonce` (unique per session)
    - a `handshake_transcript_hash` (hash of canonical handshake messages)
  - Verifiers and UI must surface TOFU as a degraded posture (not a silent detail).
  - later attestation work that upgrades TOFU to an attested binding lives in `runecontext/changes/CHG-2026-030-98b8-isolate-attestation-v0/`.
- [ ] Trusted Go verification is authoritative via shared library/contracts used by narrow trusted daemons:
  - broker verifies signed manifests, signed approvals, and bound verifier references before admitting capability effects
  - launcher verifies isolate handshake proof-of-possession and session-key bindings
  - auditd verifies audit events, receipts, and anchor evidence
  - secretsd verifies sign-request preconditions before using long-lived private keys
- [ ] Runner-side TypeScript verification remains fixture/parity/supporting behavior only and must not become an authoritative trust-admission path.
- [ ] Algorithm agility (MVP requirement):
  - Every signature envelope includes `{alg, key_id}`.
  - Schema/versioning rules define how additional algorithms are introduced (schema bump; verifier support).

Parallelization: can be implemented in parallel with protocol schema work; the canonicalization interface (`JCS -> bytes -> hash/sign`) must be agreed early.

## Rotation + Revocation (Minimal)

- [ ] Define how keys rotate, how active and historical verifier records are discovered, and how revocation is represented and checked.
- [ ] Record rotations and revocations as signed audit events and/or receipts.
- [ ] Rotation/revocation must not rewrite history:
  - old signatures remain cryptographically verifiable
  - revocation changes future acceptance posture
  - compromise metadata such as `suspect_since` may be recorded without changing historical bytes

Parallelization: can be implemented in parallel with audit log verify; rotation/revocation events are additional audit event types.

## Authoritative State Integrity

- [ ] Protect/export/import canonical RuneCode trusted state using a dedicated `backup_integrity` authority managed via the same logical custody model rather than ad-hoc shared-secret semantics.
- [ ] Cover canonical trusted state including:
  - verifier records
  - signed manifests
  - approval artifacts
  - audit evidence
  - artifact integrity/state metadata
  - trusted backups and exports
- [ ] Treat runner-internal non-canonical orchestration state, including LangGraph checkpoints/state, as outside this trust root unless exported into canonical protocol objects.
- [ ] Ensure authoritative state integrity still works when the serving process or UI surface is ephemeral.

Parallelization: can be implemented in parallel with trusted-state persistence work; it depends on stable verifier records and detached attestation contracts.

## Acceptance Criteria

- [ ] Capability manifests, approval requests, approval decisions, and isolate-attributed audit evidence are verifiably signed using the canonical payload contract.
- [ ] Launcher/broker cannot forge isolate-attributed events without the isolate private key.
- [ ] Logical authorities remain scope-distinct even when they are physically colocated on one machine in MVP.
- [ ] Trusted Go verification is authoritative and runner-side verification remains non-authoritative.
- [ ] The system records key protection posture and identity-binding posture and treats degraded posture as an auditable condition.
- [ ] Highest-assurance approvals cannot be satisfied by delivery-channel clicks alone.
- [ ] Authoritative backup/export integrity covers canonical trusted state and is distinct from runner-internal non-canonical state.
- [ ] Verification fails closed and produces clear error artifacts.

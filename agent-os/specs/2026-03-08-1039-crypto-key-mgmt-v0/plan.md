# Crypto / Key Management v0

User-visible outcome: RuneCode can sign and verify manifests and audit events using a clear key hierarchy, while recording the host’s key protection posture.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-crypto-key-mgmt-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Define MVP Key Hierarchy

- Per-machine signing keypair:
  - Used to sign run/stage capability manifests and trusted host-component events.
  - The private key must be protected by hardware-backed or OS-provided key storage when available.
- Per-isolate identity signing keypair:
  - Used by isolates to sign isolate-attributed audit events.
  - Must be generated inside the isolate boundary; the launcher/broker must never generate or hold isolate private keys.
  - The isolate public key is bound to a specific run/session at handshake time and recorded in durable local state.
- Optional data-at-rest encryption key hierarchy (MVP posture-aware):
  - Keys used for app-level encryption of sensitive local state when not relying solely on full-disk/workspace encryption.
  - Must fail closed by default if no secure storage is available.
- Separate key namespace for image/toolchain signing (can be stubbed for MVP if image signing is not yet enforced).

Parallelization: can be implemented in parallel with schema work; depends on stable signed object envelopes and key id formats.

## Task 3: Key Storage + Posture Recording

- Prefer hardware-backed keys when available (TPM/Secure Enclave).
- Otherwise use OS key storage where possible.
- If neither hardware-backed nor OS key storage is available:
  - default behavior is fail closed (no silent plaintext-on-disk fallback)
  - allow an explicit, audited opt-in to passphrase-derived encryption for developer/portable setups
- Passphrase-derived encryption requirements (MVP):
  - KDF: Argon2id (RFC 9106) with stored parameters per ciphertext.
  - Default parameters (baseline): memory=64 MiB, iterations=3, parallelism=1, salt=16 bytes, key=32 bytes.
  - Passphrase policy: reject passphrases shorter than 16 characters; warn on 16-19; recommend 20+ (multi-word).
  - Never persist the passphrase; derived keys are kept in memory only for the minimum required window (best-effort zeroization).
- Record and surface key protection level ("posture") in audit events and UI.

Parallelization: can be implemented in parallel with secretsd work; coordinate on a shared KDF policy so “dev/portable mode” is consistent across the system.

## Task 4: User-Presence Approval Hook (MVP Baseline)

- Add a deterministic “requires user presence” step for signing new capability manifests.
- MVP implementation can be OS-confirmation / passphrase-based, with a future hardware key tap path.

Parallelization: can be implemented in parallel with audit anchoring; both rely on user-presence-gated signing.

## Task 5: Sign/Verify Primitives

- Standardize algorithms and hash/sign inputs:
  - signatures: Ed25519
  - hashing: SHA-256 over canonical bytes
  - canonicalization: RFC 8785 JCS (see schema spec)
- Define a minimal isolate key provisioning + pinning protocol (MVP):
  - isolate generates keypair on first boot/session start
  - isolate sends public key as part of a mutually authenticated session handshake
  - broker pins the public key to `{run_id, isolate_id, session_id}` and records it durably
- TOFU risk posture (MVP):
  - Treat the first provisioning handshake as TOFU and record it explicitly.
  - Record additional binding context in durable state + audit metadata:
    - `provisioning_mode = tofu`
    - isolate image digest (and signer, if available)
    - a launcher-generated `session_nonce` (unique per session)
    - a `handshake_transcript_hash` (hash of canonical handshake messages)
  - Verifiers and UI must surface TOFU as a degraded posture (not a silent detail).
  - post-MVP: measured boot / attestation can upgrade TOFU to an attested binding without changing the audit/event format.

- Algorithm agility (MVP requirement):
  - Every signature envelope includes `{alg, key_id}`.
  - Schema/versioning rules define how additional algorithms are introduced (schema bump; verifier support).
- Implement verification in both Go and TS where required.

Parallelization: can be implemented in parallel with protocol schema work; the canonicalization interface (`JCS -> bytes -> hash/sign`) must be agreed early.

## Task 6: Rotation + Revocation (Minimal)

- Define how keys rotate and how revocation is represented and checked.
- Record rotations/revocations as audit events.
- Rotation/revocation must not rewrite history: old signatures remain verifiable.

Parallelization: can be implemented in parallel with audit log verify; rotation/revocation events are additional audit event types.

## Acceptance Criteria

- Capability manifests and isolate-attributed audit events are verifiably signed.
- Launcher/broker cannot forge isolate-attributed events without the isolate private key.
- The system records key protection posture and treats degraded posture as an auditable condition.
- Verification fails closed and produces clear error artifacts.

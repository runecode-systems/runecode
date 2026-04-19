# Design

## Overview
Define isolate attestation evidence and verification that upgrades MVP TOFU provisioning to an attestable posture when required.

## Key Decisions
- MVP TOFU session metadata remains the compatibility baseline.
- Attestation upgrades the existing `IsolateSessionBinding` model from `CHG-2026-009-1672-launcher-microvm-backend-v0`; it does not replace the existing session-binding contract with a different identity model.
- Attestation adds stronger evidence; it does not replace the need for explicit binding to session identity, image identity, and provisioning evidence.
- Verifier, policy, and TUI surfaces must expose provisioning posture explicitly.
- Invalid or replayed attestation evidence fails closed when an attested posture is required.
- Attestation evidence should bind to the same `session_nonce`, `handshake_transcript_hash`, runtime image descriptor identity, and concrete boot component digests established by the microVM launch flow.
- The attestation upgrade path must preserve compatibility with the `isolate_session_started` and `isolate_session_bound` audit event families.
- Attested runtime identity remains distinct from validated project-substrate snapshot identity; later evidence may bind both when relevant, but attestation must not redefine project-context identity.

## Main Workstreams
- Attestation Evidence Model
- Launch, Verification, and Policy Integration
- TUI + Audit Posture
- Fixtures + Cross-Platform Considerations

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.

# Crypto / Key Management v0 — Shaping Notes

## Scope

Establish the minimum cryptographic root-of-trust for signing manifests and audit events in MVP.

## Decisions

- Isolates sign their own audit events; the control plane must not be able to forge isolate-attributed events.
- Manifest signing requires explicit user presence.
- Isolate identity private keys are generated and stored inside the isolate boundary; the launcher/broker must never possess isolate private keys.
- If secure key storage is unavailable (hardware/OS keystore), the system must fail closed by default (no silent plaintext fallback).
- If passphrase-derived encryption is explicitly opted into (dev/portable mode), the KDF + passphrase policy is specified and audited (Argon2id; minimum strength requirements).
- Isolate key provisioning is TOFU for MVP; provisioning mode and handshake binding context are recorded and surfaced as a degraded posture.
- Signature envelopes include `{alg, key_id}` to keep algorithm agility feasible.

## Context

- Visuals: None.
- References: `agent-os/product/tech-stack.md`
- Product alignment: Supports cryptographic provenance and tamper-evident audit.

## Standards Applied

- None yet.

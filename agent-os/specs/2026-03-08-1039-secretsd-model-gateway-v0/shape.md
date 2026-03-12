# Secretsd + Model-Gateway v0 — Shaping Notes

## Scope

Implement secrets storage/lease issuance and a dedicated model-gateway that centralizes third-party model egress.

## Decisions

- Third-party model usage is explicit opt-in; deny by default.
- Model traffic goes only through model-gateway; workspace roles remain offline.
- Only `secretsd` stores long-lived secrets; other daemons/components must not persist secret values (leases only).
- Secret values are never accepted or delivered via environment variables; use stdin/file-descriptor onboarding and brokered lease IPC.
- Secrets storage fails closed by default if secure key storage is unavailable (no silent plaintext-on-disk fallback).
  - If passphrase-derived encryption is explicitly opted into (dev/portable), KDF + passphrase policy is specified and audited (Argon2id; minimum strength requirements).
- Model gateway egress is hardened against SSRF/DNS rebinding and enforces TLS-only provider connections.
  - SSRF/DNS rebinding protections explicitly include IPv6 private/link-local/loopback/reserved ranges and IPv4-mapped IPv6.
- Model-gateway uses a typed `LLMRequest`/`LLMResponse` boundary; inputs reference artifacts by hash (no freeform prompt blobs).
- Model-gateway fetches artifact bytes by hash (via broker-mediated CAS access) and fails closed on disallowed data classes.
- Model-gateway is implemented in Go for MVP to minimize TCB; provider request shaping stays inside the Go gateway.
- Official provider SDKs (JS) are used only for fixture generation and drift detection; they are not in the production egress path.
- Streaming and tool calling are supported only within the typed boundary; tool calls remain untrusted proposals.
- MVP default for model egress is `spec_text` only; allowing `diffs` or `approved_file_excerpts` is an explicit, auditable opt-in.
- Post-MVP: add `bridge` providers for officially supported user-installed local runtimes (subscription access) behind model-gateway, with an explicit LLM-only mode and a compatibility probe policy.
  - Permit untested vendor versions only if the probe passes, with explicit user acknowledgment recorded in audit.

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-policy-engine-v0/`
- Product alignment: Prevents any single component from combining workspace access + public egress + long-lived secrets.

## Standards Applied

- None yet.

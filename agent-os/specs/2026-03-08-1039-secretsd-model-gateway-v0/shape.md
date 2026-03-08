# Secretsd + Model-Gateway v0 — Shaping Notes

## Scope

Implement secrets storage/lease issuance and a dedicated model-gateway that centralizes third-party model egress.

## Decisions

- Third-party model usage is explicit opt-in; deny by default.
- Model traffic goes only through model-gateway; workspace roles remain offline.
- Secrets storage fails closed by default if secure key storage is unavailable (no silent plaintext-on-disk fallback).
- Model gateway egress is hardened against SSRF/DNS rebinding and enforces TLS-only provider connections.

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-policy-engine-v0/`
- Product alignment: Prevents any single component from combining workspace access + public egress + long-lived secrets.

## Standards Applied

- None yet.

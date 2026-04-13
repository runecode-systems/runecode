# Design

## Overview
Implement the network egress boundary for model traffic as a dedicated gateway feature.

## Key Decisions
- Model traffic is explicit opt-in and deny-by-default.
- Gateway has no workspace access and uses leases only.
- Typed boundaries are required for machine-consumed traffic.
- SSRF/DNS-rebinding protections and TLS enforcement are mandatory.
- Model egress should use the shared typed gateway destination/allowlist model so endpoint identity, allowed operations, and allowed egress data classes stay aligned with the broader policy foundation.

## Canonical Model Boundary

- `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` remain the canonical typed request, response, and stream families for model traffic.
- Provider adapters and bridge/runtime integrations live below those typed contracts unless a later typed extension is needed for policy, audit, or replay semantics.
- Tool calls remain untrusted proposals and never direct execution instructions.
- Stream handling keeps stable stream identity, monotonic sequencing, and exactly one terminal event.

## Destination Identity And Gateway Operations

- `destination_ref` uses one canonical `host[:port][/path]` form with no scheme, query, fragment, or embedded credentials.
- Gateway operation identity should move from ad hoc strings to one closed shared registry.
- The model-gateway should initially rely on a small operation set that distinguishes scope-change operations from request-execution operations.
- Request-execution operations should require `payload_hash`, and for model traffic that hash should bind to the canonical `LLMRequest` object hash.

## Role Separation

- `model-gateway` remains separate from `auth-gateway`.
- Model traffic may consume short-lived leased material but must not perform login, token exchange, or refresh in place.
- The gateway must not gain workspace access, long-lived secret custody, or a daemon-private user API as part of downstream provider integration.

## Shared Hardening Surface

- The gateway should reuse the shared typed destination and allowlist model for canonical destination identity.
- TLS enforcement, private-range blocking, DNS rebinding protection, redirect posture, timeout handling, and response-size limits should be shared hardening behavior rather than provider-specific conventions.
- Provider-specific adapters should not redefine those transport-hardening rules.

## Quotas And Usage Accounting

- Quota handling should use one trusted abstraction that can represent:
  - request limits
  - input-token limits
  - output-token limits
  - streamed-byte limits
  - concurrency caps
  - spend ceilings
  - request-entitlement products such as premium requests
- Provider headers and provider-specific usage signals are inputs to trusted quota handling rather than the sole source of truth.
- Quotas should be enforceable preflight, mid-stream where appropriate, and after response reconciliation from actual usage.

## Operator Posture

- Daemon-local health remains a supervision surface.
- Broker should project model-gateway posture and readiness through broker-visible subsystem status rather than a separate daemon-specific public API.

## Main Workstreams
- Gateway role boundary and egress controls.
- Canonical typed model schemas and policy integration.
- Destination and operation identity.
- Data-class filtering and redaction enforcement.
- Quotas, request binding, and audit event coverage.

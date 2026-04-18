# Tasks

## Official OAuth Client Registration

- [ ] RuneCode maintains its own official OAuth client registration for Sign in with ChatGPT.
- [ ] Use PKCE + `offline_access` to obtain refresh capability.
- [ ] Provide two official login paths:
  - browser-based login for normal desktop use
  - device-code style flow if needed for headless or constrained environments
- [ ] Fail closed on OAuth state/redirect mismatches.
- [ ] RuneCode must not rely on vendor-internal OAuth clients or piggyback registrations.
- [ ] Surface login and account-linking through broker-mediated setup flows exposed by guided TUI and straightforward CLI clients rather than runtime-local setup.

## Auth-Gateway Role (Auth Egress Only)

- [ ] Introduce a dedicated `auth-gateway` role.
- [ ] Store refresh token material and rotation metadata only in `secretsd`.
- [ ] Issue short-lived, scope-bound leases for `idToken` and `accessToken` (or equivalent) to `model-gateway`.
- [ ] Disallow environment-variable secret injection.
- [ ] Keep any manual token-entry fallback limited to trusted interactive broker-mediated prompts rather than flags or environment variables.

## Model-Gateway Bridge via Codex App-Server

- [ ] Policy constraint: RuneCode does not ship, bundle, or redistribute vendor CLIs or proprietary runtimes.
- [ ] Run the official Codex app-server runtime under the `model-gateway` role as a local bridge (stdio JSON-RPC; no listening ports by default).
- [ ] Runtime compatibility policy (post-MVP):
  - use the shared bridge/runtime protocol contract
  - keep compatibility probe-driven and fail closed on unsupported runtime versions
- [ ] Use Codex external token mode (`chatgptAuthTokens`).
- [ ] Enforce LLM-only capability scoping.
- [ ] Default to ephemeral sessions.
- [ ] Prefer protocol-level contract tests over HTTP wire fixtures.

## Policy + Audit Integration

- [ ] Default deny: enabling this provider is an explicit signed-manifest opt-in and must be surfaced as a high-risk approval.
- [ ] Audit requirements:
  - auth login/refresh lifecycle events
  - runtime identity/version discovery
  - model egress destination, bytes, timing, and outcome
- [ ] Enforce model egress data-class policy at the RuneCode `LLMRequest` boundary.

## Acceptance Criteria

- [ ] GPT model access uses ChatGPT subscription quotas via official OAuth.
- [ ] No environment-variable secret injection is used.
- [ ] No second secrets store exists: only `secretsd` persists long-lived auth material.
- [ ] Workspace roles remain offline; all model egress remains behind `model-gateway`.

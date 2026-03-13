# Secretsd + Model-Gateway v0

User-visible outcome: third-party model access is possible only via an explicitly allowed gateway role, using short-lived scoped secrets leases, with boundary redaction and complete auditing.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-secretsd-model-gateway-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Secretsd MVP Interface

- Provide a minimal secrets daemon that:
  - stores long-lived secrets at rest (prefer hardware-backed / OS key storage where available)
  - fails closed by default if secure key storage is unavailable (no silent plaintext fallback)
  - allows an explicit, audited opt-in to passphrase-derived encryption for developer/portable setups
    - passphrase-derived encryption requirements (MVP):
      - KDF: Argon2id (RFC 9106) with stored parameters per ciphertext
      - default parameters (baseline): memory=64 MiB, iterations=3, parallelism=1, salt=16 bytes, key=32 bytes
      - passphrase policy: reject < 16 chars; warn on 16-19; recommend 20+
      - never persist the passphrase; derived keys live in memory only as needed (best-effort zeroization)
  - issues short-lived, scope-bound leases only as allowed by the signed manifest
  - defines lease TTL bounds, renewal rules, and revocation semantics
  - records every lease as an audit event (without logging raw secrets)
- Define a safe secret onboarding/import flow (MVP):
  - secrets are provided via stdin or a file descriptor (never CLI args or environment variables)
  - only secret metadata/IDs are logged/audited (never secret values)

- Expose local-only health/readiness signals for `secretsd` and `model-gateway` (consumable via the broker local API).
- Emit minimal operational metrics (local-only): request/lease counts, denials, and latency histograms (no secret values).

Parallelization: can be implemented in parallel with crypto key management; coordinate on shared “key posture” and passphrase/KDF policy.

## Task 3: Model-Gateway Role

- Implement a dedicated gateway role with:
  - network egress allowlist (model provider domains only)
  - no workspace access
  - provider keys obtained only via secrets leases
  - schema-validated request/response boundary
  - RuneCode-native typed model requests/responses (no freeform prompt blobs cross the boundary; inputs reference artifacts by hash)
  - Support streaming responses within the typed boundary.
  - Support tool calling only as typed proposal objects; never execute tools from the gateway.
  - Require structured JSON outputs for any machine-consumed output.
- Implementation constraint (MVP): keep `model-gateway` implemented in Go to minimize the trusted computing base (TCB).
  - Avoid introducing npm supply-chain dependencies into a high-risk egress boundary.
  - Do not add a runtime Node "request builder" isolate for provider payload shaping.
- Model-gateway must fetch artifact bytes by hash (via broker-mediated artifact store APIs) and assemble provider requests only from allowlisted artifact data classes.
- Harden egress controls against SSRF and DNS rebinding:
  - resolve and validate destinations (block private/link-local/loopback/reserved ranges for both IPv4 and IPv6, including IPv4-mapped IPv6)
  - restrict redirects (or disable by default); if enabled, validate every hop and never follow redirects to out-of-policy hosts
  - require TLS with certificate validation and SNI matching
  - apply strict timeouts and response size limits
  - Define streaming-specific limits (chunk sizes, total streamed bytes, idle timeouts) so streaming cannot bypass size/timeout controls.

Parallelization: can be implemented in parallel with broker + policy engine once the `LLMRequest`/`LLMResponse` schemas and data-class flow rules are stable.

## Task 3b: Provider Adapters + Drift Detection (MVP)

- Translate RuneCode-native `LLMRequest` into provider-specific HTTP payloads inside the Go model-gateway.
- MVP remote API-key provider coverage includes OpenAI, Anthropic, and Google model APIs.
- Do not depend on LangChain provider packages for production egress payload shaping.
- Keep official provider SDK packages out of the production egress path.
  - Use them only for test/fixture generation to detect upstream request-shape drift.
- Golden fixture generators (MVP) use official provider SDKs only:
  - OpenAI: `openai`
  - Anthropic: `@anthropic-ai/sdk`
  - Google: `@google/genai`
- Do not use generic abstraction packages (for example AI SDK or LangChain) as the golden fixture source.
  - They may be useful for future app-layer compatibility checks, but they are out of scope for MVP drift detection because they normalize provider differences that the fixture lane must preserve.
- Use `models.dev` only for provider/model catalog and capability metadata (IDs, package mapping, model capabilities, limits/pricing hints).
  - Do not use `models.dev` as a source of truth for raw request bodies, headers, or provider wire semantics.
- Commit non-sensitive "golden" fixtures and fail CI on drift.
  - Canonicalize away volatile fields (e.g., auth headers, timestamps, content-length) while remaining strict about semantically meaningful fields.
  - Add Go adapter conformance tests that load the same fixtures and assert `LLMRequest -> provider HTTP` matches after canonicalization.
  - Provide a Node `fixturegen` tool (non-production) that regenerates fixtures using the official provider SDKs.
    - capture requests via SDK-supported custom `fetch` / proxy hooks into a local sink rather than hitting live provider endpoints
    - disable retries and other hidden transport behaviors that can blur request capture
    - pin any provider API versions/headers that materially affect request shape (for example Google `apiVersion`)
  - Run `fixturegen` locally via the Nix dev shell to update fixtures; commit the results.
  - Pin SDK versions in lockfiles; dependency upgrades require explicit fixture regeneration + review.
  - Use automated dependency updates so fixture drift is detected at upgrade time and requires explicit approval.
- Other remote API providers:
  - prefer the provider's official SDK when one exists and is stable enough for request capture
  - if a provider exposes an OpenAI-compatible API but lacks a good official SDK, treat it as a separate lower-confidence compatibility lane
    - label its fixtures as compatibility fixtures rather than first-class provider goldens
    - do not let OpenAI-compatible abstractions redefine the canonical OpenAI/Anthropic/Google fixture lanes
  - promoting a new provider to a first-class golden lane requires explicit review of its auth model, request-shape stability, and fixture capture method
- Add CI coverage (GitHub Actions) so upgrades fail closed when fixtures drift.

Parallelization: can be implemented in parallel with protocol schema work (fixtures) and with broker limits; avoid conflicts by agreeing on canonicalization and fixture locations early.

## Task 3c: Bridge Providers (Post-MVP)

- Support a second provider integration mode for subscription-backed and local runtimes:
  - policy constraint: RuneCode does not ship/bundle/redistribute vendor CLIs or proprietary runtimes; integrate with user-installed official runtimes
  - `http` providers (MVP): model-gateway translates `LLMRequest -> provider HTTP` directly.
- `bridge` providers (post-MVP): model-gateway translates `LLMRequest -> local RPC` and the local runtime performs upstream network calls.
  - prefer spawned child processes over stdio (no listening ports)
  - require runtime identity + version discovery and per-request version logging
    - compatibility policy: do not require RuneCode updates for every vendor release
      - define a "tested range" plus a compatibility probe (contract tests / schema validation / required feature flags)
      - allow newer versions if the probe passes; otherwise fail closed with a clear remediation (downgrade vendor runtime or upgrade RuneCode)
      - if running an untested-but-probe-passing version, require an explicit acknowledgment surfaced in TUI and recorded in audit metadata
  - enforce an explicit "LLM-only" mode (deny tool execution and file operations)
  - run with isolated `HOME`/tool dirs pointing at an allowlisted provider sandbox directory
    - disable core dumps; treat the sandbox dir as hostile and prevent token spills (no env/argv injection; controlled temp dirs)
    - restrict child process spawning (deny-by-default; allowlist only if required and audited)
  - default to ephemeral sessions (no persisted conversation state unless enabled by manifest+policy)
  - prefer protocol-level contract tests (RPC fixtures) over HTTP wire fixtures

## Task 4: Data-Class Policy for Model Egress

- Default deny for third-party model usage.
- When explicitly opted in, allow only specific data classes (MVP baseline: `spec_text` only).
- Expanding allowed egress classes beyond `spec_text` (e.g., `diffs`, `approved_file_excerpts`) requires an explicit signed manifest opt-in and must be surfaced as a high-risk approval in the `moderate` profile.
- Unapproved excerpts (`unapproved_file_excerpts`) are never eligible for model egress.
- Enforce redaction at the boundary structurally:
  - use schema field classification metadata (`secret` fields are rejected/stripped)
  - prefer allowlists of permitted fields/classes over heuristic redaction

Parallelization: can be implemented in parallel with artifact store flow matrix work; it depends on stable data-class taxonomy.

## Task 5: Audit + Quotas

- Log outbound requests (destination, bytes, timing) as audit events.
- Enforce basic quotas (requests/bytes/time) for the gateway role.

Parallelization: can be implemented in parallel with audit log verify and broker rate limits; align quota counters with audit event fields for observability.

## Acceptance Criteria

- No other role can directly reach the public internet for model traffic.
- Workspace roles have zero direct network egress.
- Network egress is limited to explicit gateway roles; in MVP the only egress-capable gateway role is `model-gateway`.
- Secrets are never persisted in the launcher/broker/scheduler; only leases are used.
- Model-gateway blocks SSRF/DNS rebinding classes of attacks (private IPs, unsafe redirects) by default.
- Opt-in model egress is explicit, enforceable, and auditable.
- OpenAI, Anthropic, and Google provider adapters have official-SDK-generated golden fixtures checked into the repo and validated by Go conformance tests.
- `models.dev` is used only for provider/model catalog and capability metadata; provider wire-shape truth comes from official SDK fixture generation.

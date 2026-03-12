# GitHub Copilot Subscription Provider (Official Runtime Bridge) — Post-MVP

User-visible outcome: RuneCode can access Copilot-backed models using a user's GitHub Copilot subscription via an officially supported local runtime, while preserving strict isolation (no workspace access in egress roles, `secretsd` as the only long-lived secret store, and complete auditability).

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-11-1921-github-copilot-subscription-provider-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Official Runtime + Protocol Selection

- Policy constraint: RuneCode does not ship/bundle/redistribute vendor CLIs or proprietary runtimes.
  - Integrate only with an officially supported, user-installed Copilot runtime.
- Use an officially supported Copilot local runtime (installed/managed by the user/admin).
- Select the bridge protocol surface that supports strict permission control and least privilege:
  - ACP over stdio, or
  - official SDK/JSON-RPC mode (if it provides equivalent controls)
- Prefer stdio spawning over listening ports.

Runtime compatibility policy (post-MVP):
- Goal: do not require a RuneCode update for every vendor runtime release.
- Define a "tested range" of runtime versions plus a compatibility probe:
  - probe validates required RPC methods, schema shapes, and "LLM-only" invariants
  - if the probe passes, the runtime is permitted even if the exact version is untested
  - if the probe fails, fail closed with a clear remediation (downgrade vendor runtime or upgrade RuneCode)
- For untested-but-probe-passing versions:
  - require an explicit user acknowledgment surfaced in TUI
  - record the runtime identity/version and "untested" posture in audit metadata

Parallelization: can be designed in parallel with the `model-gateway` bridge envelope work in `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`.

## Task 3: Auth Model (No Env Vars, No Second Store)

- Introduce a dedicated `auth-gateway` flow for GitHub auth when required.
- If OAuth/device-code is required for this provider, RuneCode maintains its own official OAuth client registration.
- Store long-lived auth material only in `secretsd`.
- Disallow environment-variable token injection.
  - Define a token delivery mechanism that does not use env vars (e.g., stdin/FD, or a runtime-supported config file in a secretsd-managed directory).
- If the runtime requires persisted auth state, it must be stored only in a secretsd-managed encrypted directory and treated as secret material.

Parallelization: can be implemented in parallel with `auth-gateway` role work and `secretsd`; it depends on stable lease/token delivery schemas.

## Task 4: Model-Gateway Bridge (LLM-Only)

- Run the runtime under `model-gateway` with:
  - no workspace mounts; empty/scratch `cwd`
  - isolated `HOME`/tool dirs pointing at an allowlisted provider sandbox directory
    - sandbox enforcement requirements:
      - disable core dumps
      - restrict child process spawning (deny-by-default; allowlist only if required and audited)
      - treat the sandbox directory as hostile for secrets (no env/argv injection; controlled temp dirs)
  - strict deny-by-default tool/permission requests (LLM-only mode)
  - treat any attempt to exec/write/read workspace as a policy violation
  - schema-validated structured outputs only for machine-consumed actions
- Enforce model egress data-class policy at the RuneCode `LLMRequest` boundary.
- Default to ephemeral sessions.
  - do not persist conversation state unless explicitly enabled by signed manifest + policy
  - if the runtime requires local state, it must be stored only in a secretsd-managed encrypted directory
- Prefer protocol-level contract tests over HTTP wire fixtures.
  - add RPC request/response fixture tests and stable error taxonomy mapping

Parallelization: can be prototyped in parallel with core `model-gateway` work; it depends on the shared bridge envelope/error taxonomy.

## Task 5: Policy + Audit Integration

- Default deny: enabling this provider is an explicit signed-manifest opt-in and must be surfaced as a high-risk approval.
- Audit requirements:
  - auth events: login start/completed/cancelled, token lease issuance/renewal/revocation
  - model events: provider/model identifiers, bytes, timing, and outcome (without logging secret values)

Parallelization: can be implemented in parallel with policy engine + audit event model work once reason codes and gateway audit schemas are stable.

## Acceptance Criteria

- Copilot subscription model access is possible via official mechanisms.
- No environment-variable secret injection is used.
- No second secrets store exists: only `secretsd` persists long-lived auth material.
- Workspace roles remain offline; all model egress remains behind `model-gateway`.

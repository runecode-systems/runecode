# Workflow Runner + Workspace Roles + Deterministic Gates v0 — Shaping Notes

## Scope

Build the end-to-end workflow engine and offline workspace execution roles, with deterministic gates and evidence artifacts.

## Decisions

- The scheduler is treated as untrusted; the launcher/policy is the enforcement point.
- LangGraph is an internal implementation detail of the untrusted runner.
  - Stable interfaces are RuneCode schemas + broker/local API, not LangGraph internals.
  - This keeps the door open to replace LangGraph later without changing boundaries.
- Avoid LangChain "agents" / Deep Agents as the core runtime.
  - Keep orchestration typed and step-based so capability boundaries remain deterministic and auditable.
- The workflow runner is distributed as a Node SEA (single executable) built from a bundled CommonJS script.
  - SEA is packaging (not a sandbox) and does not change the runner's trust level.
  - SEA config ignores `NODE_OPTIONS` (set `execArgvExtension: "none"`) to prevent environment-driven runtime option injection.
- The runner has no public network egress (it is not a gateway role).
- Workspace roles are offline; any public egress is only via dedicated gateway roles (model inference via model-gateway).
- Runner persistence stores control-plane state only (IDs/hashes/approvals); it must never store raw workspace/code or secrets.
- "Shared memory" (if any) is a rebuildable, ephemeral accelerator keyed by `(repo, commitSHA)`; raw content remains in the CAS.
- Pause/resume is implemented via a persisted run state machine (durable state), not in-memory orchestration.
- Approval requests/decisions are hash-bound and time-bounded (TTL/expiry); stale approvals must be re-requested.
- Gate failure semantics are explicit (fail/abort, retry, and any override requires recorded approval).
- MVP uses a "moderate" approval profile: approvals are checkpoint-style (stage sign-off and explicit posture changes), not per-action.
- The workflow produces verifiable evidence artifacts (including `audit_verification_report`), not just human-readable logs.

- Concurrency is locked per workspace (one active run per workspace by default); parallel runs across distinct workspaces are allowed.

- SEA feasibility is validated early; if SEA bundling is blocked, a pinned-Node + bundled-JS fallback may ship for early alpha without changing the trust boundary.

## Context

- Visuals: None.
- References: `agent-os/product/tech-stack.md`
- Product alignment: Spec-first, least-privilege automation with auditable evidence.

## Standards Applied

- None yet.

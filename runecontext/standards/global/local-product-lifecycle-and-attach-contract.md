---
schema_version: 1
id: global/local-product-lifecycle-and-attach-contract
title: Local Product Lifecycle And Attach Contract
status: active
suggested_context_bundles:
    - project-core
    - go-control-plane
---

# Local Product Lifecycle And Attach Contract

When RuneCode presents as one local attachable product rather than manual component assembly:

- The canonical user-facing lifecycle entrypoint is `runecode`; low-level binaries such as `runecode-broker` and `runecode-tui` remain plumbing, admin, or dev surfaces rather than the normal semantic source of product lifecycle behavior.
- Bind one local RuneCode product instance to one authoritative repository root; derive product instance identity from that repo root rather than from socket paths, runtime directories, local usernames, or other host-local mechanics.
- Keep pidfiles, runtime directories, socket files, and similar local bootstrap artifacts advisory-only; they may help recovery and supervision but must not become authoritative lifecycle truth.
- Resolve authoritative repository scope before product-instance attachment or startup, and validate that any reachable broker matches that repo-scoped product identity.
- Keep public lifecycle semantics topology-neutral: attach, start, status, stop, and restart target the logical repo-scoped product instance rather than the current daemon or process graph used on one platform.
- Treat broker-owned typed product lifecycle posture as the canonical operator-facing attach contract; do not infer attachability, lifecycle generation, or degraded/blocked reason codes from readiness summaries, version surfaces, transport success, or bootstrap-local heuristics.
- Keep attachability and normal managed operation as distinct concepts; healthy broker attach may remain available in diagnostics/remediation-only mode even when current repository project-substrate posture blocks normal managed execution.
- `runecode status` and equivalent lifecycle inspection flows must be non-starting; when no live broker is reachable they may report only that private bootstrap-local fact instead of silently creating a new product instance.
- Keep session object lifecycle, projected work posture, and client attachment state separate; closing or reopening a client must not become canonical lifecycle truth for sessions or runs.

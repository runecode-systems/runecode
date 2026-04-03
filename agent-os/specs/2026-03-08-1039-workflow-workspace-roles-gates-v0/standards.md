# Standards for Workflow Runner + Workspace Roles + Deterministic Gates v0

These standards apply to implementation work produced from this spec.

## Trust Boundary

- `runecontext/standards/security/trust-boundary-interfaces.md`
- `runecontext/standards/security/trust-boundary-layered-enforcement.md`
- `runecontext/standards/security/trust-boundary-change-checklist.md`
- `runecontext/standards/security/runner-boundary-check.md`

## Runner JavaScript/Node

- `runecontext/standards/javascript/node-engine-enforcement.md`
- `runecontext/standards/javascript/npm-ci-lockfile.md`
- `runecontext/standards/javascript/runner-node-test-runner.md`
- `runecontext/standards/javascript/runner-typescript-as-lint.md`

## Runner Distribution (Node SEA)

- The workflow runner is packaged as a Node SEA (single executable) for release/runtime distribution.
- SEA is packaging, not a security boundary; the runner remains untrusted at runtime.
- SEA config must ignore `NODE_OPTIONS` (set `execArgvExtension: "none"`) so environment variables cannot silently extend Node runtime flags.
- Bundle the runner into a single injected CommonJS script; do not depend on runtime `node_modules` resolution.

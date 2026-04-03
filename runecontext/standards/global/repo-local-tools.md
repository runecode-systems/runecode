---
schema_version: 1
id: global/repo-local-tools
title: Repo-Local Helper Tools
status: active
suggested_context_bundles:
    - go-control-plane
    - ci-tooling
aliases:
    - agent-os/standards/global/repo-local-tools
---

# Repo-Local Helper Tools

- Prefer small repo-local helper programs over shell pipelines (portability, Windows)
- Keep tools deterministic and small; prefer stdlib-only/minimal deps
- Invoke tools via language runtimes (example: `go run ./tools/<tool>`)
- Place tools under `tools/<tool>/...` (example: `tools/gofmtcheck/main.go`)

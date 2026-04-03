---
schema_version: 1
id: global/deterministic-check-write-tools
title: Deterministic Check/Write Tool Pattern
status: active
suggested_context_bundles:
    - ci-tooling
aliases:
    - agent-os/standards/global/deterministic-check-write-tools
---

# Deterministic Check/Write Tool Pattern

- Tools default to check-only (exit non-zero if changes are needed)
- Tools only mutate files with an explicit flag (example: `--write`)
- Keep behavior deterministic:
  - Sorted file discovery/output
  - Skip noisy dirs (`.git`, `.direnv`, `node_modules`, ...)
  - Batch external command invocations to avoid OS arg limits

Examples:
- Check: `go run ./tools/gofmtcheck`
- Write: `go run ./tools/gofmtcheck --write`

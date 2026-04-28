---
schema_version: 1
id: ci/windows-portability-matrix
title: Windows Portability Matrix
status: active
suggested_context_bundles:
    - ci-tooling
    - runner-boundary
---

# Windows Portability Matrix

- Windows CI runs `just ci-portability` under PowerShell (no bash dependency)
- Test Node "min + max" versions within `runner/package.json` `engines` (pin exact versions)
- Pin Windows job tooling versions for reproducibility (Go, Node, just, gopls, baseline CLIs)
- Keep one canonical TLC/model-check gate in a single CI lane (currently Linux via `just ci`), and keep Windows focused on portability checks that do not depend on TLC runtime provisioning
- Keep failure-path tests portable: do not rely on POSIX-only chmod or permission semantics when a deterministic injected failure seam can exercise the same rollback or cleanup path on Windows

```yaml
strategy:
  fail-fast: false
  matrix:
    node-version:
      - "22.22.1" # min supported
      - "24.14.0" # latest supported

steps:
  - uses: actions/setup-go@...
    with:
      go-version: "1.25.7"
  - uses: actions/setup-node@...
    with:
      node-version: ${{ matrix.node-version }}
  - run: just ci-portability
```

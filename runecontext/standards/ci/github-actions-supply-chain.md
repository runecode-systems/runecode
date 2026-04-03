---
schema_version: 1
id: ci/github-actions-supply-chain
title: GitHub Actions Supply Chain
status: active
aliases:
    - agent-os/standards/ci/github-actions-supply-chain
---

# GitHub Actions Supply Chain

- Default workflow token permissions: `contents: read`
- Only expand permissions at job-level when needed
- Pin every `uses:` to a full commit SHA
- Add a human version comment: `# vX.Y.Z`
- Exception: local actions via `uses: ./.github/...` don't need SHA pinning

```yaml
permissions:
  contents: read

steps:
  - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
```

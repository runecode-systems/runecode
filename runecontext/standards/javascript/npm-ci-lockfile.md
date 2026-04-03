---
schema_version: 1
id: javascript/npm-ci-lockfile
title: Deterministic npm Installs (Runner)
status: active
aliases:
    - agent-os/standards/javascript/npm-ci-lockfile
---

# Deterministic npm Installs (Runner)

- Commit `runner/package-lock.json`
- CI uses `npm ci` (never `npm install`)
- CI/`just ci` must not produce lockfile diffs
- Lockfile changes only come from deliberate dependency updates in PRs

```sh
cd runner
npm ci
```

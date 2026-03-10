# Deterministic npm Installs (Runner)

- Commit `runner/package-lock.json`
- CI uses `npm ci` (never `npm install`)
- CI/`just ci` must not produce lockfile diffs
- Lockfile changes only come from deliberate dependency updates in PRs

```sh
cd runner
npm ci
```

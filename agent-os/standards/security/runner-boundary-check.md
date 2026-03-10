# Runner Boundary Check

- Treat `runner/` as untrusted
- Runner must never reference trusted code under `cmd/` or `internal/`
- Allowed cross-boundary reads: `protocol/schemas/` and `protocol/fixtures/` only
- CI must run the boundary check and fail on violations

Boundary-check requirements:
- Scan JS/TS across `runner/` (not only `runner/src`)
- Exclude deps/build dirs (`node_modules`, `dist`, `coverage`, ...)
- Detect escapes via relative paths, `cmd/`/`internal/` repo-root specifiers, and absolute paths
  - Include Windows drive-letter and UNC paths
- Avoid false positives for third-party package names containing `internal`/`cmd`
- Fail closed if no runner source files are found
- Violation messages use runner-relative paths with forward slashes

```sh
cd runner
npm run boundary-check
```

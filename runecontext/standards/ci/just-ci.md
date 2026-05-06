---
schema_version: 1
id: ci/just-ci
title: '`just ci` Convention'
status: active
suggested_context_bundles:
    - ci-tooling
---

# `just ci` Convention

- `just ci` is the canonical local check entrypoint
- CI may use `just ci-fast` plus dedicated required gates when a heavyweight check needs path-aware or merge-queue scheduling
- Required shared-Linux performance contracts run in the dedicated CI lane (`just ci-required-shared-linux`) rather than every local `just ci` run
- `just ci` is check-only:
  - No formatters in write mode
  - No lockfile updates (`flake.lock`, `go.sum`, `package-lock.json`)
- Put auto-fix behavior in separate recipes (example: `just fmt`)
- Put explicit repair workflows that change tracked files in separate recipes or tools rather than inside `just ci` (example: `just refresh-release-vendor-hash`)
- Put formal model checking behind explicit check-only recipes (`just model-check-core`, `just model-check-replay`, `just model-check`) and include full model checking in `just ci` for local parity
- In GitHub CI, keep the formal security-kernel check as a dedicated required gate so PR pushes can run the core model for security-kernel-relevant code or protocol changes, run the full model for formal-spec/tooling/workflow changes, and run the full model on merge queue and `main`
- Keep recipes cross-platform (Windows-friendly): avoid bash/unix-only tools and shell pipelines
- Redundant explicit steps in `just ci` are allowed when they make failures clearer (example: runner lint even if tests also run lint)

```make
ci:
  just ci-fast
  just model-check

ci-fast:
  go run ./tools/gofmtcheck
  go run github.com/golangci/golangci-lint/cmd/golangci-lint@...
  go vet ./...
  go run ./tools/checksourcequality
  go test ./...
  go build ./cmd/...
  cd runner && npm ci
  cd runner && npm run lint
  cd runner && npm test
  cd runner && npm run boundary-check
```

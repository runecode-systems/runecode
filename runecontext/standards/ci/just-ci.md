---
schema_version: 1
id: ci/just-ci
title: '`just ci` Convention'
status: active
suggested_context_bundles:
    - ci-tooling
---

# `just ci` Convention

- `just ci` is the canonical local+CI parity command
- `just ci` is check-only:
  - No formatters in write mode
  - No lockfile updates (`flake.lock`, `go.sum`, `package-lock.json`)
- Put auto-fix behavior in separate recipes (example: `just fmt`)
- Put formal model checking behind an explicit check-only recipe (currently `just model-check`) and include it in `just ci` when it is part of required parity
- Keep recipes cross-platform (Windows-friendly): avoid bash/unix-only tools and shell pipelines
- Redundant explicit steps in `just ci` are allowed when they make failures clearer (example: runner lint even if tests also run lint)

```make
ci:
  go run ./tools/gofmtcheck
  go run github.com/golangci/golangci-lint/cmd/golangci-lint@...
  go vet ./...
  go run ./tools/checksourcequality
  just model-check
  go test ./...
  go build ./cmd/...
  cd runner && npm ci
  cd runner && npm run lint
  cd runner && npm test
  cd runner && npm run boundary-check
```

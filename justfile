golangci_lint := "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8"

default:
  @just --list

fmt:
  go run ./tools/gofmtcheck --write

lint:
  go run ./tools/gofmtcheck
  go run {{golangci_lint}} run
  go vet ./...
  go run ./tools/checksourcequality
  cd runner && npm run lint
  cd runner && npm run boundary-check

test:
  go test ./...
  cd runner && npm test

model-check:
  go run ./tools/tlccheck --mode all

model-check-core:
  go run ./tools/tlccheck --mode core

model-check-replay:
  go run ./tools/tlccheck --mode replay

ci-fast:
  go run ./tools/gofmtcheck
  go run {{golangci_lint}} run
  go vet ./...
  go run ./tools/checksourcequality
  go test ./...
  go build ./cmd/...
  cd runner && npm ci
  cd runner && npm run lint
  cd runner && npm test
  cd runner && npm run boundary-check

ci:
  just ci-fast
  just model-check

ci-required-shared-linux:
  just ci-fast
  tmpdir="$(mktemp -d)" && trap 'rm -rf "$tmpdir"' EXIT && \
    go run ./tools/perfgatesharedlinux --output "$tmpdir/perf-check.json" && \
    go run ./tools/perfcontracts --check-output "$tmpdir/perf-check.json" --lane required_shared_linux \
      --metric-id metric.runner.boundary_check.wall_ms \
      --metric-id metric.runner.protocol_fixtures.wall_ms \
      --metric-id metric.broker.unary.session_list.p95_ms

ci-portability:
  go run ./tools/gofmtcheck
  go run {{golangci_lint}} run
  go vet ./...
  go run ./tools/checksourcequality
  go test ./...
  go build ./cmd/...
  cd runner && npm ci
  cd runner && npm run lint
  cd runner && npm test
  cd runner && npm run boundary-check

refresh-release-vendor-hash:
  go run ./tools/releasebuilder refresh-vendor-hash

dev:
  @just --list

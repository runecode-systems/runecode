golangci_lint := "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8"
dev_bin_dir := "/tmp/runecode-current/bin"
dev_gocache_dir := "/tmp/runecode-ci-cache/gocache"
dev_gotmp_dir := "/tmp/runecode-ci-cache/gotmp"

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
  cd runner && npm ci
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
  cd runner && npm ci
  go test ./...
  go build ./cmd/...
  cd runner && npm run lint
  cd runner && npm test
  cd runner && npm run boundary-check

ci:
  # Canonical local check entrypoint.
  # Required shared-Linux performance contracts run in CI via ci-required-shared-linux.
  just ci-fast
  just model-check

ci-required-shared-linux:
  just ci-fast
  tmpdir="$(mktemp -d)" && trap 'rm -rf "$tmpdir"' EXIT && \
    go run ./tools/perfgatesharedlinux --output "$tmpdir/perf-check.json" && \
    go run ./tools/perfcontracts --check-output "$tmpdir/perf-check.json" --lane required_shared_linux

ci-portability:
  go run ./tools/gofmtcheck
  go run {{golangci_lint}} run
  go vet ./...
  go run ./tools/checksourcequality
  cd runner && npm ci
  go test ./...
  go build ./cmd/...
  cd runner && npm run lint
  cd runner && npm test
  cd runner && npm run boundary-check

refresh-release-vendor-hash:
  go run ./tools/releasebuilder refresh-vendor-hash

dev:
  @just --list

tui-dev-build:
  mkdir -p {{dev_bin_dir}} {{dev_gocache_dir}} {{dev_gotmp_dir}}
  GOCACHE={{dev_gocache_dir}} GOTMPDIR={{dev_gotmp_dir}} TMPDIR={{dev_gotmp_dir}} go build -o {{dev_bin_dir}}/runecode ./cmd/runecode
  GOCACHE={{dev_gocache_dir}} GOTMPDIR={{dev_gotmp_dir}} TMPDIR={{dev_gotmp_dir}} go build -o {{dev_bin_dir}}/runecode-broker ./cmd/runecode-broker
  GOCACHE={{dev_gocache_dir}} GOTMPDIR={{dev_gotmp_dir}} TMPDIR={{dev_gotmp_dir}} go build -o {{dev_bin_dir}}/runecode-tui ./cmd/runecode-tui

tui-dev: tui-dev-build
  PATH={{dev_bin_dir}}:$PATH {{dev_bin_dir}}/runecode attach

tui-dev-restart: tui-dev-build
  PATH={{dev_bin_dir}}:$PATH {{dev_bin_dir}}/runecode restart
  PATH={{dev_bin_dir}}:$PATH {{dev_bin_dir}}/runecode attach

tui-dev-status: tui-dev-build
  PATH={{dev_bin_dir}}:$PATH {{dev_bin_dir}}/runecode status

tui-dev-stop: tui-dev-build
  PATH={{dev_bin_dir}}:$PATH {{dev_bin_dir}}/runecode stop

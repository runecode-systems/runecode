golangci_lint := "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8"
dev_bin_dir := "/tmp/runecode-current/bin"
dev_gocache_dir := "/tmp/runecode-ci-cache/gocache"
dev_gotmp_dir := "/tmp/runecode-ci-cache/gotmp"
tui_snapshot_dir := "/tmp/runecode-tui-snapshots"
tui_snapshot_tag := "runecode_tui_snapshot"

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
  just tui-snapshot-ci
  just tui-release-safety
  just model-check

ci-required-shared-linux:
  just ci
  tmpdir="$(mktemp -d)" && trap 'rm -rf "$tmpdir"' EXIT && \
    go run ./tools/perfgatesharedlinux --output "$tmpdir/perf-check.json" && \
    go run ./tools/perfcontracts --check-output "$tmpdir/perf-check.json" --lane required_shared_linux

ci-portability-unix:
  just ci-fast
  just tui-snapshot-ci
  just tui-release-safety

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

_tui-snapshot scenario: tui-dev-snapshot-build
	mkdir -p {{tui_snapshot_dir}}
	{{dev_bin_dir}}/runecode-tui --snapshot-scenario {{scenario}} --snapshot-output-dir {{tui_snapshot_dir}}
	if command -v magick >/dev/null 2>&1; then for svg in {{tui_snapshot_dir}}/*.svg; do [ -e "$svg" ] || continue; magick "$svg" "${svg%.svg}.png"; done; elif command -v convert >/dev/null 2>&1; then for svg in {{tui_snapshot_dir}}/*.svg; do [ -e "$svg" ] || continue; convert "$svg" "${svg%.svg}.png"; done; fi

tui-dev-snapshot-build:
	mkdir -p {{dev_bin_dir}} {{dev_gocache_dir}} {{dev_gotmp_dir}}
	GOCACHE={{dev_gocache_dir}} GOTMPDIR={{dev_gotmp_dir}} TMPDIR={{dev_gotmp_dir}} go build -tags {{tui_snapshot_tag}} -o {{dev_bin_dir}}/runecode-tui ./cmd/runecode-tui

_tui-snapshot-viewport scenario viewport: tui-dev-snapshot-build
	mkdir -p {{tui_snapshot_dir}}
	{{dev_bin_dir}}/runecode-tui --snapshot-scenario {{scenario}} --snapshot-output-dir {{tui_snapshot_dir}} --snapshot-viewport {{viewport}}
	if command -v magick >/dev/null 2>&1; then for svg in {{tui_snapshot_dir}}/*.svg; do [ -e "$svg" ] || continue; magick "$svg" "${svg%.svg}.png"; done; elif command -v convert >/dev/null 2>&1; then for svg in {{tui_snapshot_dir}}/*.svg; do [ -e "$svg" ] || continue; convert "$svg" "${svg%.svg}.png"; done; fi

tui-snapshot-dashboard-multi:
	just _tui-snapshot-viewport dashboard desktop
	just _tui-snapshot-viewport dashboard mobile

tui-snapshot-dashboard:
	just _tui-snapshot dashboard

tui-snapshot-action-center:
	just _tui-snapshot action-center

tui-snapshot-all:
	just _tui-snapshot all

tui-snapshot-review:
	just tui-snapshot-all
	python3 ./tools/tui_snapshot_review.py "{{tui_snapshot_dir}}"

tui-snapshot-ci:
	go run ./tools/tuisnapshotci

tui-release-safety:
	mkdir -p {{dev_gocache_dir}} {{dev_gotmp_dir}}
	tmpdir="$$(mktemp -d)" && trap 'rm -rf "$$tmpdir"' EXIT && \
	  GOCACHE={{dev_gocache_dir}} GOTMPDIR={{dev_gotmp_dir}} TMPDIR={{dev_gotmp_dir}} go build -o "$$tmpdir/runecode-tui" ./cmd/runecode-tui && \
	  if "$$tmpdir/runecode-tui" --snapshot-scenario all >/dev/null 2>/dev/null; then \
	    printf '%s\n' 'runecode-tui default build unexpectedly accepted --snapshot-scenario all' >&2; \
	    exit 1; \
	  fi

golangci_lint := "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8"
dev_bin_dir := "/tmp/runecode-current/bin"
dev_gocache_dir := "/tmp/runecode-ci-cache/gocache"
dev_gotmp_dir := "/tmp/runecode-ci-cache/gotmp"
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
	sh ./tools/tui_snapshot_local.sh --binary "{{dev_bin_dir}}/runecode-tui" --scenario {{scenario}}

_tui-snapshot-bundle bundle: tui-dev-snapshot-build
	sh ./tools/tui_snapshot_local.sh --binary "{{dev_bin_dir}}/runecode-tui" --bundle {{bundle}}

tui-dev-snapshot-build:
	mkdir -p {{dev_bin_dir}} {{dev_gocache_dir}} {{dev_gotmp_dir}}
	GOCACHE={{dev_gocache_dir}} GOTMPDIR={{dev_gotmp_dir}} TMPDIR={{dev_gotmp_dir}} go build -tags {{tui_snapshot_tag}} -o {{dev_bin_dir}}/runecode-tui ./cmd/runecode-tui

_tui-snapshot-viewport scenario viewport: tui-dev-snapshot-build
	sh ./tools/tui_snapshot_local.sh --binary "{{dev_bin_dir}}/runecode-tui" --scenario {{scenario}} --viewport {{viewport}}

tui-snapshot-dashboard-multi: tui-dev-snapshot-build
	sh ./tools/tui_snapshot_local.sh --binary "{{dev_bin_dir}}/runecode-tui" --scenario dashboard --viewport desktop
	sh ./tools/tui_snapshot_local.sh --binary "{{dev_bin_dir}}/runecode-tui" --scenario dashboard --viewport mobile

tui-snapshot-dashboard:
	just _tui-snapshot dashboard

tui-snapshot-action-center:
	just _tui-snapshot action-center

tui-snapshot-all:
	just _tui-snapshot all

tui-snapshot-audit-full:
	just _tui-snapshot-bundle full-audit

tui-snapshot-audit-dashboard:
	just _tui-snapshot-bundle dashboard-audit

tui-snapshot-audit-action-center:
	just _tui-snapshot-bundle action-center-audit

tui-snapshot-audit-runs:
	just _tui-snapshot-bundle runs-audit

tui-snapshot-audit-approvals:
	just _tui-snapshot-bundle approvals-audit

tui-snapshot-audit-audit:
	just _tui-snapshot-bundle audit-route-audit

tui-snapshot-audit-status:
	just _tui-snapshot-bundle status-audit

tui-snapshot-audit-setup:
	just _tui-snapshot-bundle setup-audit

tui-snapshot-audit-chat:
	just _tui-snapshot-bundle chat-audit

tui-snapshot-review: tui-dev-snapshot-build
	sh ./tools/tui_snapshot_local.sh --binary "{{dev_bin_dir}}/runecode-tui" --bundle full-audit --review --review-mode summary

tui-snapshot-review-full: tui-dev-snapshot-build
	sh ./tools/tui_snapshot_local.sh --binary "{{dev_bin_dir}}/runecode-tui" --bundle full-audit --review --review-mode summary

tui-snapshot-review-dashboard: tui-dev-snapshot-build
	sh ./tools/tui_snapshot_local.sh --binary "{{dev_bin_dir}}/runecode-tui" --bundle dashboard-audit --review --review-mode summary

tui-snapshot-review-action-center: tui-dev-snapshot-build
	sh ./tools/tui_snapshot_local.sh --binary "{{dev_bin_dir}}/runecode-tui" --bundle action-center-audit --review --review-mode summary

tui-snapshot-review-summary: tui-dev-snapshot-build
	sh ./tools/tui_snapshot_local.sh --binary "{{dev_bin_dir}}/runecode-tui" --bundle full-audit --review --review-mode summary

tui-snapshot-review-list: tui-dev-snapshot-build
	sh ./tools/tui_snapshot_local.sh --binary "{{dev_bin_dir}}/runecode-tui" --bundle full-audit --review --review-mode list

tui-snapshot-review-open: tui-dev-snapshot-build
	sh ./tools/tui_snapshot_local.sh --binary "{{dev_bin_dir}}/runecode-tui" --bundle full-audit --review --review-mode open

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

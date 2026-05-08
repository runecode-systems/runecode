# Runner Kernel Foundation

This package is the untrusted runner-side kernel foundation for durable
`RunPlan` consumption.

## Scope

The runner remains thin and seamful:

- `run-plan.ts`: load + validate immutable `RunPlan`
- `durable-state.ts`: runner-internal journal/snapshot durable state
- `scheduler.ts`: plan-bound work listing skeleton
- `executor-adapter.ts`: executor dispatch seam
- `report-emitter.ts`: typed checkpoint/result request seam
- `broker-client.ts`: broker transport abstraction seam
- `kernel.ts`: composition root
- `cli.ts`: normal product runner launch entrypoint (`npm run start -- --plan-file <path>`)

## Trust Boundary Rules

- Runner is untrusted and must not import `cmd/`, `internal/`, or `tools/`.
- Cross-boundary file reads are limited to `protocol/schemas/` and
  `protocol/fixtures/`.
- No runner-local authorization or workflow planning authority.

## Durable-State Identity Binding

`FileDurableStateStore` binds snapshot/journal state to `{run_id, plan_id}` and
throws `PlanIdentityMismatchError` when a stale/superseded identity is loaded.

This fail-closed behavior prevents silently replaying state from a different
broker-compiled immutable plan.

The append-only journal is authoritative for recovery. Snapshot files are a
cache that can be healed from journal replay after a crash rather than becoming
the source of truth.

## Supported Product Launch Path

For the supported beta slice, launch the runner via:

- `npm run start -- --plan-file <persisted-run-plan-path> --plan-root <trusted-plan-root> --broker-transport stdio`

This path is intentionally plan-first and requires explicit broker transport
wiring in the process environment. The kernel fails closed if a broker transport
is missing, a typed transport response is absent, or the broker rejects emitted
checkpoint/result reports.

The trusted broker launcher must provide `RUNECODE_PROTOCOL_SCHEMAS_ROOT` as an
absolute path to the checked-in `protocol/schemas` bundle and launch the runner
from the `runner/` working directory. The runner also confines `--plan-file` and
`--state-root` under `--plan-root` so the untrusted process cannot be steered
toward arbitrary host paths through product launch arguments.

### Minimal local transport seam

The supported untrusted integration seam is typed newline-delimited JSON over
stdin/stdout:

- runner writes request envelopes with `message_type` plus typed protocol
  payloads (`DependencyCacheHandoffRequest`, `RunnerCheckpointReportRequest`,
  `RunnerResultReportRequest`)
- broker-side launcher/integration must answer with matching typed response
  envelopes validated against protocol schemas before the runner accepts them

This keeps the runner transport concrete for product launch without granting it
planning or authorization authority.

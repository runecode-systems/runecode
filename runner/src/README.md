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

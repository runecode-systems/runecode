/**
 * Thin untrusted runner kernel foundation.
 *
 * This package intentionally exposes seamful modules (plan loader, durable
 * state store, scheduler, executor/report seams) without introducing runner-
 * local policy authority or trusted-domain coupling.
 */

export {
  ProtocolSchemaBundle,
  type SchemaValidationResult,
} from "./protocol-schema-bundle.ts";
export {
  RunPlanLoader,
  RUN_PLAN_SCHEMA_ID,
  type RunnerPlanIdentity,
  type RunnerPlan,
  type RunnerPlanEntry,
} from "./run-plan.ts";
export {
  FileDurableStateStore,
  PlanIdentityMismatchError,
  type DurableSnapshot,
  type DurableJournalRecord,
  type DurableStateView,
} from "./durable-state.ts";
export {
  PlanScheduler,
  type ScheduledWorkItem,
} from "./scheduler.ts";
export {
  type ExecutionOutcome,
  type ExecutorAdapter,
  ExecutorAdapterRegistry,
} from "./executor-adapter.ts";
export {
  type RunnerBrokerClient,
  type BrokerAcknowledge,
  NoopRunnerBrokerClient,
} from "./broker-client.ts";
export {
  ReportEmitter,
  type CheckpointReportInput,
  type ResultReportInput,
} from "./report-emitter.ts";
export {
  RunnerKernel,
  type RunnerKernelOptions,
} from "./kernel.ts";

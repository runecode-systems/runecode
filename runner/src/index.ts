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
  InvalidApprovalWaitError,
  PlanIdentityMismatchError,
  DurableReplayError,
  replayDurableState,
  type ApprovalBindingKind,
  type ApprovalWaitStatus,
  type DurableApprovalWait,
  type DurableApprovalBlockedWorkScope,
  type DurableApprovalBrokerCorrelation,
  type EnterApprovalWaitInput,
  type ResolveApprovalWaitInput,
  type DurableJournalKind,
  type DurableAppendRecordInput,
  type DurableReplayState,
  type DurableSnapshot,
  type DurableJournalRecord,
  type DurableStateView,
} from "./durable-state.ts";
export {
  PlanScheduler,
  type SchedulerState,
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
  RUNNER_CHECKPOINT_REPORT_SCHEMA_ID,
  RUNNER_RESULT_REPORT_SCHEMA_ID,
  RUNNER_CHECKPOINT_REPORT_REQUEST_SCHEMA_ID,
  RUNNER_RESULT_REPORT_REQUEST_SCHEMA_ID,
  RUNNER_CONTRACT_SCHEMA_VERSION,
  type PlanBoundExecutionIdentity,
  type RunnerCheckpointReport,
  type RunnerResultReport,
  type RunnerCheckpointReportRequest,
  type RunnerResultReportRequest,
} from "./contracts.ts";
export {
  ReportEmitter,
  type CheckpointReportInput,
  type ResultReportInput,
} from "./report-emitter.ts";
export {
  DurableRuntimeSeam,
  type RuntimeCheckpointInput,
  type RuntimeWaitInput,
  type RuntimeWaitResumeInput,
  type RuntimeRestoredWait,
  type RunnerRuntimeSeam,
} from "./runtime-seam.ts";
export {
  RunnerKernel,
  type ApprovalWaitResolution,
  type ApprovalWaitResolver,
  type KernelExecutionContext,
  type KernelExecutionModule,
  type RunnerKernelOptions,
} from "./kernel.ts";

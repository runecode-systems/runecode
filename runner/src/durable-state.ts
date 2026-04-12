/**
 * Runner-local durable state journal and snapshot store.
 *
 * This state is explicitly non-authoritative and internal to the untrusted
 * runner process. The append-only journal is authoritative for recovery, while
 * the snapshot is a cache that can be healed from journal replay after a crash.
 */

export {
  SNAPSHOT_SCHEMA_VERSION,
  JOURNAL_SCHEMA_VERSION,
  DURABLE_ACTION_SCOPE_KINDS,
  DURABLE_BLOCKED_ACTION_KINDS,
  APPROVAL_BINDING_KINDS,
  APPROVAL_WAIT_STATUSES,
  BLOCKED_SCOPE_KINDS,
  DURABLE_JOURNAL_KINDS,
  DURABLE_APPROVAL_CLEAR_STATUSES,
  DURABLE_GATE_ATTEMPT_OUTCOMES,
  DURABLE_STEP_ATTEMPT_OUTCOMES,
  DURABLE_TERMINAL_STATUSES,
  PlanIdentityMismatchError,
  DurableReplayError,
  InvalidApprovalWaitError,
  assertIdentityMatch,
  assertBoundIdentity,
  cloneApprovalWait,
  cloneBlockedScope,
  cloneBrokerCorrelation,
  assertObjectRecord,
  requireString,
  optionalString,
  requireNonNegativeInt,
  requireEnum,
  requireExactString,
  trimOptional,
  trimRequired,
  lastJournalOccurredAt,
  type DurableJournalKind,
  type DurableActionScopeKind,
  type DurableBlockedActionKind,
  type DurableGateAttemptOutcome,
  type DurableStepAttemptOutcome,
  type DurableRunTerminalStatus,
  type ApprovalBindingKind,
  type ApprovalWaitStatus,
  type DurableApprovalClearStatus,
  type DurableApprovalBlockedWorkScope,
  type DurableApprovalBrokerCorrelation,
  type DurableApprovalWait,
  type DurableSnapshot,
  type DurableJournalBase,
  type DurableRunStartedRecord,
  type DurableStageEnteredRecord,
  type DurableStepAttemptStartedRecord,
  type DurableActionRequestIssuedRecord,
  type DurableApprovalWaitEnteredRecord,
  type DurableApprovalWaitClearedRecord,
  type DurableGateAttemptStartedRecord,
  type DurableGateAttemptFinishedRecord,
  type DurableStepAttemptFinishedRecord,
  type DurableRunTerminalRecord,
  type DurableJournalRecord,
  type DurableAppendRecordInput,
  type EnterApprovalWaitInput,
  type ResolveApprovalWaitInput,
  type DurableReplayState,
  type DurableStateView,
} from "./durable-state/types.ts";
export {
  sanitizeBlockedScope,
  sanitizeBrokerCorrelation,
  blockedScopeToDurableScope,
  approvalWaitBindingsMatch,
  assertApprovalBinding,
} from "./durable-state/helpers.ts";
export {
  buildRecord,
  assertDurableRecordMatches,
  parseSnapshot,
  parseJournalRecord,
} from "./durable-state/codec.ts";
export {
  replayDurableState,
  replayDurableStateInternal,
  healSnapshotFromJournal,
  snapshotNeedsRewrite,
} from "./durable-state/replay.ts";
export { FileDurableStateStore } from "./durable-state/store.ts";

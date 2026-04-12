import type { RunnerPlanIdentity } from "../run-plan.ts";

export const SNAPSHOT_SCHEMA_VERSION = "2";
export const JOURNAL_SCHEMA_VERSION = "2";

export const DURABLE_ACTION_SCOPE_KINDS = ["run", "stage", "step_attempt", "gate_attempt"] as const;
export const DURABLE_BLOCKED_ACTION_KINDS = ["action_gate_override", "stage_summary_sign_off"] as const;
export const APPROVAL_BINDING_KINDS = ["exact_action", "stage_sign_off"] as const;
export const APPROVAL_WAIT_STATUSES = ["pending", "approved", "denied", "expired", "superseded", "cancelled", "consumed"] as const;
export const BLOCKED_SCOPE_KINDS = ["workspace", "run", "stage", "step", "action_kind"] as const;
export const DURABLE_JOURNAL_KINDS = [
  "run_started",
  "stage_entered",
  "step_attempt_started",
  "action_request_issued",
  "approval_wait_entered",
  "approval_wait_cleared",
  "gate_attempt_started",
  "gate_attempt_finished",
  "step_attempt_finished",
  "run_terminal",
] as const;
export const DURABLE_APPROVAL_CLEAR_STATUSES = ["approved", "denied", "expired", "superseded", "cancelled", "consumed"] as const;
export const DURABLE_GATE_ATTEMPT_OUTCOMES = ["passed", "failed", "blocked"] as const;
export const DURABLE_STEP_ATTEMPT_OUTCOMES = ["succeeded", "failed", "cancelled"] as const;
export const DURABLE_TERMINAL_STATUSES = ["succeeded", "failed", "cancelled"] as const;

export type DurableJournalKind = (typeof DURABLE_JOURNAL_KINDS)[number];
export type DurableActionScopeKind = (typeof DURABLE_ACTION_SCOPE_KINDS)[number];
export type DurableGateAttemptOutcome = (typeof DURABLE_GATE_ATTEMPT_OUTCOMES)[number];
export type DurableStepAttemptOutcome = (typeof DURABLE_STEP_ATTEMPT_OUTCOMES)[number];
export type DurableRunTerminalStatus = (typeof DURABLE_TERMINAL_STATUSES)[number];
export type DurableBlockedActionKind = (typeof DURABLE_BLOCKED_ACTION_KINDS)[number];
export type ApprovalBindingKind = (typeof APPROVAL_BINDING_KINDS)[number];
export type ApprovalWaitStatus = (typeof APPROVAL_WAIT_STATUSES)[number];
export type DurableApprovalClearStatus = Exclude<ApprovalWaitStatus, "pending">;

export type DurableApprovalBlockedWorkScope = {
  scope_kind: (typeof BLOCKED_SCOPE_KINDS)[number];
  workspace_id?: string;
  run_id?: string;
  stage_id?: string;
  step_id?: string;
  role_instance_id?: string;
  action_kind: DurableBlockedActionKind;
};

export type DurableApprovalBrokerCorrelation = {
  request_id?: string;
  action_request_id?: string;
  approval_request_id?: string;
  operation_id?: string;
  parent_operation_id?: string;
};

export type DurableApprovalWait = {
  approval_id: string;
  action_request_id: string;
  run_id: string;
  plan_id: string;
  binding_kind: ApprovalBindingKind;
  bound_action_hash?: string;
  bound_stage_summary_hash?: string;
  blocked_scope: DurableApprovalBlockedWorkScope;
  broker_correlation: DurableApprovalBrokerCorrelation;
  status: "pending";
  wait_entered_idempotency_key: string;
  entered_at: string;
  updated_at: string;
};

export type DurableSnapshot = {
  schema_version: typeof SNAPSHOT_SCHEMA_VERSION;
  run_id: string;
  plan_id: string;
  supersedes_plan_id?: string;
  last_sequence: number;
  pending_approval_waits: DurableApprovalWait[];
  created_at: string;
  updated_at: string;
};

export type DurableJournalBase = {
  schema_version: typeof JOURNAL_SCHEMA_VERSION;
  sequence: number;
  kind: DurableJournalKind;
  run_id: string;
  plan_id: string;
  idempotency_key: string;
  occurred_at: string;
};

export type DurableRunStartedRecord = DurableJournalBase & {
  kind: "run_started";
  run_scope_id: string;
};

export type DurableStageEnteredRecord = DurableJournalBase & {
  kind: "stage_entered";
  stage_id: string;
};

export type DurableStepAttemptStartedRecord = DurableJournalBase & {
  kind: "step_attempt_started";
  stage_id?: string;
  step_id: string;
  step_attempt_id: string;
};

export type DurableActionRequestIssuedRecord = DurableJournalBase & {
  kind: "action_request_issued";
  action_request_id: string;
  scope_kind: DurableActionScopeKind;
  scope_id: string;
};

export type DurableApprovalWaitEnteredRecord = DurableJournalBase & {
  kind: "approval_wait_entered";
  approval_wait_id: string;
  action_request_id: string;
  binding_kind: ApprovalBindingKind;
  bound_action_hash?: string;
  bound_stage_summary_hash?: string;
  blocked_scope: DurableApprovalBlockedWorkScope;
  broker_correlation: DurableApprovalBrokerCorrelation;
};

export type DurableApprovalWaitClearedRecord = DurableJournalBase & {
  kind: "approval_wait_cleared";
  approval_wait_id: string;
  action_request_id: string;
  status: DurableApprovalClearStatus;
};

export type DurableGateAttemptStartedRecord = DurableJournalBase & {
  kind: "gate_attempt_started";
  stage_id?: string;
  gate_id: string;
  gate_attempt_id: string;
};

export type DurableGateAttemptFinishedRecord = DurableJournalBase & {
  kind: "gate_attempt_finished";
  gate_id: string;
  gate_attempt_id: string;
  outcome: DurableGateAttemptOutcome;
};

export type DurableStepAttemptFinishedRecord = DurableJournalBase & {
  kind: "step_attempt_finished";
  step_id: string;
  step_attempt_id: string;
  outcome: DurableStepAttemptOutcome;
};

export type DurableRunTerminalRecord = DurableJournalBase & {
  kind: "run_terminal";
  terminal_status: DurableRunTerminalStatus;
};

export type DurableJournalRecord =
  | DurableRunStartedRecord
  | DurableStageEnteredRecord
  | DurableStepAttemptStartedRecord
  | DurableActionRequestIssuedRecord
  | DurableApprovalWaitEnteredRecord
  | DurableApprovalWaitClearedRecord
  | DurableGateAttemptStartedRecord
  | DurableGateAttemptFinishedRecord
  | DurableStepAttemptFinishedRecord
  | DurableRunTerminalRecord;

type DurableAppendCommon = {
  idempotency_key: string;
  occurred_at?: string;
};

export type DurableAppendRecordInput =
  | (DurableAppendCommon & { kind: "run_started"; run_scope_id: string })
  | (DurableAppendCommon & { kind: "stage_entered"; stage_id: string })
  | (DurableAppendCommon & { kind: "step_attempt_started"; stage_id?: string; step_id: string; step_attempt_id: string })
  | (DurableAppendCommon & { kind: "action_request_issued"; action_request_id: string; scope_kind: DurableActionScopeKind; scope_id: string })
  | (DurableAppendCommon & {
      kind: "approval_wait_entered";
      approval_wait_id: string;
      action_request_id: string;
      binding_kind: ApprovalBindingKind;
      bound_action_hash?: string;
      bound_stage_summary_hash?: string;
      blocked_scope: DurableApprovalBlockedWorkScope;
      broker_correlation: DurableApprovalBrokerCorrelation;
    })
  | (DurableAppendCommon & {
      kind: "approval_wait_cleared";
      approval_wait_id: string;
      action_request_id: string;
      status: DurableApprovalClearStatus;
    })
  | (DurableAppendCommon & { kind: "gate_attempt_started"; stage_id?: string; gate_id: string; gate_attempt_id: string })
  | (DurableAppendCommon & { kind: "gate_attempt_finished"; gate_id: string; gate_attempt_id: string; outcome: DurableGateAttemptOutcome })
  | (DurableAppendCommon & { kind: "step_attempt_finished"; step_id: string; step_attempt_id: string; outcome: DurableStepAttemptOutcome })
  | (DurableAppendCommon & { kind: "run_terminal"; terminal_status: DurableRunTerminalStatus });

export type EnterApprovalWaitInput = {
  approval_id: string;
  run_id: string;
  plan_id: string;
  binding_kind: ApprovalBindingKind;
  bound_action_hash?: string;
  bound_stage_summary_hash?: string;
  blocked_scope: DurableApprovalBlockedWorkScope;
  broker_correlation: DurableApprovalBrokerCorrelation;
  idempotency_key: string;
  occurred_at?: string;
};

export type ResolveApprovalWaitInput = {
  approval_id: string;
  run_id: string;
  plan_id: string;
  binding_kind: ApprovalBindingKind;
  bound_action_hash?: string;
  bound_stage_summary_hash?: string;
  status: DurableApprovalClearStatus;
  idempotency_key: string;
  occurred_at?: string;
};

export type DurableReplayState = {
  run_id: string;
  plan_id: string;
  last_sequence: number;
  started: boolean;
  terminal: { sequence: number; terminal_status: DurableRunTerminalStatus } | null;
  scheduler: {
    entered_stage_ids: string[];
    current_stage_id: string | null;
  };
  waits: {
    pending_approval_waits: string[];
    resolved_approval_waits: Array<{ approval_id: string; status: DurableApprovalClearStatus }>;
  };
  attempts: {
    active_step_attempt_ids: string[];
    finished_step_attempt_ids: string[];
    active_gate_attempt_ids: string[];
    finished_gate_attempt_ids: string[];
  };
  actions: {
    issued_action_request_ids: string[];
  };
};

export type DurableStateView = {
  snapshot: DurableSnapshot;
  journal: DurableJournalRecord[];
};

export type DurableReplayInternal = DurableReplayState & {
  pending_approval_waits_by_id: Map<string, DurableApprovalWait>;
  has_run_started: boolean;
};

export class PlanIdentityMismatchError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "PlanIdentityMismatchError";
  }
}

export class DurableReplayError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "DurableReplayError";
  }
}

export class InvalidApprovalWaitError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "InvalidApprovalWaitError";
  }
}

export function assertIdentityMatch(snapshot: DurableSnapshot, planIdentity: RunnerPlanIdentity): void {
  if (snapshot.run_id !== planIdentity.run_id) {
    throw new PlanIdentityMismatchError(`durable snapshot run_id ${snapshot.run_id} does not match plan run_id ${planIdentity.run_id}`);
  }
  if (snapshot.plan_id !== planIdentity.plan_id) {
    throw new PlanIdentityMismatchError(`durable snapshot plan_id ${snapshot.plan_id} does not match plan plan_id ${planIdentity.plan_id}`);
  }
  if ((snapshot.supersedes_plan_id ?? "") !== (planIdentity.supersedes_plan_id ?? "")) {
    throw new PlanIdentityMismatchError(
      `durable snapshot supersedes_plan_id ${snapshot.supersedes_plan_id ?? "<none>"} does not match plan supersedes_plan_id ${planIdentity.supersedes_plan_id ?? "<none>"}`,
    );
  }
}

export function assertBoundIdentity(snapshot: DurableSnapshot, runId: string, planId: string): void {
  if (snapshot.run_id !== runId || snapshot.plan_id !== planId) {
    throw new PlanIdentityMismatchError(`binding ${runId}/${planId} does not match active durable binding ${snapshot.run_id}/${snapshot.plan_id}`);
  }
}

export function cloneApprovalWait(wait: DurableApprovalWait): DurableApprovalWait {
  return {
    ...wait,
    blocked_scope: cloneBlockedScope(wait.blocked_scope),
    broker_correlation: cloneBrokerCorrelation(wait.broker_correlation),
  };
}

export function cloneBlockedScope(scope: DurableApprovalBlockedWorkScope): DurableApprovalBlockedWorkScope {
  return { ...scope };
}

export function cloneBrokerCorrelation(correlation: DurableApprovalBrokerCorrelation): DurableApprovalBrokerCorrelation {
  return { ...correlation };
}

export function assertObjectRecord(value: unknown, location: string): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new Error(`${location} must be an object`);
  }
  return value as Record<string, unknown>;
}

export function requireString(record: Record<string, unknown>, field: string, location: string): string {
  const value = record[field];
  if (typeof value !== "string" || value.trim().length === 0) {
    throw new Error(`${location}.${field} must be a non-empty string`);
  }
  return value.trim();
}

export function optionalString(record: Record<string, unknown>, field: string, location: string): string | undefined {
  const value = record[field];
  if (value === undefined) {
    return undefined;
  }
  if (typeof value !== "string" || value.trim().length === 0) {
    throw new Error(`${location}.${field} must be a non-empty string when provided`);
  }
  return value.trim();
}

export function requireNonNegativeInt(record: Record<string, unknown>, field: string, location: string): number {
  const value = record[field];
  if (typeof value !== "number" || !Number.isInteger(value) || value < 0) {
    throw new Error(`${location}.${field} must be a non-negative integer`);
  }
  return value;
}

export function requireEnum<T extends string>(record: Record<string, unknown>, field: string, allowed: readonly T[], location: string): T {
  const value = record[field];
  if (typeof value !== "string" || !allowed.includes(value as T)) {
    throw new Error(`${location}.${field} must be one of: ${allowed.join(", ")}`);
  }
  return value as T;
}

export function requireExactString(record: Record<string, unknown>, field: string, expected: string, location: string): string {
  const value = requireString(record, field, location);
  if (value !== expected) {
    throw new Error(`unsupported ${location}.${field} ${value}`);
  }
  return value;
}

export function trimOptional(value: string | undefined): string | undefined {
  if (value === undefined) {
    return undefined;
  }
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : undefined;
}

export function trimRequired(value: string, location: string): string {
  const trimmed = value.trim();
  if (trimmed.length === 0) {
    throw new InvalidApprovalWaitError(`${location} must be a non-empty string`);
  }
  return trimmed;
}

export function lastJournalOccurredAt(journal: DurableJournalRecord[]): string | undefined {
  if (journal.length === 0) {
    return undefined;
  }
  return journal[journal.length - 1].occurred_at;
}

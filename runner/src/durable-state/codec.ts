import {
  APPROVAL_BINDING_KINDS,
  DURABLE_ACTION_SCOPE_KINDS,
  DURABLE_APPROVAL_CLEAR_STATUSES,
  DURABLE_GATE_ATTEMPT_OUTCOMES,
  DURABLE_JOURNAL_KINDS,
  DURABLE_STEP_ATTEMPT_OUTCOMES,
  DURABLE_TERMINAL_STATUSES,
  JOURNAL_SCHEMA_VERSION,
  SNAPSHOT_SCHEMA_VERSION,
  type DurableAppendRecordInput,
  type DurableApprovalBrokerCorrelation,
  type DurableApprovalWait,
  type DurableJournalBase,
  type DurableJournalRecord,
  type DurableSnapshot,
  DurableReplayError,
  assertObjectRecord,
  cloneApprovalWait,
  optionalString,
  requireEnum,
  requireExactString,
  requireNonNegativeInt,
  requireString,
} from "./types.ts";
import {
  assertApprovalBinding,
  blockedScopeToDurableScope,
  parseBlockedScopeFields,
  sanitizeBlockedScope,
  sanitizeBrokerCorrelation,
} from "./helpers.ts";

export function buildRecord(snapshot: DurableSnapshot, sequence: number, input: DurableAppendRecordInput): DurableJournalRecord {
  const occurredAt = input.occurred_at ?? new Date().toISOString();
  const base: DurableJournalBase = {
    schema_version: JOURNAL_SCHEMA_VERSION,
    sequence,
    kind: input.kind,
    run_id: snapshot.run_id,
    plan_id: snapshot.plan_id,
    idempotency_key: input.idempotency_key,
    occurred_at: occurredAt,
  };

  switch (input.kind) {
    case "run_started":
      return { ...base, kind: input.kind, run_scope_id: input.run_scope_id };
    case "stage_entered":
      return { ...base, kind: input.kind, stage_id: input.stage_id };
    case "step_attempt_started":
      return { ...base, kind: input.kind, stage_id: input.stage_id, step_id: input.step_id, step_attempt_id: input.step_attempt_id };
    case "action_request_issued":
      return { ...base, kind: input.kind, action_request_id: input.action_request_id, scope_kind: input.scope_kind, scope_id: input.scope_id };
    case "approval_wait_entered":
      assertApprovalBinding(input.binding_kind, input.bound_action_hash, input.bound_stage_summary_hash, input.approval_wait_id);
      return {
        ...base,
        kind: input.kind,
        approval_wait_id: input.approval_wait_id,
        action_request_id: input.action_request_id,
        binding_kind: input.binding_kind,
        bound_action_hash: input.bound_action_hash,
        bound_stage_summary_hash: input.bound_stage_summary_hash,
        blocked_scope: sanitizeBlockedScope(input.blocked_scope, `approval_wait_entered ${input.approval_wait_id}`),
        broker_correlation: sanitizeBrokerCorrelation(input.broker_correlation),
      };
    case "approval_wait_cleared":
      return {
        ...base,
        kind: input.kind,
        approval_wait_id: input.approval_wait_id,
        action_request_id: input.action_request_id,
        status: input.status,
      };
    case "gate_attempt_started":
      return { ...base, kind: input.kind, stage_id: input.stage_id, gate_id: input.gate_id, gate_attempt_id: input.gate_attempt_id };
    case "gate_attempt_finished":
      return { ...base, kind: input.kind, gate_id: input.gate_id, gate_attempt_id: input.gate_attempt_id, outcome: input.outcome };
    case "step_attempt_finished":
      return { ...base, kind: input.kind, step_id: input.step_id, step_attempt_id: input.step_attempt_id, outcome: input.outcome };
    case "run_terminal":
      return { ...base, kind: input.kind, terminal_status: input.terminal_status };
    default:
      throw new DurableReplayError(`unsupported durable journal kind ${(input as { kind: string }).kind}`);
  }
}

export function assertDurableRecordMatches(existing: DurableJournalRecord, input: DurableAppendRecordInput): void {
  const occurredAt = existing.occurred_at;
  if (input.occurred_at !== undefined && input.occurred_at !== occurredAt) {
    throw new DurableReplayError(`idempotency key ${input.idempotency_key} conflicts with existing ${existing.kind} occurred_at`);
  }

  const expected = buildRecord(
    {
      schema_version: SNAPSHOT_SCHEMA_VERSION,
      run_id: existing.run_id,
      plan_id: existing.plan_id,
      supersedes_plan_id: undefined,
      last_sequence: existing.sequence - 1,
      pending_approval_waits: [],
      created_at: occurredAt,
      updated_at: occurredAt,
    },
    existing.sequence,
    {
      ...input,
      occurred_at: occurredAt,
    },
  );

  if (JSON.stringify(existing) !== JSON.stringify(expected)) {
    throw new DurableReplayError(`idempotency key ${input.idempotency_key} conflicts with existing ${existing.kind} record`);
  }
}

export function parseSnapshot(value: unknown, location: string): DurableSnapshot {
  const record = assertObjectRecord(value, location);
  requireExactString(record, "schema_version", SNAPSHOT_SCHEMA_VERSION, location);
  const runId = requireString(record, "run_id", location);
  const planId = requireString(record, "plan_id", location);
  const supersedesPlanId = optionalString(record, "supersedes_plan_id", location);
  const lastSequence = requireNonNegativeInt(record, "last_sequence", location);
  const createdAt = requireString(record, "created_at", location);
  const updatedAt = requireString(record, "updated_at", location);
  const pendingRaw = record.pending_approval_waits;
  if (pendingRaw !== undefined && !Array.isArray(pendingRaw)) {
    throw new Error(`${location}.pending_approval_waits must be an array when provided`);
  }

  return {
    schema_version: SNAPSHOT_SCHEMA_VERSION,
    run_id: runId,
    plan_id: planId,
    supersedes_plan_id: supersedesPlanId,
    last_sequence: lastSequence,
    pending_approval_waits: Array.isArray(pendingRaw) ? pendingRaw.map((item, index) => parseApprovalWait(item, `${location}.pending_approval_waits[${index}]`)) : [],
    created_at: createdAt,
    updated_at: updatedAt,
  };
}

export function parseJournalRecord(value: unknown, location: string): DurableJournalRecord {
  const record = assertObjectRecord(value, location);
  requireExactString(record, "schema_version", JOURNAL_SCHEMA_VERSION, location);
  const sequence = requireNonNegativeInt(record, "sequence", location);
  const kind = requireEnum(record, "kind", DURABLE_JOURNAL_KINDS, location);
  const runId = requireString(record, "run_id", location);
  const planId = requireString(record, "plan_id", location);
  const idempotencyKey = requireString(record, "idempotency_key", location);
  const occurredAt = requireString(record, "occurred_at", location);
  const base = {
    schema_version: JOURNAL_SCHEMA_VERSION,
    sequence,
    kind,
    run_id: runId,
    plan_id: planId,
    idempotency_key: idempotencyKey,
    occurred_at: occurredAt,
  } as DurableJournalBase;

  switch (kind) {
    case "run_started":
      return { ...base, kind, run_scope_id: requireString(record, "run_scope_id", location) };
    case "stage_entered":
      return { ...base, kind, stage_id: requireString(record, "stage_id", location) };
    case "step_attempt_started":
      return {
        ...base,
        kind,
        stage_id: optionalString(record, "stage_id", location),
        step_id: requireString(record, "step_id", location),
        step_attempt_id: requireString(record, "step_attempt_id", location),
      };
    case "action_request_issued":
      return {
        ...base,
        kind,
        action_request_id: requireString(record, "action_request_id", location),
        scope_kind: requireEnum(record, "scope_kind", DURABLE_ACTION_SCOPE_KINDS, location),
        scope_id: requireString(record, "scope_id", location),
      };
    case "approval_wait_entered":
      return {
        ...base,
        kind,
        approval_wait_id: requireString(record, "approval_wait_id", location),
        action_request_id: requireString(record, "action_request_id", location),
        binding_kind: requireEnum(record, "binding_kind", APPROVAL_BINDING_KINDS, location),
        bound_action_hash: optionalString(record, "bound_action_hash", location),
        bound_stage_summary_hash: optionalString(record, "bound_stage_summary_hash", location),
        blocked_scope: parseBlockedScope(record.blocked_scope, `${location}.blocked_scope`),
        broker_correlation: parseBrokerCorrelation(record.broker_correlation, `${location}.broker_correlation`),
      };
    case "approval_wait_cleared":
      return {
        ...base,
        kind,
        approval_wait_id: requireString(record, "approval_wait_id", location),
        action_request_id: requireString(record, "action_request_id", location),
        status: requireEnum(record, "status", DURABLE_APPROVAL_CLEAR_STATUSES, location),
      };
    case "gate_attempt_started":
      return {
        ...base,
        kind,
        stage_id: optionalString(record, "stage_id", location),
        gate_id: requireString(record, "gate_id", location),
        gate_attempt_id: requireString(record, "gate_attempt_id", location),
      };
    case "gate_attempt_finished":
      return {
        ...base,
        kind,
        gate_id: requireString(record, "gate_id", location),
        gate_attempt_id: requireString(record, "gate_attempt_id", location),
        outcome: requireEnum(record, "outcome", DURABLE_GATE_ATTEMPT_OUTCOMES, location),
      };
    case "step_attempt_finished":
      return {
        ...base,
        kind,
        step_id: requireString(record, "step_id", location),
        step_attempt_id: requireString(record, "step_attempt_id", location),
        outcome: requireEnum(record, "outcome", DURABLE_STEP_ATTEMPT_OUTCOMES, location),
      };
    case "run_terminal":
      return { ...base, kind, terminal_status: requireEnum(record, "terminal_status", DURABLE_TERMINAL_STATUSES, location) };
    default:
      throw new Error(`${location}.kind has unsupported value ${kind}`);
  }
}

function parseApprovalWait(value: unknown, location: string): DurableApprovalWait {
  const record = assertObjectRecord(value, location);
  const approvalId = requireString(record, "approval_id", location);
  const actionRequestId = requireString(record, "action_request_id", location);
  const runId = requireString(record, "run_id", location);
  const planId = requireString(record, "plan_id", location);
  const bindingKind = requireEnum(record, "binding_kind", APPROVAL_BINDING_KINDS, location);
  const boundActionHash = optionalString(record, "bound_action_hash", location);
  const boundStageSummaryHash = optionalString(record, "bound_stage_summary_hash", location);
  assertApprovalBinding(bindingKind, boundActionHash, boundStageSummaryHash, approvalId);
  requireExactString(record, "status", "pending", location);

  return {
    approval_id: approvalId,
    action_request_id: actionRequestId,
    run_id: runId,
    plan_id: planId,
    binding_kind: bindingKind,
    bound_action_hash: boundActionHash,
    bound_stage_summary_hash: boundStageSummaryHash,
    blocked_scope: parseBlockedScope(record.blocked_scope, `${location}.blocked_scope`),
    broker_correlation: parseBrokerCorrelation(record.broker_correlation, `${location}.broker_correlation`),
    status: "pending",
    wait_entered_idempotency_key: requireString(record, "wait_entered_idempotency_key", location),
    entered_at: requireString(record, "entered_at", location),
    updated_at: requireString(record, "updated_at", location),
  };
}

function parseBlockedScope(value: unknown, location: string) {
  const record = assertObjectRecord(value, location);
  return parseBlockedScopeFields(record, location, optionalString, requireEnum, requireString);
}

function parseBrokerCorrelation(value: unknown, location: string): DurableApprovalBrokerCorrelation {
  if (value === undefined) {
    return {};
  }
  const record = assertObjectRecord(value, location);
  return sanitizeBrokerCorrelation({
    request_id: optionalString(record, "request_id", location),
    action_request_id: optionalString(record, "action_request_id", location),
    approval_request_id: optionalString(record, "approval_request_id", location),
    operation_id: optionalString(record, "operation_id", location),
    parent_operation_id: optionalString(record, "parent_operation_id", location),
  });
}

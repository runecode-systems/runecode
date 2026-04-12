import {
  type DurableActionScopeKind,
  type DurableApprovalClearStatus,
  type DurableApprovalWait,
  type DurableJournalKind,
  type DurableJournalRecord,
  type DurableReplayInternal,
  type DurableReplayState,
  type DurableSnapshot,
  DurableReplayError,
  cloneApprovalWait,
  cloneBlockedScope,
  cloneBrokerCorrelation,
  lastJournalOccurredAt,
} from "./types.ts";
import { blockedScopeToDurableScope } from "./helpers.ts";

export function replayDurableState(snapshot: DurableSnapshot, journal: DurableJournalRecord[]): DurableReplayState {
  const internal = replayDurableStateInternal(snapshot, journal);
  return {
    run_id: internal.run_id,
    plan_id: internal.plan_id,
    last_sequence: internal.last_sequence,
    started: internal.started,
    terminal: internal.terminal,
    scheduler: internal.scheduler,
    waits: internal.waits,
    attempts: internal.attempts,
    actions: internal.actions,
  };
}

export function replayDurableStateInternal(snapshot: DurableSnapshot, journal: DurableJournalRecord[]): DurableReplayInternal {
  const enteredStageIds = new Set<string>();
  const issuedActionIds = new Set<string>();
  const resolvedApprovalWaits = new Map<string, DurableApprovalClearStatus>();
  const activeStepAttemptIds = new Set<string>();
  const finishedStepAttemptIds = new Set<string>();
  const activeGateAttemptIds = new Set<string>();
  const finishedGateAttemptIds = new Set<string>();
  const actionScopeById = new Map<string, { scope_kind: DurableActionScopeKind; scope_id: string }>();
  const pendingApprovalWaitsById = new Map<string, DurableApprovalWait>();

  let started = false;
  let terminal: DurableReplayState["terminal"] = null;
  let currentStageId: string | null = null;

  for (const [index, record] of journal.entries()) {
    const expectedSequence = index + 1;
    if (record.sequence !== expectedSequence) {
      throw new DurableReplayError(`durable replay sequence ${record.sequence} must equal ${expectedSequence}`);
    }
    if (record.run_id !== snapshot.run_id || record.plan_id !== snapshot.plan_id) {
      throw new DurableReplayError(`durable replay identity mismatch at sequence ${record.sequence}`);
    }
    if (terminal) {
      throw new DurableReplayError(`durable replay found record after run_terminal at sequence ${record.sequence}`);
    }

    switch (record.kind) {
      case "run_started":
        if (started) {
          throw new DurableReplayError(`durable replay found duplicate run_started at sequence ${record.sequence}`);
        }
        started = true;
        break;
      case "stage_entered":
        requireStarted(started, record.sequence, record.kind);
        enteredStageIds.add(record.stage_id);
        currentStageId = record.stage_id;
        break;
      case "step_attempt_started":
        requireStarted(started, record.sequence, record.kind);
        assertAttemptNotSeen(record.step_attempt_id, activeStepAttemptIds, finishedStepAttemptIds, record.sequence, "step_attempt_id");
        activeStepAttemptIds.add(record.step_attempt_id);
        break;
      case "action_request_issued":
        requireStarted(started, record.sequence, record.kind);
        if (issuedActionIds.has(record.action_request_id)) {
          throw new DurableReplayError(`durable replay duplicate action_request_id ${record.action_request_id} at sequence ${record.sequence}`);
        }
        issuedActionIds.add(record.action_request_id);
        actionScopeById.set(record.action_request_id, { scope_kind: record.scope_kind, scope_id: record.scope_id });
        break;
      case "approval_wait_entered": {
        requireStarted(started, record.sequence, record.kind);
        if (resolvedApprovalWaits.has(record.approval_wait_id) || pendingApprovalWaitsById.has(record.approval_wait_id)) {
          throw new DurableReplayError(`durable replay duplicate approval_wait_id ${record.approval_wait_id} at sequence ${record.sequence}`);
        }
        if (!issuedActionIds.has(record.action_request_id)) {
          throw new DurableReplayError(`durable replay unknown action_request_id ${record.action_request_id} at sequence ${record.sequence}`);
        }

        const expectedScope = actionScopeById.get(record.action_request_id);
        const actualScope = blockedScopeToDurableScope(record.blocked_scope, record.run_id, record.approval_wait_id);
        if (!expectedScope || expectedScope.scope_kind !== actualScope.scope_kind || expectedScope.scope_id !== actualScope.scope_id) {
          throw new DurableReplayError(`durable replay scope mismatch for action_request_id ${record.action_request_id} at sequence ${record.sequence}`);
        }

        pendingApprovalWaitsById.set(record.approval_wait_id, {
          approval_id: record.approval_wait_id,
          action_request_id: record.action_request_id,
          run_id: record.run_id,
          plan_id: record.plan_id,
          binding_kind: record.binding_kind,
          bound_action_hash: record.bound_action_hash,
          bound_stage_summary_hash: record.bound_stage_summary_hash,
          blocked_scope: cloneBlockedScope(record.blocked_scope),
          broker_correlation: cloneBrokerCorrelation(record.broker_correlation),
          status: "pending",
          wait_entered_idempotency_key: record.idempotency_key,
          entered_at: record.occurred_at,
          updated_at: record.occurred_at,
        });
        break;
      }
      case "approval_wait_cleared": {
        requireStarted(started, record.sequence, record.kind);
        const pendingWait = pendingApprovalWaitsById.get(record.approval_wait_id);
        if (!pendingWait) {
          throw new DurableReplayError(`durable replay clearing unknown approval_wait_id ${record.approval_wait_id} at sequence ${record.sequence}`);
        }
        if (pendingWait.action_request_id !== record.action_request_id) {
          throw new DurableReplayError(`durable replay action_request_id mismatch for approval_wait_id ${record.approval_wait_id} at sequence ${record.sequence}`);
        }
        pendingApprovalWaitsById.delete(record.approval_wait_id);
        resolvedApprovalWaits.set(record.approval_wait_id, record.status);
        break;
      }
      case "gate_attempt_started":
        requireStarted(started, record.sequence, record.kind);
        assertAttemptNotSeen(record.gate_attempt_id, activeGateAttemptIds, finishedGateAttemptIds, record.sequence, "gate_attempt_id");
        activeGateAttemptIds.add(record.gate_attempt_id);
        break;
      case "gate_attempt_finished":
        requireStarted(started, record.sequence, record.kind);
        assertAttemptActive(record.gate_attempt_id, activeGateAttemptIds, record.sequence, "gate_attempt_id");
        activeGateAttemptIds.delete(record.gate_attempt_id);
        finishedGateAttemptIds.add(record.gate_attempt_id);
        break;
      case "step_attempt_finished":
        requireStarted(started, record.sequence, record.kind);
        assertAttemptActive(record.step_attempt_id, activeStepAttemptIds, record.sequence, "step_attempt_id");
        activeStepAttemptIds.delete(record.step_attempt_id);
        finishedStepAttemptIds.add(record.step_attempt_id);
        break;
      case "run_terminal":
        requireStarted(started, record.sequence, record.kind);
        if (pendingApprovalWaitsById.size > 0 || activeStepAttemptIds.size > 0 || activeGateAttemptIds.size > 0) {
          throw new DurableReplayError(`durable replay found run_terminal with active waits or attempts at sequence ${record.sequence}`);
        }
        terminal = { sequence: record.sequence, terminal_status: record.terminal_status };
        break;
      default:
        throw new DurableReplayError(`durable replay encountered unsupported kind ${(record as { kind: string }).kind}`);
    }
  }

  if (snapshot.last_sequence > journal.length) {
    throw new DurableReplayError(`durable snapshot last_sequence ${snapshot.last_sequence} exceeds journal length ${journal.length}`);
  }
  if (journal.length > 0 && !started) {
    throw new DurableReplayError("durable replay journal is non-empty but missing run_started");
  }

  return {
    run_id: snapshot.run_id,
    plan_id: snapshot.plan_id,
    last_sequence: journal.length,
    started,
    terminal,
    scheduler: {
      entered_stage_ids: [...enteredStageIds],
      current_stage_id: currentStageId,
    },
    waits: {
      pending_approval_waits: [...pendingApprovalWaitsById.keys()],
      resolved_approval_waits: [...resolvedApprovalWaits.entries()].map(([approval_id, status]) => ({ approval_id, status })),
    },
    attempts: {
      active_step_attempt_ids: [...activeStepAttemptIds],
      finished_step_attempt_ids: [...finishedStepAttemptIds],
      active_gate_attempt_ids: [...activeGateAttemptIds],
      finished_gate_attempt_ids: [...finishedGateAttemptIds],
    },
    actions: {
      issued_action_request_ids: [...issuedActionIds],
    },
    pending_approval_waits_by_id: pendingApprovalWaitsById,
    has_run_started: started,
  };
}

export function healSnapshotFromJournal(snapshot: DurableSnapshot, journal: DurableJournalRecord[]): DurableSnapshot {
  const replay = replayDurableStateInternal(snapshot, journal);
  const pendingApprovalWaits = [...replay.pending_approval_waits_by_id.values()].map((wait) => cloneApprovalWait(wait));
  return {
    ...snapshot,
    last_sequence: journal.length,
    pending_approval_waits: pendingApprovalWaits,
    updated_at: lastJournalOccurredAt(journal) ?? snapshot.updated_at,
  };
}

export function snapshotNeedsRewrite(current: DurableSnapshot, next: DurableSnapshot): boolean {
  return current.last_sequence !== next.last_sequence
    || current.updated_at !== next.updated_at
    || JSON.stringify(current.pending_approval_waits) !== JSON.stringify(next.pending_approval_waits);
}

function requireStarted(started: boolean, sequence: number, kind: DurableJournalKind): void {
  if (!started) {
    throw new DurableReplayError(`durable replay found ${kind} before run_started at sequence ${sequence}`);
  }
}

function assertAttemptNotSeen(
  attemptId: string,
  activeAttempts: Set<string>,
  finishedAttempts: Set<string>,
  sequence: number,
  label: string,
): void {
  if (activeAttempts.has(attemptId) || finishedAttempts.has(attemptId)) {
    throw new DurableReplayError(`durable replay duplicate ${label} ${attemptId} at sequence ${sequence}`);
  }
}

function assertAttemptActive(attemptId: string, activeAttempts: Set<string>, sequence: number, label: string): void {
  if (!activeAttempts.has(attemptId)) {
    throw new DurableReplayError(`durable replay finishing inactive ${label} ${attemptId} at sequence ${sequence}`);
  }
}

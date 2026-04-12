import {
  DURABLE_BLOCKED_ACTION_KINDS,
  BLOCKED_SCOPE_KINDS,
  type ApprovalBindingKind,
  type DurableActionScopeKind,
  type DurableApprovalBlockedWorkScope,
  type DurableApprovalBrokerCorrelation,
  type DurableApprovalWait,
  InvalidApprovalWaitError,
  trimOptional,
  trimRequired,
} from "./types.ts";

export function parseBlockedScopeFields(record: Record<string, unknown>, location: string, optionalString: (record: Record<string, unknown>, field: string, location: string) => string | undefined, requireEnum: <T extends string>(record: Record<string, unknown>, field: string, allowed: readonly T[], location: string) => T, requireString: (record: Record<string, unknown>, field: string, location: string) => string): DurableApprovalBlockedWorkScope {
  return sanitizeBlockedScope(
    {
      scope_kind: requireEnum(record, "scope_kind", BLOCKED_SCOPE_KINDS, location),
      workspace_id: optionalString(record, "workspace_id", location),
      run_id: optionalString(record, "run_id", location),
      stage_id: optionalString(record, "stage_id", location),
      step_id: optionalString(record, "step_id", location),
      role_instance_id: optionalString(record, "role_instance_id", location),
      action_kind: requireEnum(record, "action_kind", DURABLE_BLOCKED_ACTION_KINDS, location),
    },
    location,
  );
}

export function sanitizeBlockedScope(scope: DurableApprovalBlockedWorkScope, location: string): DurableApprovalBlockedWorkScope {
  const actionKindRaw = trimRequired(scope.action_kind, `${location}.action_kind`);
  if (!DURABLE_BLOCKED_ACTION_KINDS.includes(actionKindRaw as (typeof DURABLE_BLOCKED_ACTION_KINDS)[number])) {
    throw new InvalidApprovalWaitError(`${location}.action_kind ${actionKindRaw} is invalid`);
  }
  const actionKind = actionKindRaw as DurableApprovalBlockedWorkScope["action_kind"];

  const sanitized: DurableApprovalBlockedWorkScope = {
    scope_kind: scope.scope_kind,
    workspace_id: trimOptional(scope.workspace_id),
    run_id: trimOptional(scope.run_id),
    stage_id: trimOptional(scope.stage_id),
    step_id: trimOptional(scope.step_id),
    role_instance_id: trimOptional(scope.role_instance_id),
    action_kind: actionKind,
  };
  if (!BLOCKED_SCOPE_KINDS.includes(sanitized.scope_kind)) {
    throw new InvalidApprovalWaitError(`${location}.scope_kind ${scope.scope_kind} is invalid`);
  }
  if (sanitized.scope_kind === "workspace" && !sanitized.workspace_id) {
    throw new InvalidApprovalWaitError(`${location}.workspace_id is required for workspace-scoped approval waits`);
  }
  if (sanitized.scope_kind === "run" && !sanitized.run_id) {
    throw new InvalidApprovalWaitError(`${location}.run_id is required for run-scoped approval waits`);
  }
  if (sanitized.scope_kind === "stage" && !sanitized.stage_id) {
    throw new InvalidApprovalWaitError(`${location}.stage_id is required for stage-scoped approval waits`);
  }
  if (sanitized.scope_kind === "step" && !sanitized.step_id) {
    throw new InvalidApprovalWaitError(`${location}.step_id is required for step-scoped approval waits`);
  }
  return sanitized;
}

export function sanitizeBrokerCorrelation(correlation: DurableApprovalBrokerCorrelation): DurableApprovalBrokerCorrelation {
  return {
    request_id: trimOptional(correlation.request_id),
    action_request_id: trimOptional(correlation.action_request_id),
    approval_request_id: trimOptional(correlation.approval_request_id),
    operation_id: trimOptional(correlation.operation_id),
    parent_operation_id: trimOptional(correlation.parent_operation_id),
  };
}

export function blockedScopeToDurableScope(
  blockedScope: DurableApprovalBlockedWorkScope,
  runId: string,
  approvalId: string,
): { scope_kind: DurableActionScopeKind; scope_id: string } {
  switch (blockedScope.scope_kind) {
    case "run":
      return { scope_kind: "run", scope_id: blockedScope.run_id ?? runId };
    case "stage":
      return { scope_kind: "stage", scope_id: blockedScope.stage_id ?? `stage:${approvalId}` };
    case "step":
      return { scope_kind: "step_attempt", scope_id: blockedScope.step_id ?? `step:${approvalId}` };
    case "action_kind":
      return { scope_kind: "stage", scope_id: blockedScope.stage_id ?? blockedScope.action_kind };
    case "workspace":
    default:
      return { scope_kind: "run", scope_id: blockedScope.run_id ?? runId };
  }
}

export function approvalWaitBindingsMatch(
  wait: Pick<DurableApprovalWait, "binding_kind" | "bound_action_hash" | "bound_stage_summary_hash">,
  bindingKind: ApprovalBindingKind,
  boundActionHash?: string,
  boundStageSummaryHash?: string,
): boolean {
  assertApprovalBinding(wait.binding_kind, wait.bound_action_hash, wait.bound_stage_summary_hash, "existing_approval_wait");
  assertApprovalBinding(bindingKind, boundActionHash, boundStageSummaryHash, "incoming_approval_wait");
  return wait.binding_kind === bindingKind
    && wait.bound_action_hash === boundActionHash
    && wait.bound_stage_summary_hash === boundStageSummaryHash;
}

export function assertApprovalBinding(
  bindingKind: ApprovalBindingKind,
  boundActionHash: string | undefined,
  boundStageSummaryHash: string | undefined,
  approvalId: string,
): void {
  if (bindingKind === "exact_action") {
    if (!boundActionHash || boundStageSummaryHash) {
      throw new InvalidApprovalWaitError(`approval wait ${approvalId} must include only bound_action_hash for exact_action`);
    }
    return;
  }
  if (!boundStageSummaryHash || boundActionHash) {
    throw new InvalidApprovalWaitError(`approval wait ${approvalId} must include only bound_stage_summary_hash for stage_sign_off`);
  }
}

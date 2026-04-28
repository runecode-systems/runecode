/**
 * Minimal plan-bound scheduler skeleton.
 *
 * This scheduler only surfaces immutable RunPlan entries in listed order and
 * keeps no local workflow-planning authority.
 */

import { DURABLE_BLOCKED_ACTION_KINDS, type DurableApprovalWait } from "./durable-state.ts";
import type { RunnerPlan, RunnerPlanEntry } from "./run-plan.ts";

export type ScheduledWorkItem = {
  index: number;
  entry: RunnerPlanEntry;
};

export type SchedulerState = {
  pending_approval_waits: DurableApprovalWait[];
  completed_entry_ids?: string[];
};

export class PlanScheduler {
  listPlannedWork(plan: RunnerPlan, state?: SchedulerState): ScheduledWorkItem[] {
    const blockedScopes = (state?.pending_approval_waits ?? []).map((wait) => wait.blocked_scope);
    const completedEntryIDs = new Set(state?.completed_entry_ids ?? []);
    return [...plan.entries]
      .sort((left, right) => (left.order_index ?? Number.MAX_SAFE_INTEGER) - (right.order_index ?? Number.MAX_SAFE_INTEGER))
      .filter((entry) => this.isDependencyEligible(entry, completedEntryIDs))
      .map((entry, index) => ({ index, entry }))
      .filter((item) => !this.matchesAnyBlockedScope(item.entry, plan.run_id, blockedScopes));
  }

  private isDependencyEligible(entry: RunnerPlanEntry, completedEntryIDs: Set<string>): boolean {
    return entry.depends_on_entry_ids.every((entryID) => completedEntryIDs.has(entryID));
  }

  private matchesAnyBlockedScope(entry: RunnerPlanEntry, runId: string, blockedScopes: DurableApprovalWait["blocked_scope"][]): boolean {
    return blockedScopes.some((scope) => this.matchesBlockedScope(entry, runId, scope));
  }

  private matchesBlockedScope(entry: RunnerPlanEntry, runId: string, scope: DurableApprovalWait["blocked_scope"]): boolean {
    if (scope.run_id !== undefined && scope.run_id !== runId) {
      return false;
    }
    if (scope.role_instance_id !== undefined && entry.role_instance_id !== scope.role_instance_id) {
      return false;
    }

    switch (scope.scope_kind) {
      case "run":
        return true;
      case "stage":
        return scope.stage_id !== undefined && entry.stage_id === scope.stage_id;
      case "step":
        return scope.step_id !== undefined && entry.step_id === scope.step_id;
      case "action_kind":
        return this.matchesBlockedActionKind(entry, scope.action_kind);
      case "workspace":
        return true;
      default:
        return false;
    }
  }

  private matchesBlockedActionKind(entry: RunnerPlanEntry, actionKind: DurableApprovalWait["blocked_scope"]["action_kind"]): boolean {
    switch (actionKind) {
      case "action_gate_override":
      case "stage_summary_sign_off":
        return entry.entry_kind === "gate";
      default:
        throw new Error(`unsupported blocked action kind ${actionKind}; expected one of ${DURABLE_BLOCKED_ACTION_KINDS.join(", ")}`);
    }
  }
}

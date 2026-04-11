/**
 * Minimal plan-bound scheduler skeleton.
 *
 * This scheduler only surfaces immutable RunPlan entries in listed order and
 * keeps no local workflow-planning authority.
 */

import type { RunnerPlan, RunnerPlanEntry } from "./run-plan.ts";

export type ScheduledWorkItem = {
  index: number;
  entry: RunnerPlanEntry;
};

export class PlanScheduler {
  listPlannedWork(plan: RunnerPlan): ScheduledWorkItem[] {
    return [...plan.entries]
      .sort((left, right) => (left.order_index ?? Number.MAX_SAFE_INTEGER) - (right.order_index ?? Number.MAX_SAFE_INTEGER))
      .map((entry, index) => ({ index, entry }));
  }
}

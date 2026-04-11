/**
 * Thin runner kernel composition root.
 *
 * Startup loads a RunPlan, binds durable state to plan identity, and exposes
 * plan-bound scheduled work with no local planning/authorization semantics.
 */

import type { FileDurableStateStore } from "./durable-state.ts";
import { PlanScheduler, type ScheduledWorkItem } from "./scheduler.ts";
import type { RunnerPlan, RunPlanLoader } from "./run-plan.ts";

export type RunnerKernelOptions = {
  planLoader: RunPlanLoader;
  durableStateStore: FileDurableStateStore;
  scheduler?: PlanScheduler;
};

export class RunnerKernel {
  private readonly options: RunnerKernelOptions;

  private readonly scheduler: PlanScheduler;

  constructor(options: RunnerKernelOptions) {
    this.options = options;
    this.scheduler = options.scheduler ?? new PlanScheduler();
  }

  async initializeFromPlanFile(planFilePath: string): Promise<{ plan: RunnerPlan; work: ScheduledWorkItem[] }> {
    const plan = await this.options.planLoader.loadFromFile(planFilePath);
    await this.options.durableStateStore.bindPlanIdentity(this.options.planLoader.identityOf(plan));
    const work = this.scheduler.listPlannedWork(plan);
    return { plan, work };
  }
}

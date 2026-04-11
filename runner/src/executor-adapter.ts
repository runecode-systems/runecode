/**
 * Executor adapter seams for runner entry dispatch.
 *
 * Adapters are looked up by explicit plan entry kind and are intentionally
 * policy-agnostic: authorization remains broker-owned.
 */

import type { RunnerPlanEntry } from "./run-plan.ts";

export type ExecutionOutcome = {
  status: "ok" | "failed";
  details?: Record<string, unknown>;
};

export type ExecutorAdapter = {
  execute(entry: RunnerPlanEntry): Promise<ExecutionOutcome>;
};

export class ExecutorAdapterRegistry {
  private readonly adaptersByKind = new Map<string, ExecutorAdapter>();

  register(entryKind: string, adapter: ExecutorAdapter): void {
    this.adaptersByKind.set(entryKind, adapter);
  }

  resolve(entryKind: string): ExecutorAdapter | null {
    return this.adaptersByKind.get(entryKind) ?? null;
  }
}

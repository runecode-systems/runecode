/**
 * Executor adapter seams for runner entry dispatch.
 *
 * Adapters are looked up by explicit plan entry kind and are intentionally
 * policy-agnostic: authorization remains broker-owned.
 */

import type { DependencyCacheHandoffMetadata, PlanBoundExecutionIdentity } from "./contracts.ts";
import type { RunnerPlanEntry } from "./run-plan.ts";

export type ExecutionOutcome = {
  status: "ok" | "failed";
  details?: Record<string, unknown>;
  failure_reason_code?: string;
};

export type ExecutorAdapter = {
  execute(input: {
    identity: PlanBoundExecutionIdentity;
    entry: RunnerPlanEntry;
    dependency_cache_handoffs: DependencyCacheHandoffMetadata[];
  }): Promise<ExecutionOutcome>;
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

export class MinimalGateExecutorAdapter implements ExecutorAdapter {
  async execute(input: {
    identity: PlanBoundExecutionIdentity;
    entry: RunnerPlanEntry;
    dependency_cache_handoffs: DependencyCacheHandoffMetadata[];
  }): Promise<ExecutionOutcome> {
    return {
      status: "ok",
      details: {
        executor_binding_id: input.entry.executor_binding_id,
        gate_id: gateString(input.entry.gate.gate_id),
        gate_kind: gateString(input.entry.gate.gate_kind),
        handoff_count: input.dependency_cache_handoffs.length,
        step_id: input.identity.step_id,
      },
    };
  }
}

function gateString(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

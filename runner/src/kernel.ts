/**
 * Thin runner kernel composition root.
 *
 * Startup loads a RunPlan, binds durable state to plan identity, and exposes
 * plan-bound scheduled work with no local planning/authorization semantics.
 */

import {
  InvalidApprovalWaitError,
  PlanIdentityMismatchError,
  type DurableApprovalWait,
  type FileDurableStateStore,
} from "./durable-state.ts";
import { PlanScheduler, type ScheduledWorkItem } from "./scheduler.ts";
import type { DependencyCacheHandoffRequirement, RunnerPlan, RunnerPlanEntry, RunPlanLoader } from "./run-plan.ts";
import { DurableRuntimeSeam, type RunnerRuntimeSeam } from "./runtime-seam.ts";
import { NoopRunnerBrokerClient, type RunnerBrokerClient } from "./broker-client.ts";
import type {
  DependencyCacheHandoffMetadata,
  PlanBoundExecutionIdentity,
  RunnerCheckpointReport,
  RunnerResultReport,
} from "./contracts.ts";
import {
  DEPENDENCY_CACHE_HANDOFF_REQUEST_SCHEMA_ID,
  RUNNER_CHECKPOINT_REPORT_SCHEMA_ID,
  RUNNER_CONTRACT_SCHEMA_VERSION,
  RUNNER_RESULT_REPORT_SCHEMA_ID,
} from "./contracts.ts";

export type RunnerKernelOptions = {
  planLoader: RunPlanLoader;
  durableStateStore: FileDurableStateStore;
  scheduler?: PlanScheduler;
  runtimeSeam?: RunnerRuntimeSeam;
  approvalWaitResolver?: ApprovalWaitResolver;
  brokerClient?: RunnerBrokerClient;
};

export type ApprovalWaitResolution = {
  approval_id: string;
  run_id: string;
  plan_id: string;
  status: "pending" | "approved" | "denied" | "expired" | "superseded" | "cancelled" | "consumed";
  binding_kind: DurableApprovalWait["binding_kind"];
  bound_action_hash?: string;
  bound_stage_summary_hash?: string;
};

export type ClearedApprovalWait = {
  approval_id: string;
  status: Exclude<ApprovalWaitResolution["status"], "pending">;
};

export type ApprovalWaitResolver = {
  resolve(wait: DurableApprovalWait): Promise<ApprovalWaitResolution>;
};

export type KernelExecutionContext = {
  identity: PlanBoundExecutionIdentity;
  runtime: RunnerRuntimeSeam;
  dependency_cache_handoffs: DependencyCacheHandoffMetadata[];
};

export type KernelExecutionModule = {
  name: string;
  run(context: KernelExecutionContext): Promise<void>;
};

export class RunnerKernel {
  private readonly options: RunnerKernelOptions;

  private readonly scheduler: PlanScheduler;

  private readonly runtimeSeam: RunnerRuntimeSeam;

  private readonly approvalWaitResolver: ApprovalWaitResolver | undefined;

  private readonly brokerClient: RunnerBrokerClient;

  constructor(options: RunnerKernelOptions) {
    this.options = options;
    this.scheduler = options.scheduler ?? new PlanScheduler();
    this.runtimeSeam = options.runtimeSeam ?? new DurableRuntimeSeam(options.durableStateStore);
    this.approvalWaitResolver = options.approvalWaitResolver;
    this.brokerClient = options.brokerClient ?? new NoopRunnerBrokerClient();
  }

  async initializeFromPlanFile(planFilePath: string): Promise<{ plan: RunnerPlan; work: ScheduledWorkItem[] }> {
    const plan = await this.options.planLoader.loadFromFile(planFilePath);
    await this.options.durableStateStore.bindPlanIdentity(this.options.planLoader.identityOf(plan));
    const pendingApprovalWaits = await this.options.durableStateStore.listPendingApprovalWaits();
    const work = this.scheduler.listPlannedWork(plan, { pending_approval_waits: pendingApprovalWaits });
    return { plan, work };
  }

  async resumeApprovalWaits(): Promise<{ pending_waits: DurableApprovalWait[]; cleared_waits: ClearedApprovalWait[] }> {
    if (!this.approvalWaitResolver) {
      throw new Error("approval wait resolver is not configured");
    }

    const pending = await this.options.durableStateStore.listPendingApprovalWaits();
    const cleared: ClearedApprovalWait[] = [];

    for (const wait of pending) {
      const resolution = await this.approvalWaitResolver.resolve(wait);
      this.assertResumeResolution(wait, resolution);

      if (resolution.status === "pending") {
        continue;
      }

      await this.options.durableStateStore.resolveApprovalWait({
        approval_id: wait.approval_id,
        run_id: wait.run_id,
        plan_id: wait.plan_id,
        binding_kind: wait.binding_kind,
        bound_action_hash: wait.bound_action_hash,
        bound_stage_summary_hash: wait.bound_stage_summary_hash,
        status: resolution.status,
        idempotency_key: `approval_wait_resolved:${wait.approval_id}:${resolution.status}`,
      });
      cleared.push({ approval_id: wait.approval_id, status: resolution.status });
    }

    return {
      pending_waits: await this.options.durableStateStore.listPendingApprovalWaits(),
      cleared_waits: cleared,
    };
  }

  executionIdentityFromPlan(plan: RunnerPlan): PlanBoundExecutionIdentity {
    return {
      run_id: plan.run_id,
      plan_id: plan.plan_id,
    };
  }

  async composeModules(
    identity: PlanBoundExecutionIdentity,
    modules: KernelExecutionModule[],
    dependencyCacheHandoffs?: DependencyCacheHandoffRequirement[],
  ): Promise<void> {
    const resolvedHandoffs = await this.resolveDependencyCacheHandoffs(identity, dependencyCacheHandoffs ?? []);
    for (const module of modules) {
      await module.run({
        identity,
        runtime: this.runtimeSeam,
        dependency_cache_handoffs: resolvedHandoffs,
      });
    }
  }

  async composeEntryModules(
    identity: PlanBoundExecutionIdentity,
    entry: RunnerPlanEntry,
    modules: KernelExecutionModule[],
  ): Promise<void> {
    return this.composeModules(identity, modules, entry.dependency_cache_handoffs);
  }

  private async resolveDependencyCacheHandoffs(
    identity: PlanBoundExecutionIdentity,
    requirements: DependencyCacheHandoffRequirement[],
  ): Promise<DependencyCacheHandoffMetadata[]> {
    const resolved: DependencyCacheHandoffMetadata[] = [];
    for (const requirement of requirements) {
      const response = await this.brokerClient.requestDependencyCacheHandoff({
        schema_id: DEPENDENCY_CACHE_HANDOFF_REQUEST_SCHEMA_ID,
        schema_version: RUNNER_CONTRACT_SCHEMA_VERSION,
        request_id: this.dependencyCacheHandoffRequestID(identity, requirement),
        request_digest: this.digestObject(requirement.request_digest),
        consumer_role: requirement.consumer_role,
      });
      if (!response.found || !response.handoff) {
        throw new Error(`required dependency cache handoff not found for ${requirement.request_digest}`);
      }
      resolved.push(response.handoff);
    }
    return resolved;
  }

  private dependencyCacheHandoffRequestID(identity: PlanBoundExecutionIdentity, requirement: DependencyCacheHandoffRequirement): string {
    const digestSuffix = requirement.request_digest.slice(-12);
    return `dependency-handoff:${identity.run_id.slice(0, 24)}:${digestSuffix}`;
  }

  private digestObject(digestIdentity: string): { hash_alg: "sha256"; hash: string } {
    const [hashAlg, hash] = digestIdentity.split(":", 2);
    if (hashAlg !== "sha256" || !hash || !/^[a-f0-9]{64}$/.test(hash)) {
      throw new Error(`dependency handoff digest identity must be sha256:<hex>, got ${digestIdentity}`);
    }
    return { hash_alg: "sha256", hash };
  }

  checkpointReport(input: {
    identity: PlanBoundExecutionIdentity;
    checkpoint_code: string;
    idempotency_key: string;
    lifecycle_state: RunnerCheckpointReport["lifecycle_state"];
    occurred_at?: string;
  }): RunnerCheckpointReport {
    return {
      schema_id: RUNNER_CHECKPOINT_REPORT_SCHEMA_ID,
      schema_version: RUNNER_CONTRACT_SCHEMA_VERSION,
      lifecycle_state: input.lifecycle_state,
      checkpoint_code: input.checkpoint_code,
      occurred_at: input.occurred_at ?? new Date().toISOString(),
      idempotency_key: input.idempotency_key,
      stage_id: input.identity.stage_id,
      step_id: input.identity.step_id,
      role_instance_id: input.identity.role_instance_id,
      stage_attempt_id: input.identity.stage_attempt_id,
      step_attempt_id: input.identity.step_attempt_id,
      gate_attempt_id: input.identity.gate_attempt_id,
    };
  }

  resultReport(input: {
    identity: PlanBoundExecutionIdentity;
    result_code: string;
    idempotency_key: string;
    lifecycle_state: RunnerResultReport["lifecycle_state"];
    occurred_at?: string;
  }): RunnerResultReport {
    return {
      schema_id: RUNNER_RESULT_REPORT_SCHEMA_ID,
      schema_version: RUNNER_CONTRACT_SCHEMA_VERSION,
      lifecycle_state: input.lifecycle_state,
      result_code: input.result_code,
      occurred_at: input.occurred_at ?? new Date().toISOString(),
      idempotency_key: input.idempotency_key,
      stage_id: input.identity.stage_id,
      step_id: input.identity.step_id,
      role_instance_id: input.identity.role_instance_id,
      stage_attempt_id: input.identity.stage_attempt_id,
      step_attempt_id: input.identity.step_attempt_id,
      gate_attempt_id: input.identity.gate_attempt_id,
    };
  }

  private assertResumeResolution(wait: DurableApprovalWait, resolution: ApprovalWaitResolution): void {
    if (resolution.approval_id !== wait.approval_id) {
      throw new InvalidApprovalWaitError(`resume resolution approval_id ${resolution.approval_id} does not match pending wait ${wait.approval_id}`);
    }
    if (resolution.binding_kind !== wait.binding_kind) {
      throw new InvalidApprovalWaitError(`resume resolution binding kind ${resolution.binding_kind} does not match pending wait ${wait.binding_kind}`);
    }
    if (resolution.bound_action_hash !== wait.bound_action_hash) {
      throw new InvalidApprovalWaitError(`resume resolution bound_action_hash mismatch for approval ${wait.approval_id}`);
    }
    if (resolution.bound_stage_summary_hash !== wait.bound_stage_summary_hash) {
      throw new InvalidApprovalWaitError(`resume resolution bound_stage_summary_hash mismatch for approval ${wait.approval_id}`);
    }
    if (resolution.run_id !== wait.run_id || resolution.plan_id !== wait.plan_id) {
      throw new PlanIdentityMismatchError(
        `resume resolution binding ${resolution.run_id}/${resolution.plan_id} does not match pending wait ${wait.run_id}/${wait.plan_id}`,
      );
    }
  }

}

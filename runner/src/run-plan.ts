/**
 * RunPlan loading and validation for runner kernel startup.
 *
 * The runner treats RunPlan as immutable broker-compiled input and fails
 * closed on shape/schema/identity violations.
 */

import { readFile } from "node:fs/promises";
import { ProtocolSchemaBundle } from "./protocol-schema-bundle.ts";

export const RUN_PLAN_SCHEMA_ID = "runecode.protocol.v0.RunPlan";
const RUN_PLAN_SCHEMA_VERSION = "0.3.0";

export type RunnerPlanIdentity = {
  run_id: string;
  plan_id: string;
  supersedes_plan_id?: string;
};

export type RunnerPlanEntry = {
  entry_id: string;
  entry_kind: string;
  order_index?: number;
  executor_ref?: string;
  gate_id?: string;
  stage_id?: string;
  step_id?: string;
  role_instance_id?: string;
  dependency_cache_handoffs?: DependencyCacheHandoffRequirement[];
  [key: string]: unknown;
};

export type DependencyCacheHandoffRequirement = {
  request_digest: string;
  consumer_role: string;
  required: true;
};

export type RunnerDependencyEdge = {
  upstream_step_id: string;
  downstream_step_id: string;
  dependency_kind: "step_completed";
};

export type RunnerPlan = {
  schema_id: typeof RUN_PLAN_SCHEMA_ID;
  schema_version: string;
  run_id: string;
  plan_id: string;
  supersedes_plan_id?: string;
  gate_definitions: Array<Record<string, unknown>>;
  dependency_edges: RunnerDependencyEdge[];
  entries: RunnerPlanEntry[];
  [key: string]: unknown;
};

export class RunPlanLoader {
  private readonly schemaBundle: ProtocolSchemaBundle;

  constructor(schemaBundle: ProtocolSchemaBundle) {
    this.schemaBundle = schemaBundle;
  }

  async loadFromFile(filePath: string): Promise<RunnerPlan> {
    const text = await readFile(filePath, "utf8");
    return this.loadFromJsonText(text);
  }

  loadFromJsonText(jsonText: string): RunnerPlan {
    let parsed: unknown;
    try {
      parsed = JSON.parse(jsonText);
    } catch (error) {
      throw new Error(`RunPlan parse failed: ${(error as Error).message}`);
    }

    return this.loadFromUnknown(parsed);
  }

  loadFromUnknown(value: unknown): RunnerPlan {
    const candidate = this.assertRecord(value);

    const schemaId = this.requireString(candidate, "schema_id");
    const schemaVersion = this.requireString(candidate, "schema_version");
    if (schemaId !== RUN_PLAN_SCHEMA_ID) {
      throw new Error(`RunPlan schema_id must be ${RUN_PLAN_SCHEMA_ID}`);
    }
    if (schemaVersion !== RUN_PLAN_SCHEMA_VERSION) {
      throw new Error(`RunPlan schema_version must be ${RUN_PLAN_SCHEMA_VERSION}`);
    }

    const schemaValidation = this.schemaBundle.validateByRuntimeKey(schemaId, schemaVersion, candidate);
    if (!schemaValidation.ok) {
      throw new Error(`RunPlan schema validation failed: ${schemaValidation.reason}`);
    }

    const runId = this.requireString(candidate, "run_id");
    const planId = this.requireString(candidate, "plan_id");
    const supersedesPlanId = this.optionalString(candidate, "supersedes_plan_id");

    const gateDefinitionsRaw = candidate.gate_definitions;
    if (!Array.isArray(gateDefinitionsRaw)) {
      throw new Error("RunPlan.gate_definitions must be an array");
    }

    const gateDefinitions = gateDefinitionsRaw.map((entry, index) => this.assertRecord(entry, `gate_definitions[${index}]`));
    const dependencyEdges = this.parseDependencyEdges(candidate);
    const entries = gateDefinitions.map((entry, index) => this.normalizeGateDefinitionAsEntry(entry, index));
    return {
      ...candidate,
      schema_id: RUN_PLAN_SCHEMA_ID,
      schema_version: schemaVersion,
      run_id: runId,
      plan_id: planId,
      supersedes_plan_id: supersedesPlanId,
      gate_definitions: gateDefinitions,
      dependency_edges: dependencyEdges,
      entries,
    };
  }

  private parseDependencyEdges(record: Record<string, unknown>): RunnerDependencyEdge[] {
    const raw = record.dependency_edges;
    if (!Array.isArray(raw)) {
      throw new Error("RunPlan.dependency_edges must be an array");
    }
    return raw.map((value, index) => {
      const edge = this.assertRecord(value, `dependency_edges[${index}]`);
      const dependencyKind = this.requireString(edge, "dependency_kind", `dependency_edges[${index}]`);
      if (dependencyKind !== "step_completed") {
        throw new Error(`dependency_edges[${index}].dependency_kind must be step_completed`);
      }
      return {
        upstream_step_id: this.requireString(edge, "upstream_step_id", `dependency_edges[${index}]`),
        downstream_step_id: this.requireString(edge, "downstream_step_id", `dependency_edges[${index}]`),
        dependency_kind: "step_completed",
      };
    });
  }

  identityOf(plan: RunnerPlan): RunnerPlanIdentity {
    return {
      run_id: plan.run_id,
      plan_id: plan.plan_id,
      supersedes_plan_id: plan.supersedes_plan_id,
    };
  }

  private normalizeGateDefinitionAsEntry(record: Record<string, unknown>, index: number): RunnerPlanEntry {
    const gate = this.assertRecord(record.gate, `gate_definitions[${index}].gate`);
    const gateId = this.requireString(gate, "gate_id", `gate_definitions[${index}].gate`);
    const stageId = this.requireString(record, "stage_id", `gate_definitions[${index}]`);
    const stepId = this.requireString(record, "step_id", `gate_definitions[${index}]`);
    const roleInstanceId = this.requireString(record, "role_instance_id", `gate_definitions[${index}]`);
    const executorBindingId = this.requireString(record, "executor_binding_id", `gate_definitions[${index}]`);
    const orderIndex = record.order_index;
    if (typeof orderIndex !== "number" || !Number.isInteger(orderIndex) || orderIndex < 0) {
      throw new Error(`gate_definitions[${index}].order_index must be a non-negative integer`);
    }

    return {
      ...record,
      entry_id: `${stageId}:${stepId}`,
      entry_kind: "gate_definition",
      order_index: orderIndex,
      executor_ref: executorBindingId,
      gate_id: gateId,
      role_instance_id: roleInstanceId,
      stage_id: stageId,
      step_id: stepId,
      dependency_cache_handoffs: this.optionalDependencyCacheHandoffs(record, `gate_definitions[${index}]`),
    };
  }

  private optionalDependencyCacheHandoffs(record: Record<string, unknown>, location: string): DependencyCacheHandoffRequirement[] | undefined {
    const raw = record.dependency_cache_handoffs;
    if (raw === undefined) {
      return undefined;
    }
    if (!Array.isArray(raw) || raw.length === 0) {
      throw new Error(`${location}.dependency_cache_handoffs must be a non-empty array when provided`);
    }
    return raw.map((entry, index) => this.parseDependencyCacheHandoffRequirement(entry, `${location}.dependency_cache_handoffs[${index}]`));
  }

  private parseDependencyCacheHandoffRequirement(value: unknown, location: string): DependencyCacheHandoffRequirement {
    const record = this.assertRecord(value, location);
    const requestDigest = this.requireDigestObject(record.request_digest, `${location}.request_digest`);
    const consumerRole = this.requireString(record, "consumer_role", location);
    const required = record.required;
    if (required !== true) {
      throw new Error(`${location}.required must be true`);
    }
    return {
      request_digest: requestDigest,
      consumer_role: consumerRole,
      required: true,
    };
  }

  private assertRecord(value: unknown, location = "RunPlan"): Record<string, unknown> {
    if (!value || typeof value !== "object" || Array.isArray(value)) {
      throw new Error(`${location} must be an object`);
    }
    return value as Record<string, unknown>;
  }

  private requireString(record: Record<string, unknown>, field: string, location = "RunPlan"): string {
    const value = record[field];
    if (typeof value !== "string" || value.length === 0) {
      throw new Error(`${location}.${field} must be a non-empty string`);
    }
    return value;
  }

  private optionalString(record: Record<string, unknown>, field: string, location = "RunPlan"): string | undefined {
    const value = record[field];
    if (value === undefined) {
      return undefined;
    }
    if (typeof value !== "string" || value.length === 0) {
      throw new Error(`${location}.${field} must be a non-empty string when provided`);
    }
    return value;
  }

  private requireDigestObject(value: unknown, location: string): string {
    const record = this.assertRecord(value, location);
    const hashAlg = this.requireString(record, "hash_alg", location);
    const hash = this.requireString(record, "hash", location);
    if (hashAlg !== "sha256" || !/^[a-f0-9]{64}$/.test(hash)) {
      throw new Error(`${location} must be a sha256 digest object`);
    }
    return `sha256:${hash}`;
  }
}

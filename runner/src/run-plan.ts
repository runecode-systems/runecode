/**
 * RunPlan loading and validation for runner kernel startup.
 *
 * The runner treats RunPlan as immutable broker-compiled input and fails
 * closed on shape/schema/identity violations.
 */

import { readFile } from "node:fs/promises";
import { ProtocolSchemaBundle } from "./protocol-schema-bundle.ts";

export const RUN_PLAN_SCHEMA_ID = "runecode.protocol.v0.RunPlan";
const RUN_PLAN_SCHEMA_VERSION = "0.4.0";

export type RunnerPlanIdentity = {
  run_id: string;
  plan_id: string;
  supersedes_plan_id?: string;
};

export type DependencyCacheHandoffRequirement = {
  request_digest: string;
  consumer_role: string;
  required: true;
};

export type RunnerPlanEntry = {
  entry_id: string;
  entry_kind: "gate";
  order_index: number;
  stage_id: string;
  step_id: string;
  role_instance_id: string;
  executor_binding_id: string;
  checkpoint_code: string;
  gate: Record<string, unknown>;
  dependency_cache_handoffs?: DependencyCacheHandoffRequirement[];
  depends_on_entry_ids: string[];
  blocks_entry_ids: string[];
  supported_wait_kinds: Array<"waiting_operator_input" | "waiting_approval">;
  [key: string]: unknown;
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
    const entries = this.parseEntries(candidate);
    const executorBindingIDs = this.parseExecutorBindingIDs(candidate);
    this.assertEntryIDUniqueness(entries);
    this.validateCompiledEntryConsistency(entries, gateDefinitions, dependencyEdges, executorBindingIDs);

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

  identityOf(plan: RunnerPlan): RunnerPlanIdentity {
    return {
      run_id: plan.run_id,
      plan_id: plan.plan_id,
      supersedes_plan_id: plan.supersedes_plan_id,
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

  private parseEntries(record: Record<string, unknown>): RunnerPlanEntry[] {
    const raw = record.entries;
    if (!Array.isArray(raw)) {
      throw new Error("RunPlan.entries must be an array");
    }
    return raw.map((value, index) => this.parseEntry(value, index));
  }

  private parseEntry(value: unknown, index: number): RunnerPlanEntry {
    const record = this.assertRecord(value, `entries[${index}]`);
    const entryKind = this.requireString(record, "entry_kind", `entries[${index}]`);
    if (entryKind !== "gate") {
      throw new Error(`entries[${index}].entry_kind must be gate`);
    }
    const orderIndex = record.order_index;
    if (typeof orderIndex !== "number" || !Number.isInteger(orderIndex) || orderIndex < 0) {
      throw new Error(`entries[${index}].order_index must be a non-negative integer`);
    }
    const gate = this.assertRecord(record.gate, `entries[${index}].gate`);
    const waitKinds = this.parseSupportedWaitKinds(record, index);
    return {
      ...record,
      entry_id: this.requireString(record, "entry_id", `entries[${index}]`),
      entry_kind: "gate",
      order_index: orderIndex,
      stage_id: this.requireString(record, "stage_id", `entries[${index}]`),
      step_id: this.requireString(record, "step_id", `entries[${index}]`),
      role_instance_id: this.requireString(record, "role_instance_id", `entries[${index}]`),
      executor_binding_id: this.requireString(record, "executor_binding_id", `entries[${index}]`),
      checkpoint_code: this.requireString(record, "checkpoint_code", `entries[${index}]`),
      gate,
      dependency_cache_handoffs: this.optionalDependencyCacheHandoffs(record, `entries[${index}]`),
      depends_on_entry_ids: this.parseEntryIDList(record.depends_on_entry_ids, `entries[${index}].depends_on_entry_ids`),
      blocks_entry_ids: this.parseEntryIDList(record.blocks_entry_ids, `entries[${index}].blocks_entry_ids`),
      supported_wait_kinds: waitKinds,
    };
  }

  private parseSupportedWaitKinds(record: Record<string, unknown>, index: number): Array<"waiting_operator_input" | "waiting_approval"> {
    const values = this.parseEntryIDList(record.supported_wait_kinds, `entries[${index}].supported_wait_kinds`);
    const unique = new Set(values);
    if (!unique.has("waiting_operator_input") || !unique.has("waiting_approval")) {
      throw new Error(`entries[${index}].supported_wait_kinds must include waiting_operator_input and waiting_approval`);
    }
    if (unique.size !== values.length) {
      throw new Error(`entries[${index}].supported_wait_kinds must not contain duplicates`);
    }
    return values.map((value) => {
      if (value !== "waiting_operator_input" && value !== "waiting_approval") {
        throw new Error(`entries[${index}].supported_wait_kinds contains unsupported value ${value}`);
      }
      return value;
    }) as Array<"waiting_operator_input" | "waiting_approval">;
  }

  private parseEntryIDList(value: unknown, location: string): string[] {
    if (!Array.isArray(value)) {
      throw new Error(`${location} must be an array`);
    }
    return value.map((entry, index) => {
      if (typeof entry !== "string" || entry.length === 0) {
        throw new Error(`${location}[${index}] must be a non-empty string`);
      }
      return entry;
    });
  }

  private assertEntryIDUniqueness(entries: RunnerPlanEntry[]): void {
    const seen = new Set<string>();
    for (const entry of entries) {
      if (seen.has(entry.entry_id)) {
        throw new Error(`RunPlan.entries contains duplicate entry_id ${entry.entry_id}`);
      }
      seen.add(entry.entry_id);
    }
  }

  private parseExecutorBindingIDs(record: Record<string, unknown>): Set<string> {
    const raw = record.executor_bindings;
    if (!Array.isArray(raw)) {
      throw new Error("RunPlan.executor_bindings must be an array");
    }
    const bindingIDs = new Set<string>();
    raw.forEach((value, index) => {
      const binding = this.assertRecord(value, `executor_bindings[${index}]`);
      bindingIDs.add(this.requireString(binding, "binding_id", `executor_bindings[${index}]`));
    });
    return bindingIDs;
  }

  private validateCompiledEntryConsistency(entries: RunnerPlanEntry[], gateDefinitions: Array<Record<string, unknown>>, dependencyEdges: RunnerDependencyEdge[], executorBindingIDs: Set<string>): void {
    const entriesByID = new Map(entries.map((entry) => [entry.entry_id, entry]));
    const expectedDeps = new Map<string, string[]>();
    const expectedBlocks = new Map<string, string[]>();
    dependencyEdges.forEach((edge) => {
      expectedDeps.set(edge.downstream_step_id, [...(expectedDeps.get(edge.downstream_step_id) ?? []), edge.upstream_step_id]);
      expectedBlocks.set(edge.upstream_step_id, [...(expectedBlocks.get(edge.upstream_step_id) ?? []), edge.downstream_step_id]);
    });
    for (const [entryID, dependsOn] of expectedDeps) {
      expectedDeps.set(entryID, this.sortedStrings(dependsOn));
    }
    for (const [entryID, blocks] of expectedBlocks) {
      expectedBlocks.set(entryID, this.sortedStrings(blocks));
    }

    gateDefinitions.forEach((definition, index) => {
      const location = `gate_definitions[${index}]`;
      const stepID = this.requireString(definition, "step_id", location);
      const entry = entriesByID.get(stepID);
      if (!entry) {
        throw new Error(`${location}.step_id ${stepID} does not have a matching RunPlan entry`);
      }
      if (entry.entry_id !== stepID) {
        throw new Error(`entries for step ${stepID} must use entry_id equal to step_id`);
      }
      if (!executorBindingIDs.has(entry.executor_binding_id)) {
        throw new Error(`entries for step ${stepID} reference unknown executor_binding_id ${entry.executor_binding_id}`);
      }
      if (entry.stage_id !== this.requireString(definition, "stage_id", location) || entry.role_instance_id !== this.requireString(definition, "role_instance_id", location) || entry.executor_binding_id !== this.requireString(definition, "executor_binding_id", location) || entry.checkpoint_code !== this.requireString(definition, "checkpoint_code", location)) {
        throw new Error(`entries for step ${stepID} must match gate definition scope and binding fields`);
      }
      const orderIndex = definition.order_index;
      if (typeof orderIndex !== "number" || !Number.isInteger(orderIndex) || orderIndex !== entry.order_index) {
        throw new Error(`entries for step ${stepID} must match gate definition order_index`);
      }
      const gate = this.assertRecord(definition.gate, `${location}.gate`);
      if (this.stableSerialize(gate) !== this.stableSerialize(entry.gate)) {
        throw new Error(`entries for step ${stepID} must match gate definition gate payload`);
      }
      const expectedHandoffs = this.optionalDependencyCacheHandoffs(definition, location) ?? [];
      const actualHandoffs = entry.dependency_cache_handoffs ?? [];
      if (this.stableSerialize(expectedHandoffs) !== this.stableSerialize(actualHandoffs)) {
        throw new Error(`entries for step ${stepID} must match gate definition dependency_cache_handoffs`);
      }
      if (!this.sameStringArrays(this.sortedStrings(entry.depends_on_entry_ids), expectedDeps.get(stepID) ?? [])) {
        throw new Error(`entries for step ${stepID} must match dependency_edges upstream bindings`);
      }
      if (!this.sameStringArrays(this.sortedStrings(entry.blocks_entry_ids), expectedBlocks.get(stepID) ?? [])) {
        throw new Error(`entries for step ${stepID} must match dependency_edges downstream bindings`);
      }
    });
    if (entries.length !== gateDefinitions.length) {
      throw new Error("RunPlan.entries must have a one-to-one correspondence with gate_definitions");
    }
  }

  private sortedStrings(values: string[]): string[] {
    return [...values].sort();
  }

  private sameStringArrays(left: string[], right: string[]): boolean {
    if (left.length !== right.length) {
      return false;
    }
    return left.every((value, index) => value === right[index]);
  }

  private stableSerialize(value: unknown): string {
    return JSON.stringify(this.sortValue(value));
  }

  private sortValue(value: unknown): unknown {
    if (Array.isArray(value)) {
      return value.map((entry) => this.sortValue(entry));
    }
    if (!value || typeof value !== "object") {
      return value;
    }
    const record = value as Record<string, unknown>;
    return Object.keys(record).sort().reduce<Record<string, unknown>>((acc, key) => {
      acc[key] = this.sortValue(record[key]);
      return acc;
    }, {});
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

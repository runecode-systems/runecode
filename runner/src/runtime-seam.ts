/**
 * Narrow internal runtime seam for local checkpoint/wait/resume mechanics.
 *
 * This seam is intentionally internal and non-canonical: broker-owned run and
 * approval truth remain authoritative.
 */

import type { FileDurableStateStore } from "./durable-state.ts";
import { appendFile, readFile } from "node:fs/promises";
import path from "node:path";
import type { PlanBoundExecutionIdentity } from "./contracts.ts";
import { PlanIdentityMismatchError } from "./durable-state.ts";

const RUNTIME_SEAM_SCHEMA_VERSION = "1";
const RUNTIME_SEAM_KINDS = ["checkpoint", "wait_parked", "wait_resumed"] as const;
const runtimeSeamWriteLocks = new Map<string, Promise<void>>();

type RuntimeSeamRecord = {
  schema_version: typeof RUNTIME_SEAM_SCHEMA_VERSION;
  sequence: number;
  kind: "checkpoint" | "wait_parked" | "wait_resumed";
  run_id: string;
  plan_id: string;
  wait_id?: string;
  wait_kind?: "approval";
  resume_token?: string;
  checkpoint_code?: string;
  idempotency_key: string;
  occurred_at: string;
  payload_details?: Record<string, unknown>;
};

export type RuntimeCheckpointInput = {
  identity: PlanBoundExecutionIdentity;
  checkpoint_code: string;
  idempotency_key: string;
  occurred_at?: string;
  details?: Record<string, unknown>;
};

export type RuntimeWaitInput = {
  identity: PlanBoundExecutionIdentity;
  wait_kind: "approval";
  wait_id: string;
  resume_token: string;
  idempotency_key: string;
  details?: Record<string, unknown>;
  occurred_at?: string;
};

export type RuntimeWaitResumeInput = {
  identity: PlanBoundExecutionIdentity;
  wait_id: string;
  idempotency_key: string;
  details?: Record<string, unknown>;
  occurred_at?: string;
};

export type RuntimeRestoredWait = {
  wait_kind: "approval";
  wait_id: string;
  resume_token: string;
  details?: Record<string, unknown>;
};

export type RunnerRuntimeSeam = {
  checkpoint(input: RuntimeCheckpointInput): Promise<void>;
  parkWait(input: RuntimeWaitInput): Promise<void>;
  resumeWait(input: RuntimeWaitResumeInput): Promise<void>;
  restoreWaits(identity: Pick<PlanBoundExecutionIdentity, "run_id" | "plan_id">): Promise<RuntimeRestoredWait[]>;
};

export class DurableRuntimeSeam implements RunnerRuntimeSeam {
  private readonly durableStateStore: FileDurableStateStore;

  private readonly runtimeJournalPath: string;

  constructor(durableStateStore: FileDurableStateStore) {
    this.durableStateStore = durableStateStore;
    this.runtimeJournalPath = path.join(durableStateStore.runtimeStateRoot(), "runtime-seam.v1.ndjson");
  }

  async checkpoint(input: RuntimeCheckpointInput): Promise<void> {
    await this.appendRecord({
      kind: "checkpoint",
      run_id: input.identity.run_id,
      plan_id: input.identity.plan_id,
      idempotency_key: input.idempotency_key,
      occurred_at: input.occurred_at,
      checkpoint_code: input.checkpoint_code,
      payload_details: input.details,
    });
  }

  async parkWait(input: RuntimeWaitInput): Promise<void> {
    await this.appendRecord({
      kind: "wait_parked",
      run_id: input.identity.run_id,
      plan_id: input.identity.plan_id,
      idempotency_key: input.idempotency_key,
      occurred_at: input.occurred_at,
      wait_kind: input.wait_kind,
      wait_id: input.wait_id,
      resume_token: input.resume_token,
      payload_details: input.details,
    });
  }

  async resumeWait(input: RuntimeWaitResumeInput): Promise<void> {
    await this.appendRecord({
      kind: "wait_resumed",
      run_id: input.identity.run_id,
      plan_id: input.identity.plan_id,
      idempotency_key: input.idempotency_key,
      occurred_at: input.occurred_at,
      wait_id: input.wait_id,
      payload_details: input.details,
    });
  }

  async restoreWaits(identity: Pick<PlanBoundExecutionIdentity, "run_id" | "plan_id">): Promise<RuntimeRestoredWait[]> {
    const state = await this.durableStateStore.readState();
    if (state.snapshot.run_id !== identity.run_id || state.snapshot.plan_id !== identity.plan_id) {
      throw new PlanIdentityMismatchError(
        `runtime seam identity ${identity.run_id}/${identity.plan_id} does not match durable binding ${state.snapshot.run_id}/${state.snapshot.plan_id}`,
      );
    }

    const records = await this.readRecords();
    for (const record of records) {
      if (record.run_id !== state.snapshot.run_id || record.plan_id !== state.snapshot.plan_id) {
        throw new PlanIdentityMismatchError(
          `runtime seam record ${record.sequence} identity ${record.run_id}/${record.plan_id} does not match durable binding ${state.snapshot.run_id}/${state.snapshot.plan_id}`,
        );
      }
    }

    const pendingWaits = new Map((await this.durableStateStore.listPendingApprovalWaits()).map((wait) => [wait.approval_id, wait]));
    const parkedByWaitId = new Map<string, RuntimeRestoredWait>();
    for (const record of records) {
      if (record.kind === "wait_parked") {
        const waitKind = record.wait_kind;
        const waitId = record.wait_id;
        const resumeToken = record.resume_token;
        if (waitKind !== "approval" || !waitId || !resumeToken) {
          continue;
        }
        if (!pendingWaits.has(waitId)) {
          continue;
        }

        parkedByWaitId.set(waitId, {
          wait_kind: "approval",
          wait_id: waitId,
          resume_token: resumeToken,
          details: record.payload_details,
        });
      }

      if (record.kind === "wait_resumed") {
        const waitId = record.wait_id;
        if (waitId) {
          parkedByWaitId.delete(waitId);
        }
      }
    }

    return [...parkedByWaitId.values()];
  }

  private async readRecords(): Promise<RuntimeSeamRecord[]> {
    const lines = await readRuntimeLines(this.runtimeJournalPath);
    const records: RuntimeSeamRecord[] = [];
    for (const [index, line] of lines.entries()) {
      const parsed = parseRuntimeSeamRecord(JSON.parse(line) as unknown, `runtime seam line ${index + 1}`);
      const expectedSequence = index + 1;
      if (parsed.sequence !== expectedSequence) {
        throw new Error(`runtime seam record ${parsed.sequence} must equal ${expectedSequence}`);
      }
      records.push(parsed);
    }
    return records;
  }

  private async appendRecord(input: RuntimeSeamRecordInput): Promise<void> {
    await this.withWriteLock(async () => {
      const existing = await this.readRecords();
      const duplicate = existing.find((record) => record.idempotency_key === input.idempotency_key);
      if (duplicate) {
        assertRuntimeRecordMatches(duplicate, input);
        return;
      }

      const record: RuntimeSeamRecord = {
        ...input,
        schema_version: RUNTIME_SEAM_SCHEMA_VERSION,
        sequence: existing.length + 1,
        occurred_at: input.occurred_at ?? new Date().toISOString(),
      };
      await appendFile(this.runtimeJournalPath, `${JSON.stringify(record)}\n`, "utf8");
    });
  }

  private async withWriteLock<T>(operation: () => Promise<T>): Promise<T> {
    const key = this.runtimeJournalPath;
    const previous = runtimeSeamWriteLocks.get(key) ?? Promise.resolve();
    let release: () => void;
    const current = new Promise<void>((resolve) => {
      release = resolve;
    });
    const queued = previous.then(() => current);
    runtimeSeamWriteLocks.set(key, queued);
    await previous;
    try {
      return await operation();
    } finally {
      release!();
      if (runtimeSeamWriteLocks.get(key) === queued) {
        runtimeSeamWriteLocks.delete(key);
      }
    }
  }
}

type RuntimeSeamRecordInput = Omit<RuntimeSeamRecord, "schema_version" | "sequence" | "occurred_at"> & { occurred_at?: string };

async function readRuntimeLines(runtimeJournalPath: string): Promise<string[]> {
  let raw: string;
  try {
    raw = await readFile(runtimeJournalPath, "utf8");
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") {
      return [];
    }
    throw error;
  }
  return raw.split("\n").filter((line) => line.length > 0);
}

function parseRuntimeSeamRecord(value: unknown, location: string): RuntimeSeamRecord {
  const record = assertObjectRecord(value, location);
  const schemaVersion = requireString(record, "schema_version", location);
  if (schemaVersion !== RUNTIME_SEAM_SCHEMA_VERSION) {
    throw new Error(`unsupported runtime seam schema version ${schemaVersion}`);
  }

  const kind = requireEnum(record, "kind", RUNTIME_SEAM_KINDS, location);
  const parsed: RuntimeSeamRecord = {
    schema_version: RUNTIME_SEAM_SCHEMA_VERSION,
    sequence: requireNonNegativeInt(record, "sequence", location),
    kind,
    run_id: requireString(record, "run_id", location),
    plan_id: requireString(record, "plan_id", location),
    idempotency_key: requireString(record, "idempotency_key", location),
    occurred_at: requireString(record, "occurred_at", location),
    wait_id: optionalString(record, "wait_id", location),
    wait_kind: optionalApprovalWaitKind(record.wait_kind, `${location}.wait_kind`),
    resume_token: optionalString(record, "resume_token", location),
    checkpoint_code: optionalString(record, "checkpoint_code", location),
    payload_details: optionalRecord(record.payload_details, `${location}.payload_details`),
  };

  if (parsed.kind === "wait_parked") {
    if (parsed.wait_kind !== "approval" || !parsed.wait_id || !parsed.resume_token) {
      throw new Error(`${location} wait_parked records require wait_kind, wait_id, and resume_token`);
    }
  }

  if (parsed.kind === "wait_resumed" && !parsed.wait_id) {
    throw new Error(`${location} wait_resumed records require wait_id`);
  }

  if (parsed.kind === "checkpoint" && !parsed.checkpoint_code) {
    throw new Error(`${location} checkpoint records require checkpoint_code`);
  }

  return parsed;
}

function assertRuntimeRecordMatches(existing: RuntimeSeamRecord, input: RuntimeSeamRecordInput): void {
  const occurredAt = input.occurred_at ?? existing.occurred_at;
  if (
    existing.kind !== input.kind
    || existing.run_id !== input.run_id
    || existing.plan_id !== input.plan_id
    || existing.wait_id !== input.wait_id
    || existing.wait_kind !== input.wait_kind
    || existing.resume_token !== input.resume_token
    || existing.checkpoint_code !== input.checkpoint_code
    || existing.occurred_at !== occurredAt
    || JSON.stringify(existing.payload_details ?? null) !== JSON.stringify(input.payload_details ?? null)
  ) {
    throw new Error(`runtime seam idempotency key ${input.idempotency_key} conflicts with existing ${existing.kind} record`);
  }
}

function assertObjectRecord(value: unknown, location: string): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new Error(`${location} must be an object`);
  }
  return value as Record<string, unknown>;
}

function requireString(record: Record<string, unknown>, key: string, location: string): string {
  const value = record[key];
  if (typeof value !== "string" || value.trim().length === 0) {
    throw new Error(`${location}.${key} must be a non-empty string`);
  }
  return value.trim();
}

function optionalString(record: Record<string, unknown>, key: string, location: string): string | undefined {
  const value = record[key];
  if (value === undefined) {
    return undefined;
  }
  if (typeof value !== "string" || value.trim().length === 0) {
    throw new Error(`${location}.${key} must be a non-empty string when provided`);
  }
  return value.trim();
}

function requireNonNegativeInt(record: Record<string, unknown>, key: string, location: string): number {
  const value = record[key];
  if (typeof value !== "number" || !Number.isInteger(value) || value < 0) {
    throw new Error(`${location}.${key} must be a non-negative integer`);
  }
  return value;
}

function requireEnum<const T extends readonly string[]>(record: Record<string, unknown>, key: string, values: T, location: string): T[number] {
  const value = record[key];
  if (typeof value !== "string" || !values.includes(value as T[number])) {
    throw new Error(`${location}.${key} must be one of ${values.join(", ")}`);
  }
  return value as T[number];
}

function optionalApprovalWaitKind(value: unknown, location: string): "approval" | undefined {
  if (value === undefined) {
    return undefined;
  }
  if (value !== "approval") {
    throw new Error(`${location} must be approval when provided`);
  }
  return "approval";
}

function optionalRecord(value: unknown, location: string): Record<string, unknown> | undefined {
  if (value === undefined) {
    return undefined;
  }
  return assertObjectRecord(value, location);
}

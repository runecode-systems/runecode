/**
 * Runner-local durable state journal and snapshot store.
 *
 * This state is explicitly non-authoritative and internal to the untrusted
 * runner process. It is bound to RunPlan identity to prevent silent replay of
 * stale or superseded plan state.
 */

import { appendFile, mkdir, readFile, rename, writeFile } from "node:fs/promises";
import path from "node:path";
import type { RunnerPlanIdentity } from "./run-plan.ts";

const SNAPSHOT_SCHEMA_VERSION = "1";
const JOURNAL_SCHEMA_VERSION = "1";

export type DurableSnapshot = {
  schema_version: typeof SNAPSHOT_SCHEMA_VERSION;
  run_id: string;
  plan_id: string;
  supersedes_plan_id?: string;
  last_sequence: number;
  created_at: string;
  updated_at: string;
};

export type DurableJournalRecord = {
  schema_version: typeof JOURNAL_SCHEMA_VERSION;
  sequence: number;
  kind: string;
  idempotency_key: string;
  occurred_at: string;
  details?: Record<string, unknown>;
};

export type DurableStateView = {
  snapshot: DurableSnapshot;
  journal: DurableJournalRecord[];
};

export class PlanIdentityMismatchError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "PlanIdentityMismatchError";
  }
}

export class FileDurableStateStore {
  private readonly stateRoot: string;

  private readonly snapshotPath: string;

  private readonly journalPath: string;

  constructor(stateRoot: string) {
    this.stateRoot = stateRoot;
    this.snapshotPath = path.join(stateRoot, "snapshot.v1.json");
    this.journalPath = path.join(stateRoot, "journal.v1.ndjson");
  }

  async bindPlanIdentity(planIdentity: RunnerPlanIdentity): Promise<void> {
    await mkdir(this.stateRoot, { recursive: true });

    const existing = await this.tryReadSnapshot();
    if (!existing) {
      const now = new Date().toISOString();
      await this.writeSnapshot({
        schema_version: SNAPSHOT_SCHEMA_VERSION,
        run_id: planIdentity.run_id,
        plan_id: planIdentity.plan_id,
        supersedes_plan_id: planIdentity.supersedes_plan_id,
        last_sequence: 0,
        created_at: now,
        updated_at: now,
      });
      return;
    }

    this.assertIdentityMatch(existing, planIdentity);
  }

  async appendRecord(input: Omit<DurableJournalRecord, "schema_version" | "sequence" | "occurred_at"> & { occurred_at?: string }): Promise<DurableJournalRecord> {
    const snapshot = await this.readSnapshot();
    const record: DurableJournalRecord = {
      schema_version: JOURNAL_SCHEMA_VERSION,
      sequence: snapshot.last_sequence + 1,
      kind: input.kind,
      idempotency_key: input.idempotency_key,
      occurred_at: input.occurred_at ?? new Date().toISOString(),
      details: input.details,
    };

    await appendFile(this.journalPath, `${JSON.stringify(record)}\n`, "utf8");
    await this.writeSnapshot({
      ...snapshot,
      last_sequence: record.sequence,
      updated_at: record.occurred_at,
    });
    return record;
  }

  async readState(): Promise<DurableStateView> {
    const snapshot = await this.readSnapshot();
    const journal = await this.readJournal();
    return { snapshot, journal };
  }

  private async readSnapshot(): Promise<DurableSnapshot> {
    const snapshot = await this.tryReadSnapshot();
    if (!snapshot) {
      throw new Error("runner durable snapshot is missing");
    }
    return snapshot;
  }

  private async tryReadSnapshot(): Promise<DurableSnapshot | null> {
    let raw: string;
    try {
      raw = await readFile(this.snapshotPath, "utf8");
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code === "ENOENT") {
        return null;
      }
      throw error;
    }

    const parsed = JSON.parse(raw) as DurableSnapshot;
    if (parsed.schema_version !== SNAPSHOT_SCHEMA_VERSION) {
      throw new Error(`unsupported durable snapshot schema version ${parsed.schema_version}`);
    }

    return parsed;
  }

  private async readJournal(): Promise<DurableJournalRecord[]> {
    let raw: string;
    try {
      raw = await readFile(this.journalPath, "utf8");
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code === "ENOENT") {
        return [];
      }
      throw error;
    }

    const records: DurableJournalRecord[] = [];
    const lines = raw.split("\n").filter((line) => line.length > 0);
    for (const line of lines) {
      const record = JSON.parse(line) as DurableJournalRecord;
      if (record.schema_version !== JOURNAL_SCHEMA_VERSION) {
        throw new Error(`unsupported durable journal schema version ${record.schema_version}`);
      }
      records.push(record);
    }
    return records;
  }

  private assertIdentityMatch(snapshot: DurableSnapshot, planIdentity: RunnerPlanIdentity): void {
    if (snapshot.run_id !== planIdentity.run_id) {
      throw new PlanIdentityMismatchError(`durable snapshot run_id ${snapshot.run_id} does not match plan run_id ${planIdentity.run_id}`);
    }
    if (snapshot.plan_id !== planIdentity.plan_id) {
      throw new PlanIdentityMismatchError(`durable snapshot plan_id ${snapshot.plan_id} does not match plan plan_id ${planIdentity.plan_id}`);
    }
  }

  private async writeSnapshot(snapshot: DurableSnapshot): Promise<void> {
    const tempPath = `${this.snapshotPath}.tmp`;
    await writeFile(tempPath, `${JSON.stringify(snapshot, null, 2)}\n`, "utf8");
    await rename(tempPath, this.snapshotPath);
  }
}

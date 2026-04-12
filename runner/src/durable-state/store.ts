import { appendFile, mkdir, readFile, rename, writeFile } from "node:fs/promises";
import path from "node:path";
import type { RunnerPlanIdentity } from "../run-plan.ts";
import {
  JOURNAL_SCHEMA_VERSION,
  SNAPSHOT_SCHEMA_VERSION,
  type DurableAppendRecordInput,
  type DurableApprovalWait,
  type DurableJournalRecord,
  type DurableReplayState,
  type DurableSnapshot,
  type DurableStateView,
  type EnterApprovalWaitInput,
  type ResolveApprovalWaitInput,
  DurableReplayError,
  InvalidApprovalWaitError,
  assertBoundIdentity,
  assertIdentityMatch,
  cloneApprovalWait,
} from "./types.ts";
import { parseJournalRecord, parseSnapshot, buildRecord, assertDurableRecordMatches } from "./codec.ts";
import { approvalWaitBindingsMatch, assertApprovalBinding, blockedScopeToDurableScope, sanitizeBlockedScope, sanitizeBrokerCorrelation } from "./helpers.ts";
import { healSnapshotFromJournal, replayDurableState, replayDurableStateInternal, snapshotNeedsRewrite } from "./replay.ts";

const durableStateWriteLocks = new Map<string, Promise<void>>();

export class FileDurableStateStore {
  private readonly stateRoot: string;

  private readonly snapshotPath: string;

  private readonly journalPath: string;

  constructor(stateRoot: string) {
    this.stateRoot = stateRoot;
    this.snapshotPath = path.join(stateRoot, "snapshot.v2.json");
    this.journalPath = path.join(stateRoot, "journal.v2.ndjson");
  }

  runtimeStateRoot(): string {
    return this.stateRoot;
  }

  async bindPlanIdentity(planIdentity: RunnerPlanIdentity): Promise<void> {
    await this.withWriteLock(async () => {
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
          pending_approval_waits: [],
          created_at: now,
          updated_at: now,
        });
        return;
      }
      assertIdentityMatch(existing, planIdentity);
    });
  }

  async appendRecord(input: DurableAppendRecordInput): Promise<DurableJournalRecord> {
    return this.withWriteLock(async () => {
      const { records } = await this.appendRecordsLocked([input]);
      return records[0];
    });
  }

  async readState(): Promise<DurableStateView> {
    return this.withWriteLock(async () => this.readStateUnlocked());
  }

  async replayState(): Promise<DurableReplayState> {
    const { snapshot, journal } = await this.readState();
    return replayDurableState(snapshot, journal);
  }

  async enterApprovalWait(input: EnterApprovalWaitInput): Promise<DurableApprovalWait> {
    return this.withWriteLock(async () => {
      const state = await this.readStateUnlocked();
      assertBoundIdentity(state.snapshot, input.run_id, input.plan_id);
      assertApprovalBinding(input.binding_kind, input.bound_action_hash, input.bound_stage_summary_hash, input.approval_id);

      const existing = state.snapshot.pending_approval_waits.find((wait) => wait.approval_id === input.approval_id);
      if (existing) {
        if (!approvalWaitBindingsMatch(existing, input.binding_kind, input.bound_action_hash, input.bound_stage_summary_hash)) {
          throw new InvalidApprovalWaitError(`approval wait ${input.approval_id} already exists with conflicting binding`);
        }
        return cloneApprovalWait(existing);
      }

      const occurredAt = input.occurred_at ?? new Date().toISOString();
      const blockedScope = sanitizeBlockedScope(input.blocked_scope, `approval wait ${input.approval_id}`);
      if (blockedScope.scope_kind === "run" && blockedScope.run_id !== input.run_id) {
        throw new InvalidApprovalWaitError(`approval wait ${input.approval_id} run-scoped binding ${blockedScope.run_id} does not match active run ${input.run_id}`);
      }
      const brokerCorrelation = sanitizeBrokerCorrelation(input.broker_correlation);
      const actionRequestId = brokerCorrelation.action_request_id ?? brokerCorrelation.request_id ?? `approval:${input.approval_id}`;
      const durableScope = blockedScopeToDurableScope(blockedScope, input.run_id, input.approval_id);
      const replay = replayDurableStateInternal(state.snapshot, state.journal);

      const recordsToAppend: DurableAppendRecordInput[] = [];
      if (!replay.has_run_started) {
        recordsToAppend.push({
          kind: "run_started",
          idempotency_key: `approval_wait_bootstrap_run_started:${input.run_id}`,
          occurred_at: occurredAt,
          run_scope_id: input.run_id,
        });
      }

      recordsToAppend.push(
        {
          kind: "action_request_issued",
          idempotency_key: `${input.idempotency_key}:action_request_issued`,
          occurred_at: occurredAt,
          action_request_id: actionRequestId,
          scope_kind: durableScope.scope_kind,
          scope_id: durableScope.scope_id,
        },
        {
          kind: "approval_wait_entered",
          idempotency_key: input.idempotency_key,
          occurred_at: occurredAt,
          approval_wait_id: input.approval_id,
          action_request_id: actionRequestId,
          binding_kind: input.binding_kind,
          bound_action_hash: input.bound_action_hash,
          bound_stage_summary_hash: input.bound_stage_summary_hash,
          blocked_scope: blockedScope,
          broker_correlation: brokerCorrelation,
        },
      );

      const result = await this.appendRecordsLocked(recordsToAppend);
      const wait = result.snapshot.pending_approval_waits.find((entry) => entry.approval_id === input.approval_id);
      if (!wait) {
        throw new DurableReplayError(`approval wait ${input.approval_id} missing after durable append`);
      }
      return cloneApprovalWait(wait);
    });
  }

  async resolveApprovalWait(input: ResolveApprovalWaitInput): Promise<void> {
    await this.withWriteLock(async () => {
      const state = await this.readStateUnlocked();
      assertBoundIdentity(state.snapshot, input.run_id, input.plan_id);

      const wait = state.snapshot.pending_approval_waits.find((entry) => entry.approval_id === input.approval_id);
      if (!wait) {
        const priorResolution = state.journal.find(
          (entry) => entry.kind === "approval_wait_cleared" && entry.idempotency_key === input.idempotency_key,
        );
        if (priorResolution && priorResolution.kind === "approval_wait_cleared") {
          if (priorResolution.approval_wait_id !== input.approval_id || priorResolution.status !== input.status) {
            throw new InvalidApprovalWaitError(`approval wait ${input.approval_id} resolution conflicts with prior idempotent clear`);
          }
          return;
        }
        throw new InvalidApprovalWaitError(`approval wait ${input.approval_id} is not pending`);
      }
      if (!approvalWaitBindingsMatch(wait, input.binding_kind, input.bound_action_hash, input.bound_stage_summary_hash)) {
        throw new InvalidApprovalWaitError(`approval wait ${input.approval_id} binding mismatch`);
      }

      await this.appendRecordsLocked([
        {
          kind: "approval_wait_cleared",
          idempotency_key: input.idempotency_key,
          occurred_at: input.occurred_at,
          approval_wait_id: input.approval_id,
          action_request_id: wait.action_request_id,
          status: input.status,
        },
      ]);
    });
  }

  async listPendingApprovalWaits(): Promise<DurableApprovalWait[]> {
    const state = await this.readState();
    return state.snapshot.pending_approval_waits.map((wait) => cloneApprovalWait(wait));
  }

  private async readStateUnlocked(): Promise<DurableStateView> {
    const snapshot = await this.readSnapshot();
    const journal = await this.readJournal();
    const healed = healSnapshotFromJournal(snapshot, journal);
    if (snapshotNeedsRewrite(snapshot, healed)) {
      await this.writeSnapshot(healed);
    }
    return { snapshot: healed, journal };
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
    return parseSnapshot(JSON.parse(raw) as unknown, "durable snapshot");
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
    for (const [index, line] of raw.split("\n").filter((entry) => entry.length > 0).entries()) {
      records.push(parseJournalRecord(JSON.parse(line) as unknown, `durable journal line ${index + 1}`));
    }
    return records;
  }

  private async appendRecordsLocked(inputs: DurableAppendRecordInput[]): Promise<DurableStateView & { records: DurableJournalRecord[] }> {
    await mkdir(this.stateRoot, { recursive: true });
    const state = await this.readStateUnlocked();
    const journal = [...state.journal];
    const emitted: DurableJournalRecord[] = [];
    const fresh: DurableJournalRecord[] = [];

    for (const input of inputs) {
      const existing = journal.find((record) => record.idempotency_key === input.idempotency_key);
      if (existing) {
        assertDurableRecordMatches(existing, input);
        emitted.push(existing);
        continue;
      }

      const record = buildRecord(state.snapshot, journal.length + 1, input);
      fresh.push(record);
      journal.push(record);
      emitted.push(record);
    }

    if (fresh.length > 0) {
      await appendFile(this.journalPath, `${fresh.map((record) => JSON.stringify(record)).join("\n")}\n`, "utf8");
    }

    const nextSnapshot = healSnapshotFromJournal(state.snapshot, journal);
    if (snapshotNeedsRewrite(state.snapshot, nextSnapshot)) {
      await this.writeSnapshot(nextSnapshot);
    }
    return { snapshot: nextSnapshot, journal, records: emitted };
  }

  private async writeSnapshot(snapshot: DurableSnapshot): Promise<void> {
    const tempPath = `${this.snapshotPath}.${process.pid}.${Date.now()}.${Math.random().toString(16).slice(2)}.tmp`;
    await writeFile(tempPath, `${JSON.stringify(snapshot, null, 2)}\n`, "utf8");
    await rename(tempPath, this.snapshotPath);
  }

  private async withWriteLock<T>(operation: () => Promise<T>): Promise<T> {
    const previous = durableStateWriteLocks.get(this.stateRoot) ?? Promise.resolve();
    let release: () => void;
    const current = new Promise<void>((resolve) => {
      release = resolve;
    });
    const queued = previous.then(() => current);
    durableStateWriteLocks.set(this.stateRoot, queued);
    await previous;
    try {
      return await operation();
    } finally {
      release!();
      if (durableStateWriteLocks.get(this.stateRoot) === queued) {
        durableStateWriteLocks.delete(this.stateRoot);
      }
    }
  }
}

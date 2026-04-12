const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const { loadRunnerModules } = require("./runner-test-helpers.js");

test("fails closed on durable-state plan identity mismatch", async (t) => {
  const {
    FileDurableStateStore,
    PlanIdentityMismatchError,
  } = await loadRunnerModules();

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-state-"));
  t.after(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });
  await store.appendRecord({ kind: "run_started", idempotency_key: "k1", run_scope_id: "run_alpha" });

  await assert.rejects(
    () => store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_beta" }),
    (error) => error instanceof PlanIdentityMismatchError,
  );
});

test("fails closed on durable snapshot schema mismatch", async (t) => {
  const {
    FileDurableStateStore,
  } = await loadRunnerModules();

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-state-"));
  t.after(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  const snapshotPath = path.join(root, "snapshot.v2.json");
  fs.mkdirSync(root, { recursive: true });
  fs.writeFileSync(snapshotPath, `${JSON.stringify({
    schema_version: "999",
    run_id: "run_alpha",
    plan_id: "plan_alpha",
    last_sequence: 0,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
  }, null, 2)}\n`, "utf8");

  const store = new FileDurableStateStore(root);
  await assert.rejects(
    () => store.readState(),
    /unsupported durable snapshot\.schema_version/,
  );
});

test("replays durable journal into deterministic scheduler/wait/attempt state", async (t) => {
  const {
    FileDurableStateStore,
  } = await loadRunnerModules();

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-state-"));
  t.after(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });

  await store.appendRecord({ kind: "run_started", idempotency_key: "k1", run_scope_id: "run_alpha" });
  await store.appendRecord({ kind: "stage_entered", idempotency_key: "k2", stage_id: "stage_build" });
  await store.appendRecord({ kind: "step_attempt_started", idempotency_key: "k3", stage_id: "stage_build", step_id: "step_lint", step_attempt_id: "step_lint#a1" });
  await store.appendRecord({ kind: "action_request_issued", idempotency_key: "k4", action_request_id: "action-1", scope_kind: "step_attempt", scope_id: "step_lint#a1" });
  await store.appendRecord({
    kind: "approval_wait_entered",
    idempotency_key: "k5",
    approval_wait_id: "approval-1",
    action_request_id: "action-1",
    binding_kind: "exact_action",
    bound_action_hash: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
    blocked_scope: {
      scope_kind: "step",
      run_id: "run_alpha",
      step_id: "step_lint#a1",
      action_kind: "action_gate_override",
    },
    broker_correlation: {
      request_id: "action-1",
    },
  });
  await store.appendRecord({ kind: "approval_wait_cleared", idempotency_key: "k6", approval_wait_id: "approval-1", action_request_id: "action-1", status: "approved" });
  await store.appendRecord({ kind: "gate_attempt_started", idempotency_key: "k7", stage_id: "stage_build", gate_id: "lint", gate_attempt_id: "gate_lint#a1" });
  await store.appendRecord({ kind: "gate_attempt_finished", idempotency_key: "k8", gate_id: "lint", gate_attempt_id: "gate_lint#a1", outcome: "passed" });
  await store.appendRecord({ kind: "step_attempt_finished", idempotency_key: "k9", step_id: "step_lint", step_attempt_id: "step_lint#a1", outcome: "succeeded" });
  await store.appendRecord({ kind: "run_terminal", idempotency_key: "k10", terminal_status: "succeeded" });

  const replay = await store.replayState();
  assert.equal(replay.last_sequence, 10);
  assert.deepEqual(replay.scheduler.entered_stage_ids, ["stage_build"]);
  assert.equal(replay.scheduler.current_stage_id, "stage_build");
  assert.deepEqual(replay.waits.pending_approval_waits, []);
  assert.deepEqual(replay.waits.resolved_approval_waits, [{ approval_id: "approval-1", status: "approved" }]);
  assert.deepEqual(replay.attempts.active_step_attempt_ids, []);
  assert.deepEqual(replay.attempts.finished_step_attempt_ids, ["step_lint#a1"]);
  assert.deepEqual(replay.attempts.active_gate_attempt_ids, []);
  assert.deepEqual(replay.attempts.finished_gate_attempt_ids, ["gate_lint#a1"]);
  assert.deepEqual(replay.actions.issued_action_request_ids, ["action-1"]);
  assert.deepEqual(replay.terminal, { sequence: 10, terminal_status: "succeeded" });
});

test("fails closed on invalid replay state transitions", async (t) => {
  const {
    FileDurableStateStore,
    DurableReplayError,
  } = await loadRunnerModules();

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-state-"));
  t.after(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });
  await store.appendRecord({ kind: "run_started", idempotency_key: "k1", run_scope_id: "run_alpha" });
  await assert.rejects(
    () => store.appendRecord({ kind: "step_attempt_finished", idempotency_key: "k2", step_id: "step_lint", step_attempt_id: "step_lint#a1", outcome: "failed" }),
    (error) => error instanceof DurableReplayError,
  );

  await assert.rejects(
    () => store.replayState(),
    (error) => error instanceof DurableReplayError,
  );
});

test("fails closed on conflicting durable idempotent append payloads", async (t) => {
  const {
    FileDurableStateStore,
    DurableReplayError,
  } = await loadRunnerModules();

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-state-"));
  t.after(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });
  await store.appendRecord({ kind: "run_started", idempotency_key: "k1", run_scope_id: "run_alpha" });

  await assert.rejects(
    () => store.appendRecord({ kind: "run_started", idempotency_key: "k1", run_scope_id: "run_beta" }),
    (error) => error instanceof DurableReplayError,
  );
});

test("persists approval waits with canonical binding and restart-safe resume fields", async (t) => {
  const {
    FileDurableStateStore,
  } = await loadRunnerModules();

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-state-"));
  t.after(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });

  await store.enterApprovalWait({
    approval_id: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
    run_id: "run_alpha",
    plan_id: "plan_alpha",
    binding_kind: "exact_action",
    bound_action_hash: "sha256:2222222222222222222222222222222222222222222222222222222222222222",
    blocked_scope: {
      scope_kind: "step",
      run_id: "run_alpha",
      stage_id: "stage_alpha",
      step_id: "step_lint#a1",
      role_instance_id: "role_alpha",
      action_kind: "action_gate_override",
    },
    broker_correlation: {
      request_id: "action-1",
      operation_id: "op-approval-1",
    },
    idempotency_key: "wait-enter-1",
  });

  const restartedStore = new FileDurableStateStore(root);
  await restartedStore.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });
  const waits = await restartedStore.listPendingApprovalWaits();

  assert.equal(waits.length, 1);
  assert.equal(waits[0].approval_id, "sha256:1111111111111111111111111111111111111111111111111111111111111111");
  assert.equal(waits[0].action_request_id, "action-1");
  assert.equal(waits[0].binding_kind, "exact_action");
  assert.equal(waits[0].bound_action_hash, "sha256:2222222222222222222222222222222222222222222222222222222222222222");
  assert.equal(waits[0].blocked_scope.step_id, "step_lint#a1");
  assert.equal(waits[0].broker_correlation.request_id, "action-1");
  assert.equal(waits[0].run_id, "run_alpha");
  assert.equal(waits[0].plan_id, "plan_alpha");
});

test("fails closed when stage or step blocked scopes omit required identifiers", async (t) => {
  const {
    FileDurableStateStore,
    InvalidApprovalWaitError,
  } = await loadRunnerModules();

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-state-"));
  t.after(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });

  await assert.rejects(
    () => store.enterApprovalWait({
      approval_id: "sha256:1212121212121212121212121212121212121212121212121212121212121212",
      run_id: "run_alpha",
      plan_id: "plan_alpha",
      binding_kind: "stage_sign_off",
      bound_stage_summary_hash: "sha256:3434343434343434343434343434343434343434343434343434343434343434",
      blocked_scope: {
        scope_kind: "stage",
        run_id: "run_alpha",
        action_kind: "stage_summary_sign_off",
      },
      broker_correlation: { request_id: "missing-stage-id" },
      idempotency_key: "wait-enter-missing-stage-id",
    }),
    (error) => error instanceof InvalidApprovalWaitError,
  );

  await assert.rejects(
    () => store.enterApprovalWait({
      approval_id: "sha256:5656565656565656565656565656565656565656565656565656565656565656",
      run_id: "run_alpha",
      plan_id: "plan_alpha",
      binding_kind: "exact_action",
      bound_action_hash: "sha256:7878787878787878787878787878787878787878787878787878787878787878",
      blocked_scope: {
        scope_kind: "step",
        run_id: "run_alpha",
        action_kind: "action_gate_override",
      },
      broker_correlation: { request_id: "missing-step-id" },
      idempotency_key: "wait-enter-missing-step-id",
    }),
    (error) => error instanceof InvalidApprovalWaitError,
  );

  await assert.rejects(
    () => store.enterApprovalWait({
      approval_id: "sha256:9090909090909090909090909090909090909090909090909090909090909090",
      run_id: "run_alpha",
      plan_id: "plan_alpha",
      binding_kind: "stage_sign_off",
      bound_stage_summary_hash: "sha256:9191919191919191919191919191919191919191919191919191919191919191",
      blocked_scope: {
        scope_kind: "workspace",
        run_id: "run_alpha",
        action_kind: "stage_summary_sign_off",
      },
      broker_correlation: { request_id: "missing-workspace-id" },
      idempotency_key: "wait-enter-missing-workspace-id",
    }),
    (error) => error instanceof InvalidApprovalWaitError,
  );

  await assert.rejects(
    () => store.enterApprovalWait({
      approval_id: "sha256:a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2a2",
      run_id: "run_alpha",
      plan_id: "plan_alpha",
      binding_kind: "stage_sign_off",
      bound_stage_summary_hash: "sha256:b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3b3",
      blocked_scope: {
        scope_kind: "run",
        run_id: "run_beta",
        action_kind: "stage_summary_sign_off",
      },
      broker_correlation: { request_id: "mismatched-run-id" },
      idempotency_key: "wait-enter-mismatched-run-id",
    }),
    (error) => error instanceof InvalidApprovalWaitError,
  );

  await assert.rejects(
    () => store.enterApprovalWait({
      approval_id: "sha256:1234123412341234123412341234123412341234123412341234123412341234",
      run_id: "run_alpha",
      plan_id: "plan_alpha",
      binding_kind: "exact_action",
      bound_action_hash: "sha256:cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd",
      blocked_scope: {
        scope_kind: "action_kind",
        run_id: "run_alpha",
        action_kind: "action_unrecognized",
      },
      broker_correlation: { request_id: "invalid-action-kind" },
      idempotency_key: "wait-enter-invalid-action-kind",
    }),
    (error) => error instanceof InvalidApprovalWaitError,
  );
});

test("writes durable state with private directory and file permissions", async (t) => {
  if (process.platform === "win32") {
    t.skip("permission bit checks are platform-specific");
  }

  const {
    FileDurableStateStore,
  } = await loadRunnerModules();

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-state-"));
  t.after(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });
  await store.appendRecord({ kind: "run_started", idempotency_key: "perm-k1", run_scope_id: "run_alpha" });

  const rootMode = fs.statSync(root).mode & 0o777;
  const snapshotMode = fs.statSync(path.join(root, "snapshot.v2.json")).mode & 0o777;
  const journalMode = fs.statSync(path.join(root, "journal.v2.ndjson")).mode & 0o777;

  assert.equal(rootMode, 0o700);
  assert.equal(snapshotMode, 0o600);
  assert.equal(journalMode, 0o600);
});

test("heals snapshot state from journal after crash window during wait resolution", async (t) => {
  const {
    FileDurableStateStore,
  } = await loadRunnerModules();

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-state-"));
  t.after(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });
  await store.enterApprovalWait({
    approval_id: "sha256:8888888888888888888888888888888888888888888888888888888888888888",
    run_id: "run_alpha",
    plan_id: "plan_alpha",
    binding_kind: "exact_action",
    bound_action_hash: "sha256:9999999999999999999999999999999999999999999999999999999999999999",
    blocked_scope: {
      scope_kind: "run",
      run_id: "run_alpha",
      action_kind: "action_gate_override",
    },
    broker_correlation: {
      request_id: "action-crash-1",
    },
    idempotency_key: "wait-enter-crash-1",
  });

  const snapshotPath = path.join(root, "snapshot.v2.json");
  const snapshot = JSON.parse(fs.readFileSync(snapshotPath, "utf8"));
  snapshot.pending_approval_waits = [snapshot.pending_approval_waits[0]];
  snapshot.last_sequence = 2;
  fs.writeFileSync(snapshotPath, `${JSON.stringify(snapshot, null, 2)}\n`, "utf8");

  await store.resolveApprovalWait({
    approval_id: "sha256:8888888888888888888888888888888888888888888888888888888888888888",
    run_id: "run_alpha",
    plan_id: "plan_alpha",
    binding_kind: "exact_action",
    bound_action_hash: "sha256:9999999999999999999999999999999999999999999999999999999999999999",
    status: "approved",
    idempotency_key: "wait-clear-crash-1",
  });

  const restartedStore = new FileDurableStateStore(root);
  const waits = await restartedStore.listPendingApprovalWaits();
  assert.deepEqual(waits, []);
  const replay = await restartedStore.replayState();
  assert.deepEqual(replay.waits.pending_approval_waits, []);
  assert.deepEqual(replay.waits.resolved_approval_waits, [{
    approval_id: "sha256:8888888888888888888888888888888888888888888888888888888888888888",
    status: "approved",
  }]);
});

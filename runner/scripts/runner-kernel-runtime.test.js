const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const { loadRunnerModules, repoRoot, validRunPlanFixture } = require("./runner-test-helpers.js");

test("loads RunPlan via schema bundle and schedules work", async (t) => {
  const {
    ProtocolSchemaBundle,
    RunPlanLoader,
    PlanScheduler,
  } = await loadRunnerModules();

  const schemaBundle = await ProtocolSchemaBundle.fromProtocolSchemasRoot(path.join(repoRoot, "protocol", "schemas"));
  const loader = new RunPlanLoader(schemaBundle);
  const scheduler = new PlanScheduler();

  const plan = loader.loadFromUnknown(validRunPlanFixture());
  const work = scheduler.listPlannedWork(plan);

  assert.equal(plan.run_id, "run_alpha");
  assert.equal(plan.plan_id, "plan_alpha");
  assert.equal(work.length, 1);
  assert.equal(work[0].entry.entry_kind, "gate_definition");

  t.diagnostic(`scheduled ${work.length} plan entries`);
});

test("scheduler continues unrelated eligible work while exact bound scope remains blocked", async () => {
  const {
    ProtocolSchemaBundle,
    RunPlanLoader,
    PlanScheduler,
    FileDurableStateStore,
  } = await loadRunnerModules();

  const schemaBundle = await ProtocolSchemaBundle.fromProtocolSchemasRoot(path.join(repoRoot, "protocol", "schemas"));
  const loader = new RunPlanLoader(schemaBundle);
  const scheduler = new PlanScheduler();

  const plan = loader.loadFromUnknown(validRunPlanFixture({
    role_instance_ids: ["role_alpha", "role_beta"],
    gate_definitions: [
      ...validRunPlanFixture().gate_definitions,
      {
        ...validRunPlanFixture().gate_definitions[0],
        gate: {
          ...validRunPlanFixture().gate_definitions[0].gate,
          gate_id: "test",
          gate_kind: "test",
          plan_binding: {
            checkpoint_code: "quality",
            order_index: 1,
          },
        },
        order_index: 1,
        role_instance_id: "role_beta",
      },
    ],
  }));

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-state-"));
  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });

  await store.enterApprovalWait({
    approval_id: "sha256:3333333333333333333333333333333333333333333333333333333333333333",
    run_id: "run_alpha",
    plan_id: "plan_alpha",
    binding_kind: "stage_sign_off",
    bound_stage_summary_hash: "sha256:4444444444444444444444444444444444444444444444444444444444444444",
    blocked_scope: {
      scope_kind: "action_kind",
      run_id: "run_alpha",
      role_instance_id: "role_alpha",
      action_kind: "action_gate_override",
    },
    broker_correlation: {
      request_id: "action-approval-role-alpha",
    },
    idempotency_key: "wait-enter-role-alpha",
  });

  const work = scheduler.listPlannedWork(plan, { pending_approval_waits: await store.listPendingApprovalWaits() });
  assert.equal(work.length, 1);
  assert.equal(work[0].entry.role_instance_id, "role_beta");

  fs.rmSync(root, { recursive: true, force: true });
});

test("workspace-scoped waits block all scheduled work", async () => {
  const {
    ProtocolSchemaBundle,
    RunPlanLoader,
    PlanScheduler,
    FileDurableStateStore,
  } = await loadRunnerModules();

  const schemaBundle = await ProtocolSchemaBundle.fromProtocolSchemasRoot(path.join(repoRoot, "protocol", "schemas"));
  const loader = new RunPlanLoader(schemaBundle);
  const scheduler = new PlanScheduler();
  const plan = loader.loadFromUnknown(validRunPlanFixture());

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-state-"));
  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });

  await store.enterApprovalWait({
    approval_id: "sha256:c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4c4",
    run_id: "run_alpha",
    plan_id: "plan_alpha",
    binding_kind: "stage_sign_off",
    bound_stage_summary_hash: "sha256:d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5d5",
    blocked_scope: {
      scope_kind: "workspace",
      workspace_id: "workspace_alpha",
      run_id: "run_alpha",
      action_kind: "stage_summary_sign_off",
    },
    broker_correlation: { request_id: "workspace-block-1" },
    idempotency_key: "wait-enter-workspace-block-1",
  });

  const work = scheduler.listPlannedWork(plan, { pending_approval_waits: await store.listPendingApprovalWaits() });
  assert.deepEqual(work, []);

  fs.rmSync(root, { recursive: true, force: true });
});

test("fails closed when resume resolution binding/hash does not match pending wait", async (t) => {
  const {
    ProtocolSchemaBundle,
    RunPlanLoader,
    RunnerKernel,
    FileDurableStateStore,
    InvalidApprovalWaitError,
    PlanIdentityMismatchError,
  } = await loadRunnerModules();

  const schemaBundle = await ProtocolSchemaBundle.fromProtocolSchemasRoot(path.join(repoRoot, "protocol", "schemas"));
  const loader = new RunPlanLoader(schemaBundle);

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-state-"));
  t.after(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });
  await store.enterApprovalWait({
    approval_id: "sha256:5555555555555555555555555555555555555555555555555555555555555555",
    run_id: "run_alpha",
    plan_id: "plan_alpha",
    binding_kind: "stage_sign_off",
    bound_stage_summary_hash: "sha256:6666666666666666666666666666666666666666666666666666666666666666",
    blocked_scope: {
      scope_kind: "run",
      run_id: "run_alpha",
      action_kind: "stage_summary_sign_off",
    },
    broker_correlation: {
      request_id: "action-wait-1",
    },
    idempotency_key: "wait-enter-5",
  });

  const kernelWrongHash = new RunnerKernel({
    planLoader: loader,
    durableStateStore: store,
    approvalWaitResolver: {
      async resolve(wait) {
        return {
          approval_id: wait.approval_id,
          run_id: wait.run_id,
          plan_id: wait.plan_id,
          status: "approved",
          binding_kind: wait.binding_kind,
          bound_stage_summary_hash: "sha256:7777777777777777777777777777777777777777777777777777777777777777",
        };
      },
    },
  });

  await assert.rejects(
    () => kernelWrongHash.resumeApprovalWaits(),
    (error) => error instanceof InvalidApprovalWaitError,
  );

  const kernelStalePlan = new RunnerKernel({
    planLoader: loader,
    durableStateStore: store,
    approvalWaitResolver: {
      async resolve(wait) {
        return {
          approval_id: wait.approval_id,
          run_id: wait.run_id,
          plan_id: "plan_beta",
          status: "approved",
          binding_kind: wait.binding_kind,
          bound_stage_summary_hash: wait.bound_stage_summary_hash,
        };
      },
    },
  });

  await assert.rejects(
    () => kernelStalePlan.resumeApprovalWaits(),
    (error) => error instanceof PlanIdentityMismatchError,
  );
});

test("runtime seam restores caller details without nesting and only for pending durable waits", async (t) => {
  const {
    FileDurableStateStore,
    DurableRuntimeSeam,
  } = await loadRunnerModules();

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-runtime-"));
  t.after(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });
  await store.enterApprovalWait({
    approval_id: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    run_id: "run_alpha",
    plan_id: "plan_alpha",
    binding_kind: "exact_action",
    bound_action_hash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    blocked_scope: {
      scope_kind: "step",
      run_id: "run_alpha",
      step_id: "step_alpha",
      action_kind: "action_gate_override",
    },
    broker_correlation: {
      request_id: "action-runtime-1",
    },
    idempotency_key: "wait-enter-runtime-1",
  });

  const seam = new DurableRuntimeSeam(store);
  await seam.parkWait({
    identity: { run_id: "run_alpha", plan_id: "plan_alpha" },
    wait_kind: "approval",
    wait_id: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    resume_token: "token-runtime-1",
    idempotency_key: "park-runtime-1",
    details: { reason: "awaiting_signoff" },
  });

  const restored = await seam.restoreWaits({ run_id: "run_alpha", plan_id: "plan_alpha" });
  assert.equal(restored.length, 1);
  assert.deepEqual(restored[0].details, { reason: "awaiting_signoff" });

  await store.resolveApprovalWait({
    approval_id: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    run_id: "run_alpha",
    plan_id: "plan_alpha",
    binding_kind: "exact_action",
    bound_action_hash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    status: "approved",
    idempotency_key: "clear-runtime-1",
  });

  const afterClear = await seam.restoreWaits({ run_id: "run_alpha", plan_id: "plan_alpha" });
  assert.deepEqual(afterClear, []);
});

test("kernel resumeApprovalWaits returns explicit cleared statuses", async (t) => {
  const {
    ProtocolSchemaBundle,
    RunPlanLoader,
    RunnerKernel,
    FileDurableStateStore,
  } = await loadRunnerModules();

  const schemaBundle = await ProtocolSchemaBundle.fromProtocolSchemasRoot(path.join(repoRoot, "protocol", "schemas"));
  const loader = new RunPlanLoader(schemaBundle);
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-state-"));
  t.after(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });
  await store.enterApprovalWait({
    approval_id: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
    run_id: "run_alpha",
    plan_id: "plan_alpha",
    binding_kind: "stage_sign_off",
    bound_stage_summary_hash: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
    blocked_scope: {
      scope_kind: "run",
      run_id: "run_alpha",
      action_kind: "stage_summary_sign_off",
    },
    broker_correlation: {
      request_id: "action-status-1",
    },
    idempotency_key: "wait-enter-status-1",
  });

  const kernel = new RunnerKernel({
    planLoader: loader,
    durableStateStore: store,
    approvalWaitResolver: {
      async resolve(wait) {
        return {
          approval_id: wait.approval_id,
          run_id: wait.run_id,
          plan_id: wait.plan_id,
          status: "denied",
          binding_kind: wait.binding_kind,
          bound_stage_summary_hash: wait.bound_stage_summary_hash,
        };
      },
    },
  });

  const result = await kernel.resumeApprovalWaits();
  assert.deepEqual(result.cleared_waits, [{
    approval_id: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
    status: "denied",
  }]);
});

test("report emitter wraps typed request envelopes", async () => {
  const {
    ReportEmitter,
  } = await loadRunnerModules();

  const captured = [];
  const emitter = new ReportEmitter({
    async sendRunnerCheckpointReport(request) {
      captured.push(request);
      return { accepted: true };
    },
    async sendRunnerResultReport(request) {
      captured.push(request);
      return { accepted: true };
    },
  });

  await emitter.emitCheckpointReport({
    request_id: "req-1",
    identity: {
      run_id: "run_alpha",
      plan_id: "plan_alpha",
      stage_id: "stage_alpha",
      step_attempt_id: "step_attempt_alpha",
    },
    report: {
      lifecycle_state: "active",
      checkpoint_code: "gate_running",
      occurred_at: "2026-01-01T00:00:00Z",
      idempotency_key: "cp-1",
    },
  });

  assert.equal(captured.length, 1);
  assert.equal(captured[0].schema_id, "runecode.protocol.v0.RunnerCheckpointReportRequest");
  assert.equal(captured[0].run_id, "run_alpha");
  assert.equal(captured[0].report.schema_id, "runecode.protocol.v0.RunnerCheckpointReport");
  assert.equal(captured[0].report.step_attempt_id, "step_attempt_alpha");
});

test("kernel runtime seam restores parked waits for active plan identity", async (t) => {
  const {
    FileDurableStateStore,
    DurableRuntimeSeam,
  } = await loadRunnerModules();

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-runtime-"));
  t.after(() => {
    fs.rmSync(root, { recursive: true, force: true });
  });

  const store = new FileDurableStateStore(root);
  await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });

  const seam = new DurableRuntimeSeam(store);
  const identity = { run_id: "run_alpha", plan_id: "plan_alpha", step_id: "step_alpha" };

  await store.enterApprovalWait({
    approval_id: "approval-2",
    run_id: "run_alpha",
    plan_id: "plan_alpha",
    binding_kind: "exact_action",
    bound_action_hash: "sha256:abababababababababababababababababababababababababababababababab",
    blocked_scope: {
      scope_kind: "step",
      run_id: "run_alpha",
      step_id: "step_alpha",
      action_kind: "action_gate_override",
    },
    broker_correlation: {
      request_id: "action-runtime-2",
    },
    idempotency_key: "wait-enter-runtime-2",
  });

  await seam.checkpoint({
    identity,
    checkpoint_code: "active_step",
    idempotency_key: "runtime-checkpoint-1",
  });

  await seam.parkWait({
    identity,
    wait_kind: "approval",
    wait_id: "approval-1",
    resume_token: "token-1",
    idempotency_key: "runtime-wait-1",
    details: { reason: "awaiting_signoff" },
  });

  await seam.parkWait({
    identity,
    wait_kind: "approval",
    wait_id: "approval-2",
    resume_token: "token-2",
    idempotency_key: "runtime-wait-2",
  });

  await seam.resumeWait({
    identity,
    wait_id: "approval-1",
    idempotency_key: "runtime-resume-1",
  });

  const restored = await seam.restoreWaits({ run_id: "run_alpha", plan_id: "plan_alpha" });
  assert.equal(restored.length, 1);
  assert.equal(restored[0].wait_id, "approval-2");
  assert.equal(restored[0].resume_token, "token-2");
});

test("kernel composes modules with plan-bound identity", async () => {
  const {
    RunnerKernel,
  } = await loadRunnerModules();

  const calls = [];
  const runtime = {
    async checkpoint(input) {
      calls.push({ kind: "checkpoint", input });
    },
    async parkWait(input) {
      calls.push({ kind: "park", input });
    },
    async resumeWait(input) {
      calls.push({ kind: "resume", input });
    },
    async restoreWaits() {
      return [];
    },
  };

  const kernel = new RunnerKernel({
    planLoader: { loadFromFile: async () => { throw new Error("unused"); }, identityOf: () => ({ run_id: "r", plan_id: "p" }) },
    durableStateStore: {
      bindPlanIdentity: async () => {},
      appendRecord: async () => ({ sequence: 1 }),
      readState: async () => ({
        snapshot: {
          schema_version: "2",
          run_id: "run_alpha",
          plan_id: "plan_alpha",
          last_sequence: 0,
          pending_approval_waits: [],
          created_at: "2026-01-01T00:00:00Z",
          updated_at: "2026-01-01T00:00:00Z",
        },
        journal: [],
      }),
      runtimeStateRoot: () => process.cwd(),
      listPendingApprovalWaits: async () => [],
    },
    runtimeSeam: runtime,
  });

  const identity = { run_id: "run_alpha", plan_id: "plan_alpha", step_id: "step_alpha" };
  await kernel.composeModules(identity, [
    {
      name: "first",
      async run(context) {
        assert.equal(context.identity.plan_id, "plan_alpha");
        await context.runtime.parkWait({
          identity: context.identity,
          wait_kind: "approval",
          wait_id: "approval-1",
          resume_token: "token-1",
          idempotency_key: "park-1",
        });
      },
    },
  ]);

  assert.equal(calls.length, 1);
  assert.equal(calls[0].kind, "park");
  assert.equal(calls[0].input.identity.run_id, "run_alpha");
});

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const repoRoot = path.resolve(__dirname, "..", "..");

async function loadRunnerModules() {
  return import("../src/index.ts");
}

function validRunPlanFixture(overrides = {}) {
  return {
    schema_id: "runecode.protocol.v0.RunPlan",
    schema_version: "0.1.0",
    plan_id: "plan_alpha",
    run_id: "run_alpha",
    workflow_id: "workflow_alpha",
    process_id: "process_alpha",
    workflow_definition_hash: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    process_definition_hash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    policy_context_hash: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
    compiled_at: "2026-01-01T00:00:00Z",
    role_instance_ids: ["role_alpha"],
    executor_bindings: [
      {
        binding_id: "binding_alpha",
        executor_id: "executor_alpha",
        executor_class: "workspace_ordinary",
        allowed_role_kinds: ["developer"],
      },
    ],
    gate_definitions: [
      {
        schema_id: "runecode.protocol.v0.GateDefinition",
        schema_version: "0.1.0",
        gate: {
          schema_id: "runecode.protocol.v0.GateContract",
          schema_version: "0.1.0",
          gate_id: "lint",
          gate_kind: "lint",
          gate_version: "0.1.0",
          normalized_inputs: [],
          plan_binding: {
            checkpoint_code: "quality",
            order_index: 0,
          },
          retry_semantics: {
            retry_mode: "new_attempt_required",
            max_attempts: 2,
          },
          override_semantics: {
            override_mode: "policy_action_required",
            action_kind: "action_gate_override",
            approval_trigger_code: "gate_override",
          },
        },
        checkpoint_code: "quality",
        order_index: 0,
        role_instance_id: "role_alpha",
        executor_binding_id: "binding_alpha",
      },
    ],
    ...overrides,
  };
}

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
  await store.appendRecord({ kind: "run_started", idempotency_key: "k1" });

  await assert.rejects(
    () => store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_beta" }),
    (error) => error instanceof PlanIdentityMismatchError,
  );
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
    run_id: "run_alpha",
    report: { schema_id: "runecode.protocol.v0.RunnerCheckpointReport" },
  });

  assert.equal(captured.length, 1);
  assert.equal(captured[0].schema_id, "runecode.protocol.v0.RunnerCheckpointReportRequest");
  assert.equal(captured[0].run_id, "run_alpha");
});

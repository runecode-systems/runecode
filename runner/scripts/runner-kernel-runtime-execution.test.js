const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const { loadRunnerModules, repoRoot, validRunPlanFixture } = require("./runner-test-helpers.js");

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

test("kernel executes scheduled gate entries and fails closed on rejected reports", async () => {
  const {
    ProtocolSchemaBundle,
    RunPlanLoader,
    RunnerKernel,
    FileDurableStateStore,
  } = await loadRunnerModules();

  const schemaBundle = await ProtocolSchemaBundle.fromProtocolSchemasRoot(path.join(repoRoot, "protocol", "schemas"));
  const loader = new RunPlanLoader(schemaBundle);

  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-execute-"));
  try {
    const store = new FileDurableStateStore(root);
    const acceptedRequests = [];
    const kernel = new RunnerKernel({
      planLoader: loader,
      durableStateStore: store,
      brokerClient: {
        async requestDependencyCacheHandoff(request) {
          acceptedRequests.push({ kind: "handoff", request });
          return {
            schema_id: "runecode.protocol.v0.DependencyCacheHandoffResponse",
            schema_version: "0.1.0",
            request_id: request.request_id,
            found: true,
            handoff: {
              schema_id: "runecode.protocol.v0.DependencyCacheHandoffMetadata",
              schema_version: "0.1.0",
              request_digest: request.request_digest,
              resolved_unit_digest: { hash_alg: "sha256", hash: "e".repeat(64) },
              manifest_digest: { hash_alg: "sha256", hash: "f".repeat(64) },
              payload_digests: [{ hash_alg: "sha256", hash: "1".repeat(64) }],
              materialization_mode: "derived_read_only",
              handoff_mode: "broker_internal_artifact_handoff",
            },
          };
        },
        async sendRunnerCheckpointReport(request) {
          acceptedRequests.push({ kind: "checkpoint", request });
          return { accepted: true };
        },
        async sendRunnerResultReport(request) {
          acceptedRequests.push({ kind: "result", request });
          return { accepted: true };
        },
      },
    });

    const planPath = path.join(root, "runplan.json");
    fs.writeFileSync(planPath, JSON.stringify(validRunPlanFixture(), null, 2));

    const execution = await kernel.executeScheduledWorkFromPlanFile(planPath);
    assert.equal(execution.work.length, 1);
    assert.equal(execution.executed.length, 1);
    assert.equal(execution.executed[0].entry_id, "quality_lint");
    assert.equal(execution.executed[0].outcome.status, "ok");
    assert.deepEqual(acceptedRequests.map((entry) => entry.kind), ["handoff", "checkpoint", "result"]);
    assert.equal(acceptedRequests[1].request.report.gate_lifecycle_state, "running");
    assert.equal(acceptedRequests[2].request.report.gate_lifecycle_state, "passed");

    const rejectingKernel = new RunnerKernel({
      planLoader: loader,
      durableStateStore: new FileDurableStateStore(path.join(root, "reject-state")),
      brokerClient: {
        async requestDependencyCacheHandoff(request) {
          return {
            schema_id: "runecode.protocol.v0.DependencyCacheHandoffResponse",
            schema_version: "0.1.0",
            request_id: request.request_id,
            found: true,
            handoff: {
              schema_id: "runecode.protocol.v0.DependencyCacheHandoffMetadata",
              schema_version: "0.1.0",
              request_digest: request.request_digest,
              resolved_unit_digest: { hash_alg: "sha256", hash: "e".repeat(64) },
              manifest_digest: { hash_alg: "sha256", hash: "f".repeat(64) },
              payload_digests: [{ hash_alg: "sha256", hash: "1".repeat(64) }],
              materialization_mode: "derived_read_only",
              handoff_mode: "broker_internal_artifact_handoff",
            },
          };
        },
        async sendRunnerCheckpointReport() {
          return { accepted: false, reason: "broker rejected report at lifecycle active" };
        },
        async sendRunnerResultReport() {
          return { accepted: true };
        },
      },
    });

    await assert.rejects(
      () => rejectingKernel.executeScheduledWorkFromPlanFile(planPath),
      /broker rejected report at lifecycle active/,
    );
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("kernel fails closed when a valid plan produces no scheduled work", async () => {
  const {
    ProtocolSchemaBundle,
    RunPlanLoader,
    FileDurableStateStore,
    RunnerKernel,
  } = await loadRunnerModules();

  const schemaBundle = await ProtocolSchemaBundle.fromProtocolSchemasRoot(path.join(repoRoot, "protocol", "schemas"));
  const loader = new RunPlanLoader(schemaBundle);
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-runtime-"));
  try {
    const planPath = path.join(root, "runplan.json");
    fs.writeFileSync(planPath, JSON.stringify(validRunPlanFixture(), null, 2));
    const stateRoot = path.join(root, "state");
    const store = new FileDurableStateStore(stateRoot);
    await store.bindPlanIdentity({ run_id: "run_alpha", plan_id: "plan_alpha" });
    await store.enterApprovalWait({
      approval_id: "approval-block-all",
      run_id: "run_alpha",
      plan_id: "plan_alpha",
      binding_kind: "exact_action",
      bound_action_hash: "sha256:" + "a".repeat(64),
      blocked_scope: { scope_kind: "run", run_id: "run_alpha", action_kind: "action_gate_override" },
      broker_correlation: { request_id: "req-block-all" },
      idempotency_key: "approval-block-all",
    });

    const kernel = new RunnerKernel({
      planLoader: loader,
      durableStateStore: store,
      brokerClient: {
        async requestDependencyCacheHandoff() {
          throw new Error("unused");
        },
        async sendRunnerCheckpointReport() {
          throw new Error("unused");
        },
        async sendRunnerResultReport() {
          throw new Error("unused");
        },
      },
    });

    await assert.rejects(
      () => kernel.executeScheduledWorkFromPlanFile(planPath),
      /produced no scheduled work/,
    );
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("stdio broker client validates typed transport responses", async () => {
  const {
    ProtocolSchemaBundle,
    StdioRunnerBrokerClient,
  } = await loadRunnerModules();

  const schemaBundle = await ProtocolSchemaBundle.fromProtocolSchemasRoot(path.join(repoRoot, "protocol", "schemas"));
  const { PassThrough } = require("node:stream");
  const input = new PassThrough();
  const output = new PassThrough();
  const writes = [];

  output.on("data", (chunk) => {
    writes.push(chunk.toString("utf8"));
  });

  const client = new StdioRunnerBrokerClient({
    schemaBundle,
    input,
    output,
  });

  input.end(`${JSON.stringify({
    message_type: "runner_checkpoint_report_response",
    payload: {
      schema_id: "runecode.protocol.v0.RunnerCheckpointReportResponse",
      schema_version: "0.1.0",
      request_id: "req-stdio-1",
      run_id: "run_alpha",
      accepted: true,
      canonical_lifecycle_state: "active",
      accepted_at: "2026-01-01T00:00:00Z",
      idempotency_key: "cp-stdio-1",
    },
  })}\n`);

  const ack = await client.sendRunnerCheckpointReport({
    schema_id: "runecode.protocol.v0.RunnerCheckpointReportRequest",
    schema_version: "0.1.0",
    request_id: "req-stdio-1",
    run_id: "run_alpha",
    report: {
      schema_id: "runecode.protocol.v0.RunnerCheckpointReport",
      schema_version: "0.1.0",
      lifecycle_state: "active",
      checkpoint_code: "quality",
      occurred_at: "2026-01-01T00:00:00Z",
      idempotency_key: "cp-stdio-1",
    },
  });

  assert.deepEqual(ack, { accepted: true });
  assert.equal(writes.length, 1);
  const outbound = JSON.parse(writes[0]);
  assert.equal(outbound.message_type, "runner_checkpoint_report_request");
  assert.equal(outbound.payload.schema_id, "runecode.protocol.v0.RunnerCheckpointReportRequest");
});

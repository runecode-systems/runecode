const assert = require("node:assert/strict");
const test = require("node:test");

const { loadRunnerModules } = require("./runner-test-helpers.js");

test("noop broker client returns unaccepted acknowledgements", async () => {
  const {
    NoopRunnerBrokerClient,
  } = await loadRunnerModules();

  const client = new NoopRunnerBrokerClient();
  const checkpoint = await client.sendRunnerCheckpointReport({
    schema_id: "runecode.protocol.v0.RunnerCheckpointReportRequest",
    schema_version: "0.1.0",
    request_id: "noop-checkpoint",
    run_id: "run_alpha",
    report: {
      schema_id: "runecode.protocol.v0.RunnerCheckpointReport",
      schema_version: "0.1.0",
      lifecycle_state: "active",
      checkpoint_code: "gate_running",
      occurred_at: "2026-01-01T00:00:00Z",
      idempotency_key: "noop-cp-1",
    },
  });
  const result = await client.sendRunnerResultReport({
    schema_id: "runecode.protocol.v0.RunnerResultReportRequest",
    schema_version: "0.1.0",
    request_id: "noop-result",
    run_id: "run_alpha",
    report: {
      schema_id: "runecode.protocol.v0.RunnerResultReport",
      schema_version: "0.1.0",
      lifecycle_state: "completed",
      result_code: "step_succeeded",
      occurred_at: "2026-01-01T00:00:00Z",
      idempotency_key: "noop-result-1",
    },
  });

  assert.deepEqual(checkpoint, { accepted: false, reason: "broker client not configured" });
  assert.deepEqual(result, { accepted: false, reason: "broker client not configured" });
});

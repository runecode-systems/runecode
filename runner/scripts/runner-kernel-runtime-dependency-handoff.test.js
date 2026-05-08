const assert = require("node:assert/strict");
const { createHash } = require("node:crypto");
const test = require("node:test");

const { loadRunnerModules } = require("./runner-test-helpers.js");

test("kernel fails closed when a required dependency cache handoff is missing", async () => {
  const {
    RunnerKernel,
  } = await loadRunnerModules();

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
    brokerClient: {
      async requestDependencyCacheHandoff(request) {
        return {
          schema_id: "runecode.protocol.v0.DependencyCacheHandoffResponse",
          schema_version: "0.1.0",
          request_id: request.request_id,
          found: false,
        };
      },
      async sendRunnerCheckpointReport() {
        return { accepted: false, reason: "unused" };
      },
      async sendRunnerResultReport() {
        return { accepted: false, reason: "unused" };
      },
    },
  });

  await assert.rejects(
    () => kernel.composeEntryModules({ run_id: "run_alpha", plan_id: "plan_alpha" }, {
      entry_id: "entry-1",
      entry_kind: "gate",
      dependency_cache_handoffs: [{
        request_digest: "sha256:" + "d".repeat(64),
        consumer_role: "workspace",
        required: true,
      }],
    }, [{ name: "noop", async run() {} }]),
    /required dependency cache handoff not found/,
  );
});

test("kernel emits stable unique dependency cache handoff request ids without truncation assumptions", async () => {
  const {
    RunnerKernel,
  } = await loadRunnerModules();

  const captured = [];
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
    brokerClient: {
      async requestDependencyCacheHandoff(request) {
        captured.push(request);
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
        return { accepted: true };
      },
      async sendRunnerResultReport() {
        return { accepted: true };
      },
    },
  });

  const longRunIDA = `run_${"shared-prefix-".repeat(12)}A`;
  const longRunIDB = `run_${"shared-prefix-".repeat(12)}B`;
  const digestA = `sha256:${"a".repeat(64)}`;
  const digestB = `sha256:${"b".repeat(64)}`;
  const expectedRequestID = (runID, requestDigest) => `dependency-handoff:${createHash("sha256").update(runID).update("\n").update(requestDigest).digest("hex")}`;

  await kernel.composeModules(
    { run_id: longRunIDA, plan_id: "plan_alpha" },
    [{ name: "noop-a", async run() {} }],
    [
      { request_digest: digestA, consumer_role: "workspace", required: true },
      { request_digest: digestB, consumer_role: "workspace", required: true },
    ],
  );
  await kernel.composeModules(
    { run_id: longRunIDA, plan_id: "plan_alpha" },
    [{ name: "noop-b", async run() {} }],
    [{ request_digest: digestA, consumer_role: "workspace", required: true }],
  );
  await kernel.composeModules(
    { run_id: longRunIDB, plan_id: "plan_alpha" },
    [{ name: "noop-c", async run() {} }],
    [{ request_digest: digestA, consumer_role: "workspace", required: true }],
  );

  const requestIDs = captured.map((request) => request.request_id);
  assert.deepEqual(requestIDs, [
    expectedRequestID(longRunIDA, digestA),
    expectedRequestID(longRunIDA, digestB),
    expectedRequestID(longRunIDA, digestA),
    expectedRequestID(longRunIDB, digestA),
  ]);
  assert.equal(requestIDs[0], requestIDs[2]);
  assert.notEqual(requestIDs[0], requestIDs[1]);
  assert.notEqual(requestIDs[0], requestIDs[3]);
  assert.equal(requestIDs[0].length, "dependency-handoff:".length + 64);
});

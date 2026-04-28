const assert = require("node:assert/strict");
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

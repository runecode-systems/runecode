const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");
const { spawnSync } = require("node:child_process");

const scriptPath = path.join(__dirname, "perf-runner-workflow.js");

function writeRunPlan(root, overrides = {}) {
  const runPlan = {
    schema_id: "runecode.protocol.v0.RunPlan",
    schema_version: "0.4.0",
    plan_id: "plan_workflow_first_party_minimal",
    run_id: "run_workflow_first_party_minimal",
    workflow_id: "workflow_first_party_minimal",
    workflow_version: "1.0.0",
    process_id: "process_first_party_minimal",
    approval_profile: "moderate",
    autonomy_posture: "balanced",
    workflow_definition_hash: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    process_definition_hash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    policy_context_hash: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
    compiled_at: "2026-01-01T00:00:00Z",
    role_instance_ids: ["role_alpha"],
    executor_bindings: [{
      binding_id: "binding_alpha",
      executor_id: "executor_alpha",
      executor_class: "workspace_ordinary",
      allowed_role_kinds: ["developer"],
    }],
    gate_definitions: [{
      schema_id: "runecode.protocol.v0.GateDefinition",
      schema_version: "0.2.0",
      gate: {
        schema_id: "runecode.protocol.v0.GateContract",
        schema_version: "0.1.0",
        gate_id: "lint",
        gate_kind: "lint",
        gate_version: "0.1.0",
        normalized_inputs: [],
        plan_binding: { checkpoint_code: "quality", order_index: 0 },
        retry_semantics: { retry_mode: "new_attempt_required", max_attempts: 2 },
        override_semantics: { override_mode: "policy_action_required", action_kind: "action_gate_override", approval_trigger_code: "gate_override" },
      },
      checkpoint_code: "quality",
      order_index: 0,
      stage_id: "quality_stage",
      step_id: "quality_lint",
      role_instance_id: "role_alpha",
      executor_binding_id: "binding_alpha",
      dependency_cache_handoffs: [{
        request_digest: { hash_alg: "sha256", hash: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd" },
        consumer_role: "workspace",
        required: true,
      }],
    }],
    dependency_edges: [],
    entries: [{
      entry_id: "quality_lint",
      entry_kind: "gate",
      order_index: 0,
      stage_id: "quality_stage",
      step_id: "quality_lint",
      role_instance_id: "role_alpha",
      executor_binding_id: "binding_alpha",
      checkpoint_code: "quality",
      gate: {
        schema_id: "runecode.protocol.v0.GateContract",
        schema_version: "0.1.0",
        gate_id: "lint",
        gate_kind: "lint",
        gate_version: "0.1.0",
        normalized_inputs: [],
        plan_binding: { checkpoint_code: "quality", order_index: 0 },
        retry_semantics: { retry_mode: "new_attempt_required", max_attempts: 2 },
        override_semantics: { override_mode: "policy_action_required", action_kind: "action_gate_override", approval_trigger_code: "gate_override" },
      },
      dependency_cache_handoffs: [{
        request_digest: { hash_alg: "sha256", hash: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd" },
        consumer_role: "workspace",
        required: true,
      }],
      depends_on_entry_ids: [],
      blocks_entry_ids: [],
      supported_wait_kinds: ["waiting_operator_input", "waiting_approval"],
    }],
    ...overrides,
  };
  const runplanPath = path.join(root, "runplan.json");
  fs.writeFileSync(runplanPath, JSON.stringify(runPlan, null, 2));
  return runplanPath;
}

function runPerf(mode, runplanPath, fixture) {
  const args = ["--experimental-strip-types", scriptPath, "--mode", mode, "--runplan", runplanPath];
  if (fixture) {
    args.push("--fixture", fixture);
  }
  return spawnSync(process.execPath, args, { encoding: "utf8" });
}

test("workflow-path requires supported first-party fixture argument", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-perf-workflow-"));
  try {
    const runplanPath = writeRunPlan(root);
    const result = runPerf("workflow-path", runplanPath, "");
    assert.equal(result.status, 1);
    assert.match(result.stderr, /requires --fixture workflow\.first-party-minimal\.v1/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("workflow-path rejects supported fixture when no work is schedulable", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-perf-workflow-"));
  try {
    const runplanPath = writeRunPlan(root, {
      dependency_edges: [{
        dependency_kind: "step_completed",
        upstream_step_id: "quality_lint",
        downstream_step_id: "quality_lint",
      }],
      entries: [{
        entry_id: "quality_lint",
        entry_kind: "gate",
        order_index: 0,
        stage_id: "quality_stage",
        step_id: "quality_lint",
        role_instance_id: "role_alpha",
        executor_binding_id: "binding_alpha",
        checkpoint_code: "quality",
        gate: {
          schema_id: "runecode.protocol.v0.GateContract",
          schema_version: "0.1.0",
          gate_id: "lint",
          gate_kind: "lint",
          gate_version: "0.1.0",
          normalized_inputs: [],
          plan_binding: { checkpoint_code: "quality", order_index: 0 },
          retry_semantics: { retry_mode: "new_attempt_required", max_attempts: 2 },
          override_semantics: { override_mode: "policy_action_required", action_kind: "action_gate_override", approval_trigger_code: "gate_override" },
        },
        dependency_cache_handoffs: [{
          request_digest: { hash_alg: "sha256", hash: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd" },
          consumer_role: "workspace",
          required: true,
        }],
        depends_on_entry_ids: ["quality_lint"],
        blocks_entry_ids: ["quality_lint"],
        supported_wait_kinds: ["waiting_operator_input", "waiting_approval"],
      }],
    });
    const result = runPerf("workflow-path", runplanPath, "workflow.first-party-minimal.v1");
    assert.equal(result.status, 1);
    assert.match(result.stderr, /no schedulable work on supported path/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("first-party-beta rejects non-supported runplan identity", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-perf-workflow-"));
  try {
    const runplanPath = writeRunPlan(root, { workflow_id: "workflow_other" });
    const result = runPerf("first-party-beta", runplanPath, "workflow.first-party-minimal.v1");
    assert.equal(result.status, 1);
    assert.match(result.stderr, /requires workflow_id workflow_first_party_minimal/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("first-party-beta accepts supported fixture and runplan", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-perf-workflow-"));
  try {
    const runplanPath = writeRunPlan(root);
    const result = runPerf("first-party-beta", runplanPath, "workflow.first-party-minimal.v1");
    assert.equal(result.status, 0);
    assert.match(result.stdout.trim(), /^\d+$/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

const path = require("node:path");

const repoRoot = path.resolve(__dirname, "..", "..");

async function loadRunnerModules() {
  return import("../src/index.ts");
}

function validRunPlanFixture(overrides = {}) {
  return {
    schema_id: "runecode.protocol.v0.RunPlan",
    schema_version: "0.4.0",
    plan_id: "plan_alpha",
    run_id: "run_alpha",
    workflow_id: "workflow_alpha",
    workflow_version: "1.0.0",
    process_id: "process_alpha",
    approval_profile: "moderate",
    autonomy_posture: "balanced",
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
        schema_version: "0.2.0",
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
        stage_id: "quality_stage",
        step_id: "quality_lint",
        role_instance_id: "role_alpha",
        executor_binding_id: "binding_alpha",
        dependency_cache_handoffs: [
          {
            request_digest: { hash_alg: "sha256", hash: "d".repeat(64) },
            consumer_role: "workspace",
            required: true,
          },
        ],
      },
    ],
    dependency_edges: [],
    entries: [
      {
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
        dependency_cache_handoffs: [
          {
            request_digest: { hash_alg: "sha256", hash: "d".repeat(64) },
            consumer_role: "workspace",
            required: true,
          },
        ],
        depends_on_entry_ids: [],
        blocks_entry_ids: [],
        supported_wait_kinds: ["waiting_operator_input", "waiting_approval"],
      },
    ],
    ...overrides,
  };
}

module.exports = {
  loadRunnerModules,
  repoRoot,
  validRunPlanFixture,
};

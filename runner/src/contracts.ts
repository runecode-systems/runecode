/**
 * Internal runner contract types.
 *
 * These types keep request/report shapes aligned with protocol schema families
 * while preserving broker-owned truth and immutable plan identity.
 */

export const RUNNER_CHECKPOINT_REPORT_SCHEMA_ID = "runecode.protocol.v0.RunnerCheckpointReport";
export const RUNNER_RESULT_REPORT_SCHEMA_ID = "runecode.protocol.v0.RunnerResultReport";
export const RUNNER_CHECKPOINT_REPORT_REQUEST_SCHEMA_ID = "runecode.protocol.v0.RunnerCheckpointReportRequest";
export const RUNNER_RESULT_REPORT_REQUEST_SCHEMA_ID = "runecode.protocol.v0.RunnerResultReportRequest";
export const DEPENDENCY_CACHE_HANDOFF_REQUEST_SCHEMA_ID = "runecode.protocol.v0.DependencyCacheHandoffRequest";
export const DEPENDENCY_CACHE_HANDOFF_RESPONSE_SCHEMA_ID = "runecode.protocol.v0.DependencyCacheHandoffResponse";
export const DEPENDENCY_CACHE_HANDOFF_METADATA_SCHEMA_ID = "runecode.protocol.v0.DependencyCacheHandoffMetadata";
export const RUNNER_CONTRACT_SCHEMA_VERSION = "0.1.0";

/**
 * These identifiers mirror authoritative protocol schema IDs and stay local to
 * typed runner helper surfaces. The protocol schema bundle remains the source
 * of truth for cross-boundary validation.
 */

export type PlanBoundExecutionIdentity = {
  run_id: string;
  plan_id: string;
  stage_id?: string;
  step_id?: string;
  role_instance_id?: string;
  stage_attempt_id?: string;
  step_attempt_id?: string;
  gate_attempt_id?: string;
};

export type RunnerCheckpointReport = {
  schema_id: typeof RUNNER_CHECKPOINT_REPORT_SCHEMA_ID;
  schema_version: typeof RUNNER_CONTRACT_SCHEMA_VERSION;
  lifecycle_state: "pending" | "starting" | "active" | "blocked" | "recovering";
  checkpoint_code: string;
  occurred_at: string;
  idempotency_key: string;
  plan_checkpoint_code?: string;
  plan_order_index?: number;
  gate_evidence_ref?: string;
  stage_id?: string;
  step_id?: string;
  role_instance_id?: string;
  stage_attempt_id?: string;
  step_attempt_id?: string;
  gate_attempt_id?: string;
  gate_id?: string;
  gate_kind?: "build" | "test" | "lint" | "format" | "secret_scan" | "policy";
  gate_version?: string;
  gate_lifecycle_state?: "planned" | "running" | "passed" | "failed" | "overridden" | "superseded";
  normalized_input_digests?: string[];
  pending_approval_count?: number;
  details?: Record<string, unknown>;
};

export type RunnerResultReport = {
  schema_id: typeof RUNNER_RESULT_REPORT_SCHEMA_ID;
  schema_version: typeof RUNNER_CONTRACT_SCHEMA_VERSION;
  lifecycle_state: "completed" | "failed" | "cancelled";
  result_code: string;
  occurred_at: string;
  idempotency_key: string;
  plan_checkpoint_code?: string;
  plan_order_index?: number;
  gate_evidence_ref?: string;
  gate_evidence?: Record<string, unknown>;
  stage_id?: string;
  step_id?: string;
  role_instance_id?: string;
  stage_attempt_id?: string;
  step_attempt_id?: string;
  gate_attempt_id?: string;
  gate_id?: string;
  gate_kind?: "build" | "test" | "lint" | "format" | "secret_scan" | "policy";
  gate_version?: string;
  gate_lifecycle_state?: "passed" | "failed" | "overridden" | "superseded";
  normalized_input_digests?: string[];
  failure_reason_code?: string;
  overridden_failed_result_ref?: string;
  details?: Record<string, unknown>;
};

export type RunnerCheckpointReportRequest = {
  schema_id: typeof RUNNER_CHECKPOINT_REPORT_REQUEST_SCHEMA_ID;
  schema_version: typeof RUNNER_CONTRACT_SCHEMA_VERSION;
  request_id: string;
  run_id: string;
  report: RunnerCheckpointReport;
};

export type RunnerResultReportRequest = {
  schema_id: typeof RUNNER_RESULT_REPORT_REQUEST_SCHEMA_ID;
  schema_version: typeof RUNNER_CONTRACT_SCHEMA_VERSION;
  request_id: string;
  run_id: string;
  report: RunnerResultReport;
};

export type DependencyCacheHandoffRequest = {
	schema_id: typeof DEPENDENCY_CACHE_HANDOFF_REQUEST_SCHEMA_ID;
	schema_version: typeof RUNNER_CONTRACT_SCHEMA_VERSION;
	request_id: string;
	request_digest: { hash_alg: "sha256"; hash: string };
	consumer_role: string;
};

export type DependencyCacheHandoffMetadata = {
	schema_id: typeof DEPENDENCY_CACHE_HANDOFF_METADATA_SCHEMA_ID;
	schema_version: typeof RUNNER_CONTRACT_SCHEMA_VERSION;
	request_digest: { hash_alg: "sha256"; hash: string };
	resolved_unit_digest: { hash_alg: "sha256"; hash: string };
	manifest_digest: { hash_alg: "sha256"; hash: string };
	payload_digests: Array<{ hash_alg: "sha256"; hash: string }>;
	materialization_mode: "derived_read_only";
	handoff_mode: "broker_internal_artifact_handoff";
};

export type DependencyCacheHandoffResponse = {
	schema_id: typeof DEPENDENCY_CACHE_HANDOFF_RESPONSE_SCHEMA_ID;
	schema_version: typeof RUNNER_CONTRACT_SCHEMA_VERSION;
	request_id: string;
	found: boolean;
	handoff?: DependencyCacheHandoffMetadata;
};

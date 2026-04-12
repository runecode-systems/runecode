/**
 * Typed runner checkpoint/result emitter seam.
 *
 * This module stamps request envelopes and delegates delivery to a broker
 * client abstraction without introducing local authority semantics.
 */

import type { RunnerBrokerClient, BrokerAcknowledge } from "./broker-client.ts";
import {
  RUNNER_CHECKPOINT_REPORT_REQUEST_SCHEMA_ID,
  RUNNER_CONTRACT_SCHEMA_VERSION,
  RUNNER_RESULT_REPORT_REQUEST_SCHEMA_ID,
  RUNNER_CHECKPOINT_REPORT_SCHEMA_ID,
  RUNNER_RESULT_REPORT_SCHEMA_ID,
  type PlanBoundExecutionIdentity,
  type RunnerCheckpointReport,
  type RunnerResultReport,
} from "./contracts.ts";

export type CheckpointReportInput = {
  request_id: string;
  identity: PlanBoundExecutionIdentity;
  report: Omit<RunnerCheckpointReport, "schema_id" | "schema_version" | "stage_id" | "step_id" | "role_instance_id" | "stage_attempt_id" | "step_attempt_id" | "gate_attempt_id">;
};

export type ResultReportInput = {
  request_id: string;
  identity: PlanBoundExecutionIdentity;
  report: Omit<RunnerResultReport, "schema_id" | "schema_version" | "stage_id" | "step_id" | "role_instance_id" | "stage_attempt_id" | "step_attempt_id" | "gate_attempt_id">;
};

export class ReportEmitter {
  private readonly brokerClient: RunnerBrokerClient;

  constructor(brokerClient: RunnerBrokerClient) {
    this.brokerClient = brokerClient;
  }

  emitCheckpointReport(input: CheckpointReportInput): Promise<BrokerAcknowledge> {
    const report: RunnerCheckpointReport = {
      ...input.report,
      schema_id: RUNNER_CHECKPOINT_REPORT_SCHEMA_ID,
      schema_version: RUNNER_CONTRACT_SCHEMA_VERSION,
      stage_id: input.identity.stage_id,
      step_id: input.identity.step_id,
      role_instance_id: input.identity.role_instance_id,
      stage_attempt_id: input.identity.stage_attempt_id,
      step_attempt_id: input.identity.step_attempt_id,
      gate_attempt_id: input.identity.gate_attempt_id,
    };

    return this.brokerClient.sendRunnerCheckpointReport({
      schema_id: RUNNER_CHECKPOINT_REPORT_REQUEST_SCHEMA_ID,
      schema_version: RUNNER_CONTRACT_SCHEMA_VERSION,
      request_id: input.request_id,
      run_id: input.identity.run_id,
      report,
    });
  }

  emitResultReport(input: ResultReportInput): Promise<BrokerAcknowledge> {
    const report: RunnerResultReport = {
      ...input.report,
      schema_id: RUNNER_RESULT_REPORT_SCHEMA_ID,
      schema_version: RUNNER_CONTRACT_SCHEMA_VERSION,
      stage_id: input.identity.stage_id,
      step_id: input.identity.step_id,
      role_instance_id: input.identity.role_instance_id,
      stage_attempt_id: input.identity.stage_attempt_id,
      step_attempt_id: input.identity.step_attempt_id,
      gate_attempt_id: input.identity.gate_attempt_id,
    };

    return this.brokerClient.sendRunnerResultReport({
      schema_id: RUNNER_RESULT_REPORT_REQUEST_SCHEMA_ID,
      schema_version: RUNNER_CONTRACT_SCHEMA_VERSION,
      request_id: input.request_id,
      run_id: input.identity.run_id,
      report,
    });
  }
}

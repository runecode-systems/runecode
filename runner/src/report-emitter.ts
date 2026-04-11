/**
 * Typed runner checkpoint/result emitter seam.
 *
 * This module stamps request envelopes and delegates delivery to a broker
 * client abstraction without introducing local authority semantics.
 */

import type { RunnerBrokerClient, BrokerAcknowledge } from "./broker-client.ts";

const RUNNER_CHECKPOINT_REPORT_REQUEST_SCHEMA_ID = "runecode.protocol.v0.RunnerCheckpointReportRequest";
const RUNNER_RESULT_REPORT_REQUEST_SCHEMA_ID = "runecode.protocol.v0.RunnerResultReportRequest";
const REQUEST_SCHEMA_VERSION = "0.1.0";

export type CheckpointReportInput = {
  request_id: string;
  run_id: string;
  report: Record<string, unknown>;
};

export type ResultReportInput = {
  request_id: string;
  run_id: string;
  report: Record<string, unknown>;
};

export class ReportEmitter {
  private readonly brokerClient: RunnerBrokerClient;

  constructor(brokerClient: RunnerBrokerClient) {
    this.brokerClient = brokerClient;
  }

  emitCheckpointReport(input: CheckpointReportInput): Promise<BrokerAcknowledge> {
    return this.brokerClient.sendRunnerCheckpointReport({
      schema_id: RUNNER_CHECKPOINT_REPORT_REQUEST_SCHEMA_ID,
      schema_version: REQUEST_SCHEMA_VERSION,
      request_id: input.request_id,
      run_id: input.run_id,
      report: input.report,
    });
  }

  emitResultReport(input: ResultReportInput): Promise<BrokerAcknowledge> {
    return this.brokerClient.sendRunnerResultReport({
      schema_id: RUNNER_RESULT_REPORT_REQUEST_SCHEMA_ID,
      schema_version: REQUEST_SCHEMA_VERSION,
      request_id: input.request_id,
      run_id: input.run_id,
      report: input.report,
    });
  }
}

/**
 * Broker client seam for runner report delivery.
 *
 * This abstraction isolates transport details while preserving typed protocol
 * request shapes.
 */

export type BrokerAcknowledge = {
  accepted: boolean;
  reason?: string;
};

import type {
  RunnerCheckpointReportRequest,
  RunnerResultReportRequest,
} from "./contracts.ts";

export type RunnerBrokerClient = {
  sendRunnerCheckpointReport(request: RunnerCheckpointReportRequest): Promise<BrokerAcknowledge>;
  sendRunnerResultReport(request: RunnerResultReportRequest): Promise<BrokerAcknowledge>;
};

export class NoopRunnerBrokerClient implements RunnerBrokerClient {
  async sendRunnerCheckpointReport(_request: RunnerCheckpointReportRequest): Promise<BrokerAcknowledge> {
    return { accepted: false, reason: "broker client not configured" };
  }

  async sendRunnerResultReport(_request: RunnerResultReportRequest): Promise<BrokerAcknowledge> {
    return { accepted: false, reason: "broker client not configured" };
  }
}

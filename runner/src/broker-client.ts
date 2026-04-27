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
  DependencyCacheHandoffRequest,
  DependencyCacheHandoffResponse,
  RunnerCheckpointReportRequest,
  RunnerResultReportRequest,
} from "./contracts.ts";

export type RunnerBrokerClient = {
  requestDependencyCacheHandoff(request: DependencyCacheHandoffRequest): Promise<DependencyCacheHandoffResponse>;
  sendRunnerCheckpointReport(request: RunnerCheckpointReportRequest): Promise<BrokerAcknowledge>;
  sendRunnerResultReport(request: RunnerResultReportRequest): Promise<BrokerAcknowledge>;
};

export class NoopRunnerBrokerClient implements RunnerBrokerClient {
  async requestDependencyCacheHandoff(request: DependencyCacheHandoffRequest): Promise<DependencyCacheHandoffResponse> {
    return {
      schema_id: "runecode.protocol.v0.DependencyCacheHandoffResponse",
      schema_version: "0.1.0",
      request_id: request.request_id,
      found: false,
    };
  }

  async sendRunnerCheckpointReport(_request: RunnerCheckpointReportRequest): Promise<BrokerAcknowledge> {
    return { accepted: false, reason: "broker client not configured" };
  }

  async sendRunnerResultReport(_request: RunnerResultReportRequest): Promise<BrokerAcknowledge> {
    return { accepted: false, reason: "broker client not configured" };
  }
}

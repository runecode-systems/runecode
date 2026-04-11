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

export type RunnerBrokerClient = {
  sendRunnerCheckpointReport(request: unknown): Promise<BrokerAcknowledge>;
  sendRunnerResultReport(request: unknown): Promise<BrokerAcknowledge>;
};

export class NoopRunnerBrokerClient implements RunnerBrokerClient {
  async sendRunnerCheckpointReport(_request: unknown): Promise<BrokerAcknowledge> {
    return { accepted: false, reason: "broker client not configured" };
  }

  async sendRunnerResultReport(_request: unknown): Promise<BrokerAcknowledge> {
    return { accepted: false, reason: "broker client not configured" };
  }
}

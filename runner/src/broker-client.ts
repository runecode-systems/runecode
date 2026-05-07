/**
 * Broker client seam for runner report delivery.
 *
 * The runner stays transport-agnostic at the kernel boundary, while supported
 * product execution uses typed transport messages validated against protocol
 * schemas before crossing the trust boundary.
 */

import { createInterface } from "node:readline/promises";
import type { Interface as ReadLineInterface } from "node:readline";
import type { Readable, Writable } from "node:stream";
import { ProtocolSchemaBundle } from "./protocol-schema-bundle.ts";
import {
  DEPENDENCY_CACHE_HANDOFF_REQUEST_SCHEMA_ID,
  DEPENDENCY_CACHE_HANDOFF_RESPONSE_SCHEMA_ID,
  RUNNER_CHECKPOINT_REPORT_REQUEST_SCHEMA_ID,
  RUNNER_CHECKPOINT_REPORT_RESPONSE_SCHEMA_ID,
  RUNNER_CONTRACT_SCHEMA_VERSION,
  RUNNER_RESULT_REPORT_REQUEST_SCHEMA_ID,
  RUNNER_RESULT_REPORT_RESPONSE_SCHEMA_ID,
  type DependencyCacheHandoffRequest,
  type DependencyCacheHandoffResponse,
  type RunnerCheckpointReportRequest,
  type RunnerCheckpointReportResponse,
  type RunnerResultReportRequest,
  type RunnerResultReportResponse,
} from "./contracts.ts";

export type BrokerAcknowledge = {
  accepted: boolean;
  reason?: string;
};

export type RunnerBrokerClient = {
  requestDependencyCacheHandoff(request: DependencyCacheHandoffRequest): Promise<DependencyCacheHandoffResponse>;
  sendRunnerCheckpointReport(request: RunnerCheckpointReportRequest): Promise<BrokerAcknowledge>;
  sendRunnerResultReport(request: RunnerResultReportRequest): Promise<BrokerAcknowledge>;
};

type StdioBrokerTransportMessage = {
  message_type: "dependency_cache_handoff_request" | "runner_checkpoint_report_request" | "runner_result_report_request";
  payload: DependencyCacheHandoffRequest | RunnerCheckpointReportRequest | RunnerResultReportRequest;
};

type StdioBrokerTransportResponse = {
  message_type:
    | "dependency_cache_handoff_response"
    | "runner_checkpoint_report_response"
    | "runner_result_report_response";
  payload: DependencyCacheHandoffResponse | RunnerCheckpointReportResponse | RunnerResultReportResponse;
};

type StdioRunnerBrokerClientOptions = {
  schemaBundle: ProtocolSchemaBundle;
  input?: Readable;
  output?: Writable;
};

export class RunnerBrokerTransportError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "RunnerBrokerTransportError";
  }
}

export class MissingRunnerBrokerTransportError extends Error {
  constructor() {
    super("runner broker transport is required for supported execution path");
    this.name = "MissingRunnerBrokerTransportError";
  }
}

const brokerLifecycleStates = new Set(["pending", "starting", "active", "blocked", "recovering", "completed", "failed", "cancelled"]);

export class StdioRunnerBrokerClient implements RunnerBrokerClient {
  private readonly schemaBundle: ProtocolSchemaBundle;

  private readonly input: Readable;

  private readonly output: Writable;

  private readonly lines: ReadLineInterface;

  private readonly lineIterator: AsyncIterator<string>;

  private responseChain: Promise<void> = Promise.resolve();

  private poisoned: Error | undefined;

  constructor(options: StdioRunnerBrokerClientOptions) {
    this.schemaBundle = options.schemaBundle;
    this.input = options.input ?? process.stdin;
    this.output = options.output ?? process.stdout;
    this.lines = createInterface({ input: this.input });
    this.lineIterator = this.lines[Symbol.asyncIterator]();
  }

  async requestDependencyCacheHandoff(request: DependencyCacheHandoffRequest): Promise<DependencyCacheHandoffResponse> {
    return this.roundTrip(
      {
        message_type: "dependency_cache_handoff_request",
        payload: request,
      },
      {
        expectedMessageType: "dependency_cache_handoff_response",
        responseSchemaId: DEPENDENCY_CACHE_HANDOFF_RESPONSE_SCHEMA_ID,
        responseSchemaVersion: RUNNER_CONTRACT_SCHEMA_VERSION,
      },
    );
  }

  async sendRunnerCheckpointReport(request: RunnerCheckpointReportRequest): Promise<BrokerAcknowledge> {
    const response = await this.roundTrip<RunnerCheckpointReportResponse>(
      {
        message_type: "runner_checkpoint_report_request",
        payload: request,
      },
      {
        expectedMessageType: "runner_checkpoint_report_response",
        responseSchemaId: RUNNER_CHECKPOINT_REPORT_RESPONSE_SCHEMA_ID,
        responseSchemaVersion: RUNNER_CONTRACT_SCHEMA_VERSION,
      },
    );
    return responseToAcknowledge(response);
  }

  async sendRunnerResultReport(request: RunnerResultReportRequest): Promise<BrokerAcknowledge> {
    const response = await this.roundTrip<RunnerResultReportResponse>(
      {
        message_type: "runner_result_report_request",
        payload: request,
      },
      {
        expectedMessageType: "runner_result_report_response",
        responseSchemaId: RUNNER_RESULT_REPORT_RESPONSE_SCHEMA_ID,
        responseSchemaVersion: RUNNER_CONTRACT_SCHEMA_VERSION,
      },
    );
    return responseToAcknowledge(response);
  }

  private async roundTrip<T extends DependencyCacheHandoffResponse | RunnerCheckpointReportResponse | RunnerResultReportResponse>(
    request: StdioBrokerTransportMessage,
    expectation: {
      expectedMessageType: StdioBrokerTransportResponse["message_type"];
      responseSchemaId: string;
      responseSchemaVersion: string;
    },
  ): Promise<T> {
    const next = this.responseChain.then(async () => {
      if (this.poisoned) {
        throw this.poisoned;
      }
      this.validateOutgoingRequest(request);
      await this.writeMessage(request);
      const response = await this.readResponse();
      if (response.message_type !== expectation.expectedMessageType) {
        throw new RunnerBrokerTransportError(
          `broker transport returned ${response.message_type}; expected ${expectation.expectedMessageType}`,
        );
      }
      const validation = this.schemaBundle.validateByRuntimeKey(
        expectation.responseSchemaId,
        expectation.responseSchemaVersion,
        response.payload,
      );
      if (!validation.ok) {
        throw new RunnerBrokerTransportError(
          `broker transport response schema validation failed: ${validation.reason}`,
        );
      }
      return response.payload as T;
    });
    this.responseChain = next.then(
      () => undefined,
      (error) => {
        this.poisonTransport(error instanceof Error ? error : new RunnerBrokerTransportError(String(error)));
      },
    );
    return next;
  }

  private poisonTransport(error: Error): void {
    if (!this.poisoned) {
      this.poisoned = error;
      this.lines.close();
    }
  }

  private validateOutgoingRequest(request: StdioBrokerTransportMessage): void {
    const runtimeKey = (() => {
      switch (request.message_type) {
        case "dependency_cache_handoff_request":
          return DEPENDENCY_CACHE_HANDOFF_REQUEST_SCHEMA_ID;
        case "runner_checkpoint_report_request":
          return RUNNER_CHECKPOINT_REPORT_REQUEST_SCHEMA_ID;
        case "runner_result_report_request":
          return RUNNER_RESULT_REPORT_REQUEST_SCHEMA_ID;
      }
    })();
    const validation = this.schemaBundle.validateByRuntimeKey(runtimeKey, RUNNER_CONTRACT_SCHEMA_VERSION, request.payload);
    if (!validation.ok) {
      throw new RunnerBrokerTransportError(`broker request schema validation failed: ${validation.reason}`);
    }
  }

  private async writeMessage(message: StdioBrokerTransportMessage): Promise<void> {
    const encoded = `${JSON.stringify(message)}\n`;
    await new Promise<void>((resolve, reject) => {
      this.output.write(encoded, "utf8", (error) => {
        if (error) {
          reject(new RunnerBrokerTransportError(`broker transport write failed: ${error.message}`));
          return;
        }
        resolve();
      });
    });
  }

  private async readResponse(): Promise<StdioBrokerTransportResponse> {
    const { value, done } = await this.lineIterator.next();
    if (done || value === undefined) {
      throw new RunnerBrokerTransportError("broker transport closed before returning a typed response");
    }
    let parsed: unknown;
    try {
      parsed = JSON.parse(value);
    } catch (error) {
      throw new RunnerBrokerTransportError(`broker transport response parse failed: ${(error as Error).message}`);
    }
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      throw new RunnerBrokerTransportError("broker transport response must be an object");
    }
    const record = parsed as Record<string, unknown>;
    const messageType = record.message_type;
    if (
      messageType !== "dependency_cache_handoff_response"
      && messageType !== "runner_checkpoint_report_response"
      && messageType !== "runner_result_report_response"
    ) {
      throw new RunnerBrokerTransportError("broker transport response message_type is invalid");
    }
    const payload = record.payload;
    if (!payload || typeof payload !== "object" || Array.isArray(payload)) {
      throw new RunnerBrokerTransportError("broker transport response payload must be an object");
    }
    return {
      message_type: messageType,
      payload: payload as DependencyCacheHandoffResponse | RunnerCheckpointReportResponse | RunnerResultReportResponse,
    };
  }
}

export function createSupportedRunnerBrokerClient(options: {
  transport: "stdio" | "none";
  schemaBundle: ProtocolSchemaBundle;
  input?: Readable;
  output?: Writable;
}): RunnerBrokerClient {
  if (options.transport === "stdio") {
    return new StdioRunnerBrokerClient({
      schemaBundle: options.schemaBundle,
      input: options.input,
      output: options.output,
    });
  }
  throw new MissingRunnerBrokerTransportError();
}

function responseToAcknowledge(response: RunnerCheckpointReportResponse | RunnerResultReportResponse): BrokerAcknowledge {
  if (response.accepted) {
    return { accepted: true };
  }
  const lifecycle = brokerLifecycleStates.has(response.canonical_lifecycle_state)
    ? response.canonical_lifecycle_state
    : "unknown";
  return {
    accepted: false,
    reason: `broker rejected report at lifecycle ${lifecycle}`,
  };
}

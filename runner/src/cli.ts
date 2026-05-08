/**
 * Product runner entrypoint for supported execution path.
 *
 * Usage:
 *   node --experimental-strip-types src/cli.ts --plan-file <path>
 *
 * This entrypoint intentionally requires an explicit broker transport
 * implementation for supported execution mode.
 */

import { ProtocolSchemaBundle } from "./protocol-schema-bundle.ts";
import { RunPlanLoader } from "./run-plan.ts";
import { FileDurableStateStore } from "./durable-state.ts";
import { RunnerKernel } from "./kernel.ts";
import { createSupportedRunnerBrokerClient } from "./broker-client.ts";
import { isAbsolute, relative, resolve } from "node:path";
import { ExecutorAdapterRegistry, MinimalGateExecutorAdapter } from "./executor-adapter.ts";
import {
  DEPENDENCY_CACHE_HANDOFF_REQUEST_SCHEMA_ID,
  DEPENDENCY_CACHE_HANDOFF_RESPONSE_SCHEMA_ID,
  RUNNER_CHECKPOINT_REPORT_REQUEST_SCHEMA_ID,
  RUNNER_CHECKPOINT_REPORT_RESPONSE_SCHEMA_ID,
  RUNNER_CONTRACT_SCHEMA_VERSION,
  RUNNER_RESULT_REPORT_REQUEST_SCHEMA_ID,
  RUNNER_RESULT_REPORT_RESPONSE_SCHEMA_ID,
} from "./contracts.ts";

type RunnerCLIOptions = {
  planFile: string;
  planRoot: string;
  stateRoot: string;
  protocolSchemasRoot: string;
  brokerTransport: "stdio" | "none";
};

function parseArgs(argv: string[]): RunnerCLIOptions {
  let planFile = "";
  let planRoot = process.cwd();
  let stateRoot = ".runecode/runner-state";
  let protocolSchemasRoot = defaultProtocolSchemasRoot();
  let brokerTransport: "stdio" | "none" = "none";
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--plan-file" && i + 1 < argv.length) {
      planFile = argv[i + 1] ?? "";
      i += 1;
      continue;
    }
    if (arg === "--plan-root" && i + 1 < argv.length) {
      planRoot = argv[i + 1] ?? planRoot;
      i += 1;
      continue;
    }
    if (arg === "--state-root" && i + 1 < argv.length) {
      stateRoot = argv[i + 1] ?? stateRoot;
      i += 1;
      continue;
    }
    if (arg === "--protocol-schemas-root" && i + 1 < argv.length) {
      throw new Error("--protocol-schemas-root is not supported for product runner execution");
    }
    if (arg === "--broker-transport" && i + 1 < argv.length) {
      const value = argv[i + 1];
      if (value !== "stdio" && value !== "none") {
        throw new Error("--broker-transport must be stdio or none");
      }
      brokerTransport = value;
      i += 1;
    }
  }
  if (!planFile.trim()) {
    throw new Error("--plan-file is required");
  }
  const resolvedPlanRoot = resolve(planRoot);
  return {
    planFile: resolveConfinedPath(resolvedPlanRoot, planFile, "--plan-file"),
    planRoot: resolvedPlanRoot,
    stateRoot: resolveConfinedPath(resolvedPlanRoot, stateRoot, "--state-root"),
    protocolSchemasRoot,
    brokerTransport,
  };
}

function defaultProtocolSchemasRoot(): string {
  const configured = process.env.RUNECODE_PROTOCOL_SCHEMAS_ROOT ?? "";
  if (!configured.trim()) {
    throw new Error("RUNECODE_PROTOCOL_SCHEMAS_ROOT is required for product runner execution");
  }
  if (!isAbsolute(configured)) {
    throw new Error("RUNECODE_PROTOCOL_SCHEMAS_ROOT must be absolute");
  }
  return resolve(configured);
}

function assertRequiredRunnerSchemasPresent(bundle: ProtocolSchemaBundle): void {
  for (const [schemaID, schemaVersion] of [
    [DEPENDENCY_CACHE_HANDOFF_REQUEST_SCHEMA_ID, RUNNER_CONTRACT_SCHEMA_VERSION],
    [DEPENDENCY_CACHE_HANDOFF_RESPONSE_SCHEMA_ID, RUNNER_CONTRACT_SCHEMA_VERSION],
    [RUNNER_CHECKPOINT_REPORT_REQUEST_SCHEMA_ID, RUNNER_CONTRACT_SCHEMA_VERSION],
    [RUNNER_CHECKPOINT_REPORT_RESPONSE_SCHEMA_ID, RUNNER_CONTRACT_SCHEMA_VERSION],
    [RUNNER_RESULT_REPORT_REQUEST_SCHEMA_ID, RUNNER_CONTRACT_SCHEMA_VERSION],
    [RUNNER_RESULT_REPORT_RESPONSE_SCHEMA_ID, RUNNER_CONTRACT_SCHEMA_VERSION],
  ] as const) {
    if (!bundle.hasRuntimeKey(schemaID, schemaVersion)) {
      throw new Error(`RUNECODE_PROTOCOL_SCHEMAS_ROOT is missing required runner schema ${schemaID}@${schemaVersion}`);
    }
  }
}

function resolveConfinedPath(root: string, value: string, label: string): string {
  const resolved = isAbsolute(value) ? resolve(value) : resolve(root, value);
  const rel = relative(root, resolved);
  if (rel === "" || (!rel.startsWith("..") && !isAbsolute(rel))) {
    return resolved;
  }
  throw new Error(`${label} must resolve inside --plan-root`);
}

async function main(): Promise<void> {
  const options = parseArgs(process.argv.slice(2));
  const schemas = await ProtocolSchemaBundle.fromProtocolSchemasRoot(options.protocolSchemasRoot);
  assertRequiredRunnerSchemasPresent(schemas);
  const loader = new RunPlanLoader(schemas);
  const store = new FileDurableStateStore(options.stateRoot);
  const brokerClient = createSupportedRunnerBrokerClient({
    transport: options.brokerTransport,
    schemaBundle: schemas,
  });
  const executorAdapterRegistry = new ExecutorAdapterRegistry();
  executorAdapterRegistry.register("gate", new MinimalGateExecutorAdapter());
  const kernel = new RunnerKernel({
    planLoader: loader,
    durableStateStore: store,
    brokerClient,
    executorAdapterRegistry,
  });
  const execution = await kernel.executeScheduledWorkFromPlanFile(options.planFile);
  brokerClient.close();
  process.stderr.write(
    `executed ${execution.executed.length}/${execution.work.length} scheduled entries\n`,
  );
}

main().catch((err) => {
  const message = err instanceof Error ? err.message : String(err);
  process.stderr.write(`${message}\n`);
  process.exitCode = 1;
});

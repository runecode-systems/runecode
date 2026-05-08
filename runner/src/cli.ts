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
import { existsSync, realpathSync } from "node:fs";
import { basename, dirname, isAbsolute, relative, resolve } from "node:path";
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
  let brokerTransport: "stdio" | "none" = "none";
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--plan-file") {
      planFile = readFlagValue(argv, i, "--plan-file");
      i += 1;
      continue;
    }
    if (arg === "--plan-root") {
      planRoot = readFlagValue(argv, i, "--plan-root");
      i += 1;
      continue;
    }
    if (arg === "--state-root") {
      stateRoot = readFlagValue(argv, i, "--state-root");
      i += 1;
      continue;
    }
    if (arg === "--protocol-schemas-root") {
      throw new Error("--protocol-schemas-root is not supported for product runner execution");
    }
    if (arg === "--broker-transport") {
      const value = readFlagValue(argv, i, "--broker-transport");
      if (value !== "stdio" && value !== "none") {
        throw new Error("--broker-transport must be stdio or none");
      }
      brokerTransport = value;
      i += 1;
      continue;
    }
    throw new Error(`unknown argument: ${arg}`);
  }
  if (!planFile.trim()) {
    throw new Error("--plan-file is required");
  }
  const resolvedPlanRoot = resolve(planRoot);
  const confinedPlanRoot = canonicalizeConfinedRoot(resolvedPlanRoot, "--plan-root");
  return {
    planFile: resolveConfinedPath(confinedPlanRoot, planFile, "--plan-file"),
    planRoot: confinedPlanRoot,
    stateRoot: resolveConfinedPath(confinedPlanRoot, stateRoot, "--state-root"),
    protocolSchemasRoot: defaultProtocolSchemasRoot(),
    brokerTransport,
  };
}

function readFlagValue(argv: string[], index: number, flag: string): string {
  const value = argv[index + 1] ?? "";
  if (!value || value.startsWith("--")) {
    throw new Error(`${flag} requires a value`);
  }
  return value;
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
  const canonicalResolved = canonicalizeExistingPathPrefix(resolved);
  const rel = relative(root, canonicalResolved);
  if (rel === "" || (!rel.startsWith("..") && !isAbsolute(rel))) {
    return resolved;
  }
  throw new Error(`${label} must resolve inside --plan-root`);
}

function canonicalizeConfinedRoot(root: string, label: string): string {
  try {
    return realpathSync(root);
  } catch {
    throw new Error(`${label} must exist`);
  }
}

function canonicalizeExistingPathPrefix(pathValue: string): string {
  let current = pathValue;
  const suffix: string[] = [];
  for (;;) {
    if (existsSync(current)) {
      let canonical = realpathSync(current);
      while (suffix.length > 0) {
        canonical = resolve(canonical, suffix.pop() ?? "");
      }
      return canonical;
    }
    const parent = dirname(current);
    if (parent === current) {
      throw new Error(`path does not exist: ${pathValue}`);
    }
    suffix.push(basename(current));
    current = parent;
  }
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

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
import type { RunnerBrokerClient } from "./broker-client.ts";
import { existsSync } from "node:fs";
import { resolve } from "node:path";

type RunnerCLIOptions = {
  planFile: string;
  stateRoot: string;
  protocolSchemasRoot: string;
};

function parseArgs(argv: string[]): RunnerCLIOptions {
  let planFile = "";
  let stateRoot = ".runecode/runner-state";
  let protocolSchemasRoot = defaultProtocolSchemasRoot();
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--plan-file" && i + 1 < argv.length) {
      planFile = argv[i + 1] ?? "";
      i += 1;
      continue;
    }
    if (arg === "--state-root" && i + 1 < argv.length) {
      stateRoot = argv[i + 1] ?? stateRoot;
      i += 1;
      continue;
    }
    if (arg === "--protocol-schemas-root" && i + 1 < argv.length) {
      protocolSchemasRoot = argv[i + 1] ?? protocolSchemasRoot;
      i += 1;
    }
  }
  if (!planFile.trim()) {
    throw new Error("--plan-file is required");
  }
  return { planFile, stateRoot, protocolSchemasRoot };
}

function defaultProtocolSchemasRoot(): string {
  const repoRootCandidate = resolve(process.cwd(), "protocol/schemas");
  if (existsSync(repoRootCandidate)) {
    return repoRootCandidate;
  }
  return resolve(process.cwd(), "../protocol/schemas");
}

async function main(): Promise<void> {
  const options = parseArgs(process.argv.slice(2));
  const schemas = await ProtocolSchemaBundle.fromProtocolSchemasRoot(options.protocolSchemasRoot);
  const loader = new RunPlanLoader(schemas);
  const store = new FileDurableStateStore(options.stateRoot);
  const brokerClient: RunnerBrokerClient = {
    async requestDependencyCacheHandoff() {
      throw new Error("broker transport not wired");
    },
    async sendRunnerCheckpointReport() {
      throw new Error("broker transport not wired");
    },
    async sendRunnerResultReport() {
      throw new Error("broker transport not wired");
    },
  };
  const kernel = new RunnerKernel({
    planLoader: loader,
    durableStateStore: store,
    brokerClient,
  });
  await kernel.initializeFromPlanFile(options.planFile);
}

main().catch((err) => {
  const message = err instanceof Error ? err.message : String(err);
  process.stderr.write(`${message}\n`);
  process.exitCode = 1;
});

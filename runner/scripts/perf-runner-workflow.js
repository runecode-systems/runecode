#!/usr/bin/env node

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { performance } = require("node:perf_hooks");

const repoRoot = path.resolve(__dirname, "..", "..");

async function loadRunner() {
  return import("../src/index.ts");
}

function parseArgs(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i += 1) {
    const token = argv[i];
    if (!token.startsWith("--")) {
      continue;
    }
    const key = token.slice(2);
    const value = argv[i + 1];
    if (value === undefined || value.startsWith("--")) {
      out[key] = "";
      continue;
    }
    out[key] = value;
    i += 1;
  }
  return out;
}

async function loadPlan(runplanPath) {
  const { ProtocolSchemaBundle, RunPlanLoader } = await loadRunner();
  const schemaBundle = await ProtocolSchemaBundle.fromProtocolSchemasRoot(path.join(repoRoot, "protocol", "schemas"));
  const loader = new RunPlanLoader(schemaBundle);
  const resolvedRunplanPath = path.resolve(runplanPath);
  const tmpRoot = path.resolve(os.tmpdir());
  if (!resolvedRunplanPath.startsWith(`${tmpRoot}${path.sep}`) && !resolvedRunplanPath.startsWith(`${repoRoot}${path.sep}`)) {
    throw new Error("--runplan must resolve under the repository root or system temp directory");
  }
  const raw = fs.readFileSync(resolvedRunplanPath, "utf8");
  const parsed = JSON.parse(raw);
  return loader.loadFromUnknown(parsed);
}

async function runMode(mode, runplanPath) {
  const { PlanScheduler } = await loadRunner();
  const plan = await loadPlan(runplanPath);
  const scheduler = new PlanScheduler();
  const start = performance.now();

  switch (mode) {
    case "cold-start": {
      const work = scheduler.listPlannedWork(plan);
      if (!Array.isArray(work) || work.length === 0) {
        throw new Error("cold-start failed: no planned work");
      }
      break;
    }
    case "workflow-path": {
      const first = scheduler.listPlannedWork(plan, { pending_approval_waits: [] });
      const completed = first.map((w) => w.entry.entry_id);
      const second = scheduler.listPlannedWork(plan, { pending_approval_waits: [], completed_entry_ids: completed });
      if (!Array.isArray(second)) {
        throw new Error("workflow-path failed: invalid scheduler result");
      }
      break;
    }
    case "first-party-beta": {
      const workflowID = String(plan.workflow_id || "").trim();
      if (!workflowID.includes("first-party") && !workflowID.includes("minimal")) {
        // deterministic supported CHG-049 beta slice only
      }
      const work = scheduler.listPlannedWork(plan, { pending_approval_waits: [] });
      if (work.length < 1) {
        throw new Error("first-party-beta failed: no schedulable entry");
      }
      break;
    }
    case "immutable-startup": {
      const serialized = JSON.stringify(plan);
      const roundtrip = JSON.parse(serialized);
      if (roundtrip.plan_id !== plan.plan_id) {
        throw new Error("immutable-startup failed: plan roundtrip mismatch");
      }
      const work = scheduler.listPlannedWork(roundtrip, { pending_approval_waits: [] });
      if (work.length === 0) {
        throw new Error("immutable-startup failed: no planned work");
      }
      break;
    }
    default:
      throw new Error(`unsupported --mode ${mode}`);
  }

  return Math.max(0, Math.round(performance.now() - start));
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const mode = String(args.mode || "").trim();
  const runplanPath = String(args.runplan || "").trim();
  if (!mode || !runplanPath) {
    throw new Error("--mode and --runplan are required");
  }
  const wallMs = await runMode(mode, runplanPath);
  process.stdout.write(`${wallMs}\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.message}\n`);
  process.exit(1);
});

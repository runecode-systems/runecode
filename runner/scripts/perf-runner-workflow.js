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
  const resolvedRunplanPath = fs.realpathSync(path.resolve(runplanPath));
  const tmpRoot = path.resolve(os.tmpdir());
  if (!resolvedRunplanPath.startsWith(`${tmpRoot}${path.sep}`) && !resolvedRunplanPath.startsWith(`${repoRoot}${path.sep}`)) {
    throw new Error("--runplan must resolve under the repository root or system temp directory");
  }
  const raw = fs.readFileSync(resolvedRunplanPath, "utf8");
  const parsed = JSON.parse(raw);
  return loader.loadFromUnknown(parsed);
}

async function runMode(mode, runplanPath, fixtureID) {
  const { PlanScheduler } = await loadRunner();
  const plan = await loadPlan(runplanPath);
  const scheduler = new PlanScheduler();
  const normalizedFixtureID = String(fixtureID || "").trim();

  const expectFirstPartyMinimalFixture = () => {
    if (normalizedFixtureID !== "workflow.first-party-minimal.v1") {
      throw new Error(`mode ${mode} requires --fixture workflow.first-party-minimal.v1`);
    }
    if (String(plan.workflow_id || "").trim() !== "workflow_first_party_minimal") {
      throw new Error(`mode ${mode} requires workflow_id workflow_first_party_minimal`);
    }
    if (String(plan.process_id || "").trim() !== "process_first_party_minimal") {
      throw new Error(`mode ${mode} requires process_id process_first_party_minimal`);
    }
  };

  switch (mode) {
    case "cold-start": {
      const start = performance.now();
      const work = scheduler.listPlannedWork(plan);
      if (!Array.isArray(work) || work.length === 0) {
        throw new Error("cold-start failed: no planned work");
      }
      return Math.max(0, Math.round(performance.now() - start));
    }
    case "workflow-path": {
      expectFirstPartyMinimalFixture();
      const start = performance.now();
      const blocked = scheduler.listPlannedWork(plan, {
        pending_approval_waits: [{ blocked_scope: { scope_kind: "run", run_id: plan.run_id } }],
      });
      if (!Array.isArray(blocked) || blocked.length !== 0) {
        throw new Error("workflow-path failed: expected wait-scoped blocking");
      }
      const first = scheduler.listPlannedWork(plan, { pending_approval_waits: [], completed_entry_ids: [] });
      if (!Array.isArray(first) || first.length === 0) {
        throw new Error("workflow-path failed: no schedulable work on supported path");
      }
      const completed = first.map((w) => w.entry.entry_id);
      const second = scheduler.listPlannedWork(plan, { pending_approval_waits: [], completed_entry_ids: completed });
      if (!Array.isArray(second) || second.length !== 0) {
        throw new Error("workflow-path failed: invalid scheduler result");
      }
      return Math.max(0, Math.round(performance.now() - start));
    }
    case "first-party-beta": {
      expectFirstPartyMinimalFixture();
      const start = performance.now();
      const work = scheduler.listPlannedWork(plan, { pending_approval_waits: [] });
      if (work.length < 1) {
        throw new Error("first-party-beta failed: no schedulable entry");
      }
      if (work[0]?.entry?.entry_id !== "quality_lint" || work[0]?.entry?.entry_kind !== "gate") {
        throw new Error("first-party-beta failed: fixture does not match supported first-party beta slice");
      }
      return Math.max(0, Math.round(performance.now() - start));
    }
    case "immutable-startup": {
      const start = performance.now();
      const serialized = JSON.stringify(plan);
      const roundtrip = JSON.parse(serialized);
      if (roundtrip.plan_id !== plan.plan_id) {
        throw new Error("immutable-startup failed: plan roundtrip mismatch");
      }
      const work = scheduler.listPlannedWork(roundtrip, { pending_approval_waits: [] });
      if (work.length === 0) {
        throw new Error("immutable-startup failed: no planned work");
      }
      return Math.max(0, Math.round(performance.now() - start));
    }
    default:
      throw new Error(`unsupported --mode ${mode}`);
  }
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const mode = String(args.mode || "").trim();
  const runplanPath = String(args.runplan || "").trim();
  const fixtureID = String(args.fixture || "").trim();
  if (!mode || !runplanPath) {
    throw new Error("--mode and --runplan are required");
  }
  const wallMs = await runMode(mode, runplanPath, fixtureID);
  process.stdout.write(`${wallMs}\n`);
}

main().catch((error) => {
  process.stderr.write(`${error.message}\n`);
  process.exit(1);
});

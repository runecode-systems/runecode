const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");
const { spawnSync } = require("node:child_process");
const { createHash } = require("node:crypto");

const { repoRoot, validRunPlanFixture } = require("./runner-test-helpers.js");

const cliPath = path.join(repoRoot, "runner", "src", "cli.ts");

function writePlan(root) {
  const planPath = path.join(root, "runplan.json");
  fs.writeFileSync(planPath, JSON.stringify(validRunPlanFixture(), null, 2));
  return planPath;
}

function dependencyHandoffRequestID(runID, requestDigest) {
  return `dependency-handoff:${createHash("sha256").update(runID).update("\n").update(requestDigest).digest("hex")}`;
}

function runCLI(args, options = {}) {
  return spawnSync(
    process.execPath,
    ["--experimental-strip-types", cliPath, ...args],
    {
      cwd: options.cwd ?? path.join(repoRoot, "runner"),
      encoding: "utf8",
      input: options.input,
      env: {
        ...process.env,
        RUNECODE_PROTOCOL_SCHEMAS_ROOT: path.join(repoRoot, "protocol", "schemas"),
        ...(options.env ?? {}),
      },
    },
  );
}

test("cli fails closed when broker transport is missing", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-"));
  try {
    const planPath = writePlan(root);
    const result = runCLI(["--plan-file", planPath, "--plan-root", root]);
    assert.equal(result.status, 1);
    assert.match(result.stderr, /runner broker transport is required/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("cli rejects plan files outside the declared plan root", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-"));
  const otherRoot = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-other-"));
  try {
    const planPath = writePlan(otherRoot);
    const result = runCLI(["--plan-file", planPath, "--plan-root", root, "--broker-transport", "stdio"]);
    assert.equal(result.status, 1);
    assert.match(result.stderr, /--plan-file must resolve inside --plan-root/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
    fs.rmSync(otherRoot, { recursive: true, force: true });
  }
});

test("cli rejects plan files that escape plan root through symlinks", (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-"));
  const otherRoot = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-other-"));
  try {
    const escapedPlanPath = writePlan(otherRoot);
    const linkedDir = path.join(root, "linked");
    try {
      fs.symlinkSync(otherRoot, linkedDir, "dir");
    } catch (error) {
      const code = error && typeof error === "object" && "code" in error ? error.code : "";
      if (["EPERM", "EACCES", "ENOTSUP"].includes(code)) {
        t.skip(`symlink creation unavailable: ${code}`);
      }
      throw error;
    }
    const symlinkedPlanPath = path.join(linkedDir, path.basename(escapedPlanPath));
    const result = runCLI(["--plan-file", symlinkedPlanPath, "--plan-root", root, "--broker-transport", "stdio"]);
    assert.equal(result.status, 1);
    assert.match(result.stderr, /--plan-file must resolve inside --plan-root/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
    fs.rmSync(otherRoot, { recursive: true, force: true });
  }
});

test("cli rejects caller-supplied protocol schema roots", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-"));
  try {
    const planPath = writePlan(root);
    const result = runCLI(["--plan-file", planPath, "--plan-root", root, "--protocol-schemas-root", root]);
    assert.equal(result.status, 1);
    assert.match(result.stderr, /--protocol-schemas-root is not supported/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("cli rejects unknown flags", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-"));
  try {
    const planPath = writePlan(root);
    const result = runCLI(["--plan-file", planPath, "--plan-root", root, "--unknown-flag"]);
    assert.equal(result.status, 1);
    assert.match(result.stderr, /unknown argument: --unknown-flag/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("cli rejects missing required flag values", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-"));
  try {
    const result = runCLI(["--plan-file", "--plan-root", root]);
    assert.equal(result.status, 1);
    assert.match(result.stderr, /--plan-file requires a value/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("cli fails closed when protocol schema root env is missing", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-"));
  try {
    const planPath = writePlan(root);
    const result = runCLI(["--plan-file", planPath, "--plan-root", root], { env: { RUNECODE_PROTOCOL_SCHEMAS_ROOT: "" } });
    assert.equal(result.status, 1);
    assert.match(result.stderr, /RUNECODE_PROTOCOL_SCHEMAS_ROOT is required/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("cli fails closed when protocol schema root env lacks required runner schemas", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-"));
  const fakeSchemas = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-schemas-"));
  try {
    const planPath = writePlan(root);
    fs.writeFileSync(path.join(fakeSchemas, "manifest.json"), JSON.stringify({ schema_files: [] }, null, 2));
    const result = runCLI(["--plan-file", planPath, "--plan-root", root], { env: { RUNECODE_PROTOCOL_SCHEMAS_ROOT: fakeSchemas } });
    assert.equal(result.status, 1);
    assert.match(result.stderr, /missing required runner schema/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
    fs.rmSync(fakeSchemas, { recursive: true, force: true });
  }
});

test("cli fails closed when manifest omits one required runner schema entry", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-"));
  const fakeSchemas = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-schemas-"));
  try {
    const planPath = writePlan(root);
    const manifest = JSON.parse(fs.readFileSync(path.join(repoRoot, "protocol", "schemas", "manifest.json"), "utf8"));
    manifest.schema_files = manifest.schema_files.filter(
      (entry) => !(entry.schema_id === "runecode.protocol.v0.RunnerResultReportResponse" && entry.schema_version === "0.1.0"),
    );
    fs.cpSync(path.join(repoRoot, "protocol", "schemas"), fakeSchemas, { recursive: true });
    fs.writeFileSync(path.join(fakeSchemas, "manifest.json"), JSON.stringify(manifest, null, 2));

    const result = runCLI(["--plan-file", planPath, "--plan-root", root], { env: { RUNECODE_PROTOCOL_SCHEMAS_ROOT: fakeSchemas } });
    assert.equal(result.status, 1);
    assert.match(result.stderr, /missing required runner schema .*RunnerResultReportResponse@0\.1\.0/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
    fs.rmSync(fakeSchemas, { recursive: true, force: true });
  }
});

test("cli fails closed when manifest runtime key points at malformed relaxed schema content", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-"));
  const fakeSchemas = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-schemas-"));
  try {
    const planPath = writePlan(root);
    fs.cpSync(path.join(repoRoot, "protocol", "schemas"), fakeSchemas, { recursive: true });
    const schemaPath = path.join(fakeSchemas, "objects", "DependencyCacheHandoffRequest.schema.json");
    const schema = JSON.parse(fs.readFileSync(schemaPath, "utf8"));
    schema.properties.schema_id.const = "runecode.protocol.v0.NotDependencyCacheHandoffRequest";
    delete schema.required;
    schema.additionalProperties = true;
    fs.writeFileSync(schemaPath, JSON.stringify(schema, null, 2));

    const result = runCLI(["--plan-file", planPath, "--plan-root", root], { env: { RUNECODE_PROTOCOL_SCHEMAS_ROOT: fakeSchemas } });
    assert.equal(result.status, 1);
    assert.match(result.stderr, /schema manifest entry .*DependencyCacheHandoffRequest\.schema\.json schema_id const .* does not match runecode\.protocol\.v0\.DependencyCacheHandoffRequest/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
    fs.rmSync(fakeSchemas, { recursive: true, force: true });
  }
});

test("cli works from non-runner cwd when schema root comes from env", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-"));
  try {
    const planPath = writePlan(root);
    const responses = [
      {
        message_type: "dependency_cache_handoff_response",
        payload: {
          schema_id: "runecode.protocol.v0.DependencyCacheHandoffResponse",
          schema_version: "0.1.0",
          request_id: dependencyHandoffRequestID("run_alpha", "sha256:" + "d".repeat(64)),
          found: true,
          handoff: {
            schema_id: "runecode.protocol.v0.DependencyCacheHandoffMetadata",
            schema_version: "0.1.0",
            request_digest: { hash_alg: "sha256", hash: "d".repeat(64) },
            resolved_unit_digest: { hash_alg: "sha256", hash: "e".repeat(64) },
            manifest_digest: { hash_alg: "sha256", hash: "f".repeat(64) },
            payload_digests: [{ hash_alg: "sha256", hash: "1".repeat(64) }],
            materialization_mode: "derived_read_only",
            handoff_mode: "broker_internal_artifact_handoff",
          },
        },
      },
      {
        message_type: "runner_checkpoint_report_response",
        payload: {
          schema_id: "runecode.protocol.v0.RunnerCheckpointReportResponse",
          schema_version: "0.1.0",
          request_id: "runner-checkpoint:run_alpha:quality_lint:0",
          run_id: "run_alpha",
          accepted: true,
          canonical_lifecycle_state: "active",
          accepted_at: "2026-01-01T00:00:00Z",
          idempotency_key: "runner-checkpoint:run_alpha:quality_lint:active",
        },
      },
      {
        message_type: "runner_result_report_response",
        payload: {
          schema_id: "runecode.protocol.v0.RunnerResultReportResponse",
          schema_version: "0.1.0",
          request_id: "runner-result:run_alpha:quality_lint:0",
          run_id: "run_alpha",
          accepted: true,
          canonical_lifecycle_state: "completed",
          accepted_at: "2026-01-01T00:00:00Z",
          idempotency_key: "runner-result:run_alpha:quality_lint:ok",
        },
      },
    ].map((entry) => JSON.stringify(entry)).join("\n") + "\n";

    const result = runCLI([
      "--plan-file", planPath,
      "--plan-root", root,
      "--state-root", path.join(root, "state"),
      "--broker-transport", "stdio",
    ], { input: responses, cwd: repoRoot });

    assert.equal(result.status, 0, result.stderr);
    assert.match(result.stderr, /executed 1\/1 scheduled entries/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("cli executes plan-first path over stdio transport", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-"));
  try {
    const planPath = writePlan(root);
    const responses = [
      {
        message_type: "dependency_cache_handoff_response",
        payload: {
          schema_id: "runecode.protocol.v0.DependencyCacheHandoffResponse",
          schema_version: "0.1.0",
          request_id: dependencyHandoffRequestID("run_alpha", "sha256:" + "d".repeat(64)),
          found: true,
          handoff: {
            schema_id: "runecode.protocol.v0.DependencyCacheHandoffMetadata",
            schema_version: "0.1.0",
            request_digest: { hash_alg: "sha256", hash: "d".repeat(64) },
            resolved_unit_digest: { hash_alg: "sha256", hash: "e".repeat(64) },
            manifest_digest: { hash_alg: "sha256", hash: "f".repeat(64) },
            payload_digests: [{ hash_alg: "sha256", hash: "1".repeat(64) }],
            materialization_mode: "derived_read_only",
            handoff_mode: "broker_internal_artifact_handoff",
          },
        },
      },
      {
        message_type: "runner_checkpoint_report_response",
        payload: {
          schema_id: "runecode.protocol.v0.RunnerCheckpointReportResponse",
          schema_version: "0.1.0",
          request_id: "runner-checkpoint:run_alpha:quality_lint:0",
          run_id: "run_alpha",
          accepted: true,
          canonical_lifecycle_state: "active",
          accepted_at: "2026-01-01T00:00:00Z",
          idempotency_key: "runner-checkpoint:run_alpha:quality_lint:active",
        },
      },
      {
        message_type: "runner_result_report_response",
        payload: {
          schema_id: "runecode.protocol.v0.RunnerResultReportResponse",
          schema_version: "0.1.0",
          request_id: "runner-result:run_alpha:quality_lint:0",
          run_id: "run_alpha",
          accepted: true,
          canonical_lifecycle_state: "completed",
          accepted_at: "2026-01-01T00:00:00Z",
          idempotency_key: "runner-result:run_alpha:quality_lint:ok",
        },
      },
    ].map((entry) => JSON.stringify(entry)).join("\n") + "\n";

    const result = runCLI([
      "--plan-file", planPath,
      "--plan-root", root,
      "--state-root", path.join(root, "state"),
      "--broker-transport", "stdio",
    ], { input: responses });

    assert.equal(result.status, 0, result.stderr);
    assert.match(result.stderr, /executed 1\/1 scheduled entries/);

    const lines = result.stdout.trim().split("\n").filter(Boolean).map((line) => JSON.parse(line));
    assert.equal(lines.length, 3);
    assert.equal(lines[0].message_type, "dependency_cache_handoff_request");
    assert.equal(lines[1].message_type, "runner_checkpoint_report_request");
    assert.equal(lines[2].message_type, "runner_result_report_request");
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("cli exits nonzero when typed broker response rejects a report", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-runner-cli-"));
  try {
    const planPath = writePlan(root);
    const responses = [
      {
        message_type: "dependency_cache_handoff_response",
        payload: {
          schema_id: "runecode.protocol.v0.DependencyCacheHandoffResponse",
          schema_version: "0.1.0",
          request_id: dependencyHandoffRequestID("run_alpha", "sha256:" + "d".repeat(64)),
          found: true,
          handoff: {
            schema_id: "runecode.protocol.v0.DependencyCacheHandoffMetadata",
            schema_version: "0.1.0",
            request_digest: { hash_alg: "sha256", hash: "d".repeat(64) },
            resolved_unit_digest: { hash_alg: "sha256", hash: "e".repeat(64) },
            manifest_digest: { hash_alg: "sha256", hash: "f".repeat(64) },
            payload_digests: [{ hash_alg: "sha256", hash: "1".repeat(64) }],
            materialization_mode: "derived_read_only",
            handoff_mode: "broker_internal_artifact_handoff",
          },
        },
      },
      {
        message_type: "runner_checkpoint_report_response",
        payload: {
          schema_id: "runecode.protocol.v0.RunnerCheckpointReportResponse",
          schema_version: "0.1.0",
          request_id: "runner-checkpoint:run_alpha:quality_lint:0",
          run_id: "run_alpha",
          accepted: false,
          canonical_lifecycle_state: "active",
          accepted_at: "2026-01-01T00:00:00Z",
          idempotency_key: "runner-checkpoint:run_alpha:quality_lint:active",
        },
      },
    ].map((entry) => JSON.stringify(entry)).join("\n") + "\n";

    const result = runCLI([
      "--plan-file", planPath,
      "--plan-root", root,
      "--state-root", path.join(root, "state"),
      "--broker-transport", "stdio",
    ], { input: responses });

    assert.equal(result.status, 1);
    assert.match(result.stderr, /broker rejected report at lifecycle active/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

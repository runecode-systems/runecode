const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const { createConfig, runBoundaryCheck } = require("./boundary-check");

function createTempRepo(files) {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), "runecode-boundary-"));

  for (const [relativePath, contents] of Object.entries(files)) {
    const absolutePath = path.join(repoRoot, relativePath);
    fs.mkdirSync(path.dirname(absolutePath), { recursive: true });
    fs.writeFileSync(absolutePath, contents);
  }

  return repoRoot;
}

function runCheckForRepo(repoRoot) {
  const runnerRoot = path.join(repoRoot, "runner");
  return runBoundaryCheck(createConfig({ runnerRoot, repoRoot }));
}

test("allows runner-local and protocol schema/fixture references", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "import './local';\nconst fixture = '../../protocol/fixtures/example.json';\n",
    "runner/src/local.ts": "export const ok = true;\n",
    "runner/scripts/job.js": "const schema = '../../protocol/schemas/example.json';\n",
    "protocol/fixtures/example.json": "{}\n",
    "protocol/schemas/example.json": "{}\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, true);
  assert.equal(result.error, null);
  assert.ok(result.files.length >= 3);
});

test("rejects trusted path escapes from files outside src", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "export const ok = true;\n",
    "runner/scripts/job.js": "const leaked = '../../internal/secret.txt';\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, false);
  assert.equal(result.error, null);
  assert.ok(result.violations.some((item) => item.includes("scripts/job.js")));
});

test("fails closed when no source files are present", (t) => {
  const repoRoot = createTempRepo({
    "runner/package.json": "{}\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, false);
  assert.match(result.error, /no runner source files found/);
});

test("ignores node_modules and dist directories", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "export const ok = true;\n",
    "runner/node_modules/pkg/index.js": "const leaked = '../../internal/secret.txt';\n",
    "runner/dist/bundle.js": "const leaked = '../../internal/secret.txt';\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, true);
  assert.equal(result.error, null);
  assert.equal(result.violations.length, 0);
});

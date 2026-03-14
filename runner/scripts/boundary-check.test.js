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

test("allows repo-root protocol schema/fixture specifiers", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "const schema = 'protocol/schemas/example.json';\nconst fixture = 'protocol/fixtures/example.json';\n",
    "protocol/fixtures/example.json": "{}\n",
    "protocol/schemas/example.json": "{}\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, true);
  assert.equal(result.error, null);
  assert.equal(result.violations.length, 0);
});

test("allows protocol root literals embedded in path.join", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "const schemaRoot = path.join(repoRoot, 'protocol/schemas');\nconst fixtureRoot = path.join(repoRoot, 'protocol/fixtures');\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, true);
  assert.equal(result.error, null);
  assert.equal(result.violations.length, 0);
});

test("allows bare protocol string literals", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "const label = 'protocol';\nexport default label;\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, true);
  assert.equal(result.error, null);
  assert.equal(result.violations.length, 0);
});

test("allows bare tools string literals", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "const label = 'tools';\nexport default label;\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, true);
  assert.equal(result.error, null);
  assert.equal(result.violations.length, 0);
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

test("rejects repo-root tools path references", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "const helper = 'tools/gofmtcheck/main.go';\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, false);
  assert.equal(result.error, null);
  assert.ok(result.violations.some((item) => item.includes("src/index.ts") && item.includes("restricted repo-root path 'tools/gofmtcheck/main.go'")));
});

test("rejects relative path escapes to tools", (t) => {
  const repoRoot = createTempRepo({
    "runner/scripts/job.js": "const helper = '../../tools/gofmtcheck/main.go';\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, false);
  assert.equal(result.error, null);
  assert.ok(result.violations.some((item) => item.includes("scripts/job.js") && item.includes("trusted path '../../tools/gofmtcheck/main.go'")));
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

test("allows scoped package imports that contain internal segments", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "import helper from '@scope/pkg/internal/foo';\nexport default helper;\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, true);
  assert.equal(result.error, null);
  assert.equal(result.violations.length, 0);
});

test("rejects Unix absolute path references", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "export const ok = true;\n",
    "runner/scripts/job.js": "const leaked = '/etc/passwd';\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, false);
  assert.equal(result.error, null);
  assert.ok(result.violations.some((item) => item.includes("scripts/job.js")));
});

test("rejects absolute path references even when inside runner", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "export const ok = true;\n",
    "runner/scripts/job.js": "",
  });

  const runnerLocalFile = path.join(repoRoot, "runner", "src", "index.ts");
  fs.writeFileSync(
    path.join(repoRoot, "runner", "scripts", "job.js"),
    `const local = ${JSON.stringify(runnerLocalFile)};\n`,
  );

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, false);
  assert.equal(result.error, null);
  assert.ok(result.violations.some((item) => item.includes("scripts/job.js")));
});

test("allows absolute protocol schema references", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "export const ok = true;\n",
    "runner/scripts/job.js": "",
    "protocol/schemas/example.json": "{}\n",
  });

  const schemaAbsolutePath = path.join(repoRoot, "protocol", "schemas", "example.json");
  fs.writeFileSync(
    path.join(repoRoot, "runner", "scripts", "job.js"),
    `const schema = ${JSON.stringify(schemaAbsolutePath)};\n`,
  );

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, true);
  assert.equal(result.error, null);
  assert.equal(result.violations.length, 0);
});

test("rejects Windows drive-letter absolute path references", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "export const ok = true;\n",
    "runner/scripts/job.js": "const leaked = 'C:\\\\repo\\\\internal\\\\secret.txt';\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, false);
  assert.equal(result.error, null);
  assert.ok(result.violations.some((item) => item.includes("scripts/job.js")));
});

test("rejects Windows UNC path references", (t) => {
  const repoRoot = createTempRepo({
    "runner/src/index.ts": "export const ok = true;\n",
    "runner/scripts/job.js": "const leaked = '\\\\\\\\server\\\\share\\\\internal\\\\secret.txt';\n",
  });

  t.after(() => {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  });

  const result = runCheckForRepo(repoRoot);

  assert.equal(result.ok, false);
  assert.equal(result.error, null);
  assert.ok(result.violations.some((item) => item.includes("scripts/job.js")));
});

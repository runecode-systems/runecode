const fs = require("node:fs");
const path = require("node:path");

const SOURCE_EXTENSIONS = new Set([".ts", ".tsx", ".mts", ".cts", ".js", ".mjs", ".cjs"]);
const EXCLUDED_DIRS = new Set(["node_modules", "dist", ".git", ".turbo", "coverage"]);
const EXCLUDED_RELATIVE_FILES = new Set([
  "scripts/boundary-check.js",
  "scripts/boundary-check.test.js",
]);

function isInside(targetPath, rootPath) {
  const relative = path.relative(rootPath, targetPath);
  return relative === "" || (!relative.startsWith("..") && !path.isAbsolute(relative));
}

function normalizeSpecifier(value) {
  return value.replaceAll("\\", "/");
}

function createConfig(options = {}) {
  const runnerRoot = options.runnerRoot
    ? path.resolve(options.runnerRoot)
    : path.resolve(__dirname, "..");
  const repoRoot = options.repoRoot ? path.resolve(options.repoRoot) : path.resolve(runnerRoot, "..");

  return {
    runnerRoot,
    repoRoot,
    trustedRoots: [
      path.join(repoRoot, "cmd"),
      path.join(repoRoot, "internal"),
    ],
    allowedProtocolRoots: [
      path.join(repoRoot, "protocol", "schemas"),
      path.join(repoRoot, "protocol", "fixtures"),
    ],
    excludedDirs: new Set(options.excludedDirs || EXCLUDED_DIRS),
    excludedRelativeFiles: new Set(options.excludedRelativeFiles || EXCLUDED_RELATIVE_FILES),
  };
}

function collectSourceFiles(rootDir, config) {
  if (!fs.existsSync(rootDir)) {
    return [];
  }

  const files = [];
  const stack = [rootDir];

  while (stack.length > 0) {
    const current = stack.pop();
    for (const entry of fs.readdirSync(current, { withFileTypes: true })) {
      const fullPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        if (config.excludedDirs.has(entry.name)) {
          continue;
        }

        stack.push(fullPath);
        continue;
      }

      if (SOURCE_EXTENSIONS.has(path.extname(entry.name))) {
        const relativePath = normalizeSpecifier(path.relative(config.runnerRoot, fullPath));
        if (config.excludedRelativeFiles.has(relativePath)) {
          continue;
        }

        files.push(fullPath);
      }
    }
  }

  files.sort();
  return files;
}

function extractSpecifiers(content) {
  const specifiers = new Set();
  const patterns = [
    /\b(?:import|export)\s+(?:[^;]*?\s+from\s+)?["'`]([^"'`]+)["'`]/g,
    /\brequire\(\s*["'`]([^"'`]+)["'`]\s*\)/g,
    /\bimport\(\s*["'`]([^"'`]+)["'`]\s*\)/g,
    /["'`]((?:\.\.?(?:\/|\\)[^"'`]+)|(?:cmd|internal|protocol)(?:\/|\\)[^"'`]*)["'`]/g,
  ];

  for (const pattern of patterns) {
    let match = pattern.exec(content);
    while (match !== null) {
      specifiers.add(match[1]);
      match = pattern.exec(content);
    }
  }

  return [...specifiers];
}

function checkSpecifier(filePath, specifier, config, violations) {
  const normalized = normalizeSpecifier(specifier.trim());
  if (normalized.length === 0) {
    return;
  }

  const hasRelativePrefix = normalized.startsWith(".") || normalized.startsWith("/");
  const hasBoundaryKeyword =
    normalized.startsWith("cmd/") ||
    normalized.startsWith("internal/") ||
    normalized.startsWith("protocol/") ||
    normalized.includes("/cmd/") ||
    normalized.includes("/internal/");

  if (!hasRelativePrefix && !hasBoundaryKeyword) {
    return;
  }

  if (
    normalized === "cmd" ||
    normalized === "protocol" ||
    normalized === "internal" ||
    normalized.startsWith("cmd/") ||
    normalized.startsWith("internal/") ||
    normalized.includes("/cmd/") ||
    normalized.includes("/internal/")
  ) {
    violations.push(`${path.relative(config.runnerRoot, filePath)} references trusted path '${specifier}'`);
    return;
  }

  let resolvedPath;
  if (normalized.startsWith("protocol/")) {
    resolvedPath = path.resolve(config.repoRoot, normalized);
  } else if (hasRelativePrefix) {
    resolvedPath = path.resolve(path.dirname(filePath), specifier);
  } else {
    return;
  }

  if (config.trustedRoots.some((root) => isInside(resolvedPath, root))) {
    violations.push(`${path.relative(config.runnerRoot, filePath)} escapes into trusted path '${specifier}'`);
    return;
  }

  if (isInside(resolvedPath, config.runnerRoot)) {
    return;
  }

  if (config.allowedProtocolRoots.some((root) => isInside(resolvedPath, root))) {
    return;
  }

  violations.push(
    `${path.relative(config.runnerRoot, filePath)} escapes runner boundary with '${specifier}' (only protocol/schemas and protocol/fixtures are allowed)`,
  );
}

function runBoundaryCheck(config) {
  const files = collectSourceFiles(config.runnerRoot, config);
  if (files.length === 0) {
    return {
      ok: false,
      error: `no runner source files found under ${config.runnerRoot}`,
      files,
      violations: [],
    };
  }

  const violations = [];

  for (const filePath of files) {
    const content = fs.readFileSync(filePath, "utf8");
    const specifiers = extractSpecifiers(content);
    for (const specifier of specifiers) {
      checkSpecifier(filePath, specifier, config, violations);
    }
  }

  return {
    ok: violations.length === 0,
    error: null,
    files,
    violations,
  };
}

function main() {
  const result = runBoundaryCheck(createConfig());

  if (result.error) {
    console.error(`Boundary check failed: ${result.error}`);
    process.exit(1);
  }

  if (result.violations.length > 0) {
    console.error("Boundary check failed:");
    for (const violation of result.violations) {
      console.error(`- ${violation}`);
    }
    process.exit(1);
  }

  console.log(`Boundary check passed (${result.files.length} files scanned).`);
}

module.exports = {
  createConfig,
  collectSourceFiles,
  extractSpecifiers,
  checkSpecifier,
  runBoundaryCheck,
  normalizeSpecifier,
  isInside,
};

if (require.main === module) {
  main();
}

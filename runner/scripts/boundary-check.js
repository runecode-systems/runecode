const fs = require("node:fs");
const path = require("node:path");

const SOURCE_EXTENSIONS = new Set([".ts", ".tsx", ".mts", ".cts", ".js", ".mjs", ".cjs"]);
const EXCLUDED_DIRS = new Set(["node_modules", "dist", ".git", ".turbo", "coverage"]);
const EXCLUDED_RELATIVE_FILES = new Set([
  "scripts/boundary-check.js",
  "scripts/boundary-check.test.js",
]);
const WINDOWS_DRIVE_ABSOLUTE_RE = /^[A-Za-z]:[\\/]/;

function isInside(targetPath, rootPath) {
  const normalizedTarget = normalizeForComparison(targetPath);
  const normalizedRoot = normalizeForComparison(rootPath);

  if (normalizedRoot === "/") {
    return normalizedTarget.startsWith("/");
  }

  return normalizedTarget === normalizedRoot || normalizedTarget.startsWith(`${normalizedRoot}/`);
}

function normalizeSpecifier(value) {
  return value.replaceAll("\\", "/");
}

function normalizeForComparison(value) {
  let normalized = normalizeSpecifier(String(value));

  if (WINDOWS_DRIVE_ABSOLUTE_RE.test(normalized)) {
    normalized = `${normalized[0].toLowerCase()}${normalized.slice(1)}`;
  }

  if (normalized.startsWith("//")) {
    normalized = `//${normalized.slice(2).replace(/^\/+/, "").replace(/\/+/g, "/")}`;
  } else {
    normalized = normalized.replace(/\/+/g, "/");
  }

  if (normalized.length > 1 && normalized.endsWith("/")) {
    normalized = normalized.slice(0, -1);
  }

  return normalized;
}

function isWindowsUNCSpecifier(value) {
  return value.startsWith("\\\\");
}

function isAbsolutePathSpecifier(rawSpecifier, normalizedSpecifier) {
  return (
    normalizedSpecifier.startsWith("/")
    || WINDOWS_DRIVE_ABSOLUTE_RE.test(rawSpecifier)
    || WINDOWS_DRIVE_ABSOLUTE_RE.test(normalizedSpecifier)
    || isWindowsUNCSpecifier(rawSpecifier)
  );
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
    /["'`]((?:\.\.?(?:\/|\\)[^"'`]+)|(?:[A-Za-z]:[\\/][^"'`]+)|(?:\\\\[^"'`]+)|(?:cmd|internal|protocol)(?:\/|\\)[^"'`]*)["'`]/g,
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
  const trimmed = specifier.trim();
  const normalized = normalizeSpecifier(trimmed);
  if (normalized.length === 0) {
    return;
  }
  const relativeFile = normalizeSpecifier(path.relative(config.runnerRoot, filePath));

  const hasRelativePrefix = normalized.startsWith(".");
  const hasAbsolutePrefix = isAbsolutePathSpecifier(trimmed, normalized);
  const hasBoundaryKeyword =
    normalized.startsWith("cmd/") ||
    normalized.startsWith("internal/") ||
    normalized.startsWith("protocol/");

  if (!hasRelativePrefix && !hasAbsolutePrefix && !hasBoundaryKeyword) {
    return;
  }

  if (
    normalized === "cmd" ||
    normalized === "protocol" ||
    normalized === "internal" ||
    normalized.startsWith("cmd/") ||
    normalized.startsWith("internal/")
  ) {
    violations.push(`${relativeFile} references trusted path '${specifier}'`);
    return;
  }

  let resolvedPath;
  if (normalized.startsWith("protocol/")) {
    resolvedPath = path.resolve(config.repoRoot, normalized);
  } else if (hasRelativePrefix) {
    resolvedPath = path.resolve(path.dirname(filePath), normalized);
  } else if (hasAbsolutePrefix) {
    resolvedPath = normalizeForComparison(normalized);
  } else {
    return;
  }

  if (config.trustedRoots.some((root) => isInside(resolvedPath, root))) {
    violations.push(`${relativeFile} escapes into trusted path '${specifier}'`);
    return;
  }

  if (isInside(resolvedPath, config.runnerRoot)) {
    return;
  }

  if (config.allowedProtocolRoots.some((root) => isInside(resolvedPath, root))) {
    return;
  }

  violations.push(
    `${relativeFile} escapes runner boundary with '${specifier}' (only protocol/schemas and protocol/fixtures are allowed)`,
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
  normalizeForComparison,
  isAbsolutePathSpecifier,
  isWindowsUNCSpecifier,
  isInside,
};

if (require.main === module) {
  main();
}

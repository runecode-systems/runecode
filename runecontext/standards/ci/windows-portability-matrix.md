---
schema_version: 1
id: ci/windows-portability-matrix
title: Windows Portability Matrix
status: active
suggested_context_bundles:
    - ci-tooling
    - runner-boundary
---

# Windows Portability Matrix

- Windows CI runs `just ci` under PowerShell (no bash dependency)
- Test Node "min + max" versions within `runner/package.json` `engines` (pin exact versions)
- Pin Windows job tooling versions for reproducibility (Go, Node, just, gopls, baseline CLIs)
- If `just ci` includes formal model checking, provision the TLC runtime explicitly on Windows; the current lane uses Java 17 plus a cached `tla2tools.jar` exported from the flake-pinned Nix package, verifies its SHA-256, and exports `TLA2TOOLS_JAR`

```yaml
strategy:
  fail-fast: false
  matrix:
    node-version:
      - "22.22.1" # min supported
      - "24.14.0" # latest supported

steps:
  - uses: actions/setup-go@...
    with:
      go-version: "1.25.7"
  - uses: actions/setup-node@...
    with:
      node-version: ${{ matrix.node-version }}
  - uses: actions/setup-java@...
    with:
      distribution: temurin
      java-version: "17"
  - name: Restore cached TLC jar
    uses: actions/cache/restore@...
  - name: Configure cached TLC jar
    shell: pwsh
    run: |
      $jarPath = Join-Path $env:GITHUB_WORKSPACE ".ci-cache\tlaplus\tla2tools.jar"
      $sha256 = (Get-FileHash -Algorithm SHA256 $jarPath).Hash.ToLowerInvariant()
      "TLA2TOOLS_JAR=$jarPath" | Out-File -FilePath $env:GITHUB_ENV -Encoding utf8 -Append
  - run: just ci
```

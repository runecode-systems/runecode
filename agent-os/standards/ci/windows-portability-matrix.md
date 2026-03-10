# Windows Portability Matrix

- Windows CI runs `just ci` under PowerShell (no bash dependency)
- Test Node "min + max" versions within `runner/package.json` `engines` (pin exact versions)
- Pin Windows job tooling versions for reproducibility (Go, Node, just, gopls, baseline CLIs)

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
  - run: just ci
```

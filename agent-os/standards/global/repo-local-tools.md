# Repo-Local Helper Tools

- Prefer small repo-local helper programs over shell pipelines (portability, Windows)
- Keep tools deterministic and small; prefer stdlib-only/minimal deps
- Invoke tools via language runtimes (example: `go run ./tools/<tool>`)
- Place tools under `tools/<tool>/...` (example: `tools/gofmtcheck/main.go`)

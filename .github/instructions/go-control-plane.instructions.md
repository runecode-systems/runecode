---
applyTo: "cmd/**/*.go,internal/**/*.go,tools/**/*.go"
---

Use these references for Go control-plane review comments:

- `/docs/trust-boundaries.md`
- `/go.mod`
- `/justfile`
- `/.github/workflows/ci.yml`
- `/cmd/*/main.go`
- `/internal/scaffold/stub.go`

When reviewing Go-side changes, focus on:

- Trust-boundary guarantees remain intact and no new bypass channel to trusted state is introduced.
- CLI behavior remains stable: usage errors exit with code `2`, runtime failures exit `1`, and help behavior remains explicit.
- Input validation and error handling are deterministic and avoid panics on expected user errors.
- User-facing logs and errors do not leak secrets, credentials, or sensitive local paths.
- Changes remain portable with the CI matrix expectations across Linux, macOS, and Windows.

Expect matching tests or validation updates when command behavior, validation, or boundary semantics change.

---
applyTo: "cmd/**/*.go,internal/**/*.go,tools/**/*.go"
---

Use these references for Go control-plane review comments:

- `/docs/trust-boundaries.md`
- `/docs/source-quality.md`
- `/.github/instructions/source-quality.instructions.md`
- `/go.mod`
- `/justfile`
- `/.golangci.yml`
- `/.github/workflows/ci.yml`
- `/cmd/*/main.go`
- `/internal/scaffold/stub.go`

When reviewing Go-side changes, focus on:

- Trust-boundary guarantees remain intact and no new bypass channel to trusted state is introduced.
- CLI behavior remains stable: usage errors exit with code `2`, runtime failures exit `1`, and help behavior remains explicit.
- Input validation and error handling are deterministic and avoid panics on expected user errors.
- User-facing logs and errors do not leak secrets, credentials, or sensitive local paths.
- Changes remain portable with the CI matrix expectations across Linux, macOS, and Windows.
- For detailed source-quality review criteria, follow `/docs/source-quality.md` and `/.github/instructions/source-quality.instructions.md`.
- Complex trust-boundary, policy, schema-validation, secrets, and audit logic either stays locally explainable or adds maintained rationale in docs/specs/ADRs instead of relying on large inline comment blocks.

Expect matching tests or validation updates when command behavior, validation, or boundary semantics change.

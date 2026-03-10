# Go CLI `main.go` Pattern

- Keep `cmd/<bin>/main.go` thin for safety + clarity
- `main.go` is wiring only: args/help + call into `internal/...` + exit
- No business logic in `main.go`
- Scaffold phase: reuse `internal/scaffold` helpers

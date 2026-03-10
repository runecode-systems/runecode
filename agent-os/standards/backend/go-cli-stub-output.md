# Go CLI Stub Output

- Stubs have no side effects beyond stdout/stderr
  - No network listeners
  - No filesystem writes
- Stub output is stable; always include these lines:

```text
<bin> is scaffolded and not yet implemented.
No network listeners are started in this stub.
```

- Additional context lines may be appended after the stub lines
- Use `internal/scaffold.WriteStubMessage` / `WriteHelp` to keep output consistent

---
schema_version: 1
id: javascript/runner-node-test-runner
title: Node Test Runner (Runner)
status: active
aliases:
    - agent-os/standards/javascript/runner-node-test-runner
---

# Node Test Runner (Runner)

- Prefer Node's built-in test runner: `node --test`
- Keep runner tests dependency-light (Node built-ins like `node:assert/strict`)
- Avoid Jest/Vitest unless there is a clear need
- Keep `npm test` as the combined gate: `npm run lint` then tests

```json
{
  "scripts": {
    "test": "npm run lint && node --test scripts/boundary-check.test.js"
  }
}
```

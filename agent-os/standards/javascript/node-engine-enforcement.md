# Node Engine Enforcement (Runner)

- `runner/package.json` is the source of truth for supported Node versions (`engines.node`)
- Enforce supported Node versions via `runner/.npmrc`: `engine-strict=true`
- CI tests the "min + max" Node versions within the `engines` range (pin exact versions)
- When updating Node support, update `engines` and the CI matrix together (keep them in sync)

```json
{"engines": {"node": ">=22.22.1 <25"}}
```

```ini
engine-strict=true
```

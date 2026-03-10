# TypeScript-as-Lint (Runner)

- `npm run lint` is typecheck-only: `tsc --noEmit` (no ESLint)
- Keep these `runner/tsconfig.json` invariants:
  - `strict: true`
  - `noEmit: true`
  - `rootDir: "src"`
  - `module/moduleResolution: "NodeNext"`

```json
{
  "compilerOptions": {
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true,
    "noEmit": true,
    "rootDir": "src"
  }
}
```

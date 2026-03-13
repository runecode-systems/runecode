# References for Source Quality Guardrails v0

## Product Context

- **Mission:** `agent-os/product/mission.md`
- **Tech stack:** `agent-os/product/tech-stack.md`
- **Trust boundaries:** `docs/trust-boundaries.md`
- **Repo operating guidance:** `AGENTS.md`

## Related Specs

- `agent-os/specs/2026-03-08-1039-scaffold-ci-matrix/`
- `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`
- `agent-os/specs/2026-03-08-1039-policy-engine-v0/`
- `agent-os/specs/2026-03-08-1039-broker-local-api-v0/`
- `agent-os/specs/2026-03-08-1039-secretsd-model-gateway-v0/`

## Repo Files Likely Touched by Future Implementation

- `docs/source-quality.md`
- `justfile`
- `tools/`
- `.source-quality-baseline.json`
- `.golangci.yml`
- `runner/package.json`
- `runner/eslint.config.*`
- `.github/CODEOWNERS`
- `.github/copilot-instructions.md`
- `.github/instructions/go-control-plane.instructions.md`
- `.github/instructions/runner-boundary.instructions.md`
- `.github/instructions/ci-tooling.instructions.md`
- `.github/instructions/source-quality.instructions.md`
- `AGENTS.md`
- `CONTRIBUTING.md`

## External References

- Kubernetes `hack/golangci.yaml`
- Prometheus `.golangci.yml`
- Tokio crate docs in `tokio/src/lib.rs`
- Node.js `eslint.config.mjs`
- Django coding style and `PEP 257`
- ESLint `max-lines`
- ESLint `max-lines-per-function`
- ESLint `complexity`
- golangci-lint settings for `funlen`, `gocyclo`, `cyclop`, and `gocognit`
- Effective Go: Commentary and doc-comment conventions
- PEP 257: Docstring Conventions
- Rustdoc Book: crate/module documentation guidance

## Research Notes Informing This Spec

- Mature infra/security projects commonly combine targeted documentation rules, complexity checks, and architecture docs.
- Blanket "comment everything" policies are uncommon and generally avoided.
- Repo-specific source-quality scripts are useful when teams want ratcheted file budgets, module-doc requirements, or mixed-language policy enforcement that language-native linters do not cover well.
- Mature projects often enforce comment/documentation quality indirectly through architecture docs, complexity limits, and focused documentation requirements rather than a universal "add more comments" rule.
- When major projects allow suppressions, they typically treat high-risk paths and enforcement tooling itself as special review surfaces rather than ordinary code.

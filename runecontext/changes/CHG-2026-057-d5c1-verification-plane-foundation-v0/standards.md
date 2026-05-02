## Applicable Standards
- `standards/backend/go-cli-help-exit-codes.md`: Selected from the current context bundles.
- `standards/backend/go-cli-interactive-tty-guard.md`: Selected from the current context bundles.
- `standards/backend/go-cli-main-pattern.md`: Selected from the current context bundles.
- `standards/backend/go-cli-stub-output.md`: Selected from the current context bundles.
- `standards/ci/just-ci.md`: Selected from the current context bundles.
- `standards/global/language-aware-source-docs.md`: Selected from the current context bundles.
- `standards/global/protocol-bundle-manifest.md`: Selected from the current context bundles.
- `standards/global/protocol-canonicalization-profile.md`: Selected from the current context bundles.
- `standards/global/protocol-registry-discipline.md`: Selected from the current context bundles.
- `standards/global/protocol-schema-invariants.md`: Selected from the current context bundles.
- `standards/global/repo-local-tools.md`: Selected from the current context bundles.
- `standards/global/source-quality-enforcement-layering.md`: Selected from the current context bundles.
- `standards/product/roadmap-conventions.md`: Selected from the current context bundles.
- `standards/product/tui-shell-input-and-command-surfaces.md`: Selected from the current context bundles.
- `standards/security/trust-boundary-interfaces.md`: Selected from the current context bundles.
- `standards/testing/protocol-fixture-manifest-parity.md`: Selected from the current context bundles.

## Standards Added Since Last Refresh
- `standards/backend/go-cli-help-exit-codes.md`: Newly selected during standards refresh.
- `standards/backend/go-cli-interactive-tty-guard.md`: Newly selected during standards refresh.
- `standards/backend/go-cli-main-pattern.md`: Newly selected during standards refresh.
- `standards/backend/go-cli-stub-output.md`: Newly selected during standards refresh.
- `standards/ci/just-ci.md`: Newly selected during standards refresh.
- `standards/global/language-aware-source-docs.md`: Newly selected during standards refresh.
- `standards/global/protocol-schema-invariants.md`: Newly selected during standards refresh.
- `standards/global/repo-local-tools.md`: Newly selected during standards refresh.
- `standards/global/source-quality-enforcement-layering.md`: Newly selected during standards refresh.
- `standards/product/tui-shell-input-and-command-surfaces.md`: Newly selected during standards refresh.
- `standards/testing/protocol-fixture-manifest-parity.md`: Newly selected during standards refresh.

## Resolution Notes
This project change defines the verification-plane foundation across trusted control-plane evidence, protocol-governed object shapes, anti-tamper audit verification, runtime-attestation posture, and independently verifiable export.

The selected standards intentionally keep the work anchored to:

- one trusted source of truth for canonical evidence
- fail-closed trust-boundary behavior and schema-governed contracts
- deterministic canonicalization and registry discipline for any new protocol objects
- durable local evidence persistence and verifiable anti-rewrite history
- explicit verifier identity, approval binding, and runtime evidence posture

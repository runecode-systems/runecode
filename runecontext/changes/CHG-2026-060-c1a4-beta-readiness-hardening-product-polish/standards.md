## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/global/local-product-lifecycle-and-attach-contract.md`
- `standards/global/project-substrate-contract-and-lifecycle.md`
- `standards/global/session-execution-contract-and-watch-families.md`
- `standards/global/workflow-pack-routing-and-built-in-workflow-authority.md`
- `standards/product/tui-shell-input-and-command-surfaces.md`
- `standards/security/trusted-run-plan-authority-and-selection.md`
- `standards/security/trusted-runtime-evidence-and-broker-projection.md`
- `standards/security/audit-evidence-bundles-and-offline-verification.md`
- `standards/security/audit-evidence-index-and-record-inclusion.md`
- `standards/global/source-quality-enforcement-layering.md`

## Resolution Notes
This alpha hardening umbrella is intentionally product-facing rather than architecture-replacing.

The selected standards require RuneCode to keep broker-owned lifecycle and project-substrate truth authoritative, to preserve trusted `RunPlan` authority and built-in workflow selection, to keep the TUI as a strict client of broker-owned state, to preserve the reviewed runtime-evidence and attestation posture contracts, and to exercise the verification surfaces from canonical evidence rather than derived views alone.

For this lane, that means the beta proof must cover both setup/remediation lifecycle and productive workflow execution: project-substrate inspect/adopt/init/upgrade/validate/status, `change_draft`, `spec_draft`, `draft_promote_apply`, and `approved_change_implementation`. Git remote publication remains governed by its existing gateway standards and is not part of the required beta close gate unless product messaging expands to claim publishing or collaboration.

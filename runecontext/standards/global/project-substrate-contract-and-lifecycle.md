---
schema_version: 1
id: global/project-substrate-contract-and-lifecycle
title: Project Substrate Contract And Lifecycle
status: active
suggested_context_bundles:
    - project-core
    - go-control-plane
    - protocol-foundation
---

# Project Substrate Contract And Lifecycle

Canonical RuneContext project truth for a managed repository is the repo-root `runecontext.yaml` plus the reviewed `runecontext/` substrate under that same repository root.

- Do not introduce a second RuneCode-private planning or truth surface such as `.runecontext/`, a daemon-private project mirror, or any other product-only canonical store.
- Treat repository-root resolution, discovery, and validation as deterministic and read-only; do not walk arbitrarily upward for alternate anchors, and do not mutate project state during posture inspection.
- Adoption is read-only recognition of existing compatible canonical substrate; adoption must not silently normalize, repair, or rewrite the repository.
- Initialization and upgrade are explicit operator-visible mutation flows with `preview -> apply -> validate` semantics.
- Initialization writes only canonical RuneContext substrate files and directories for the selected contract version; do not create a reduced RuneCode-specific layout.
- Initialization must fail closed on conflicting candidate state or private mirrors rather than overwriting them heuristically.
- Compatibility decisions target the repository's declared and validated substrate contract, not each operator's installed RuneCode or `runectx` version.
- Normal managed operation must fail closed for missing, invalid, non-verified, or unsupported substrate states; diagnostics and remediation may remain available.
- When trusted local services are otherwise healthy, blocked project-substrate posture may still allow diagnostics/remediation-only attach so operators can inspect canonical broker-owned state without silently re-enabling normal managed execution.
- Compatible-but-upgradeable substrate remains usable, but upgrade stays advisory until the operator explicitly previews and applies it.
- Broker-owned typed contracts are the authority surface for project posture, reason codes, preview/apply results, and remediation guidance; TUI and CLI remain thin adapters over those contracts.
- Bind later planning, audit, attestation, or verification features to a validated project-substrate snapshot digest rather than to raw paths, ambient local assumptions, or version strings alone.

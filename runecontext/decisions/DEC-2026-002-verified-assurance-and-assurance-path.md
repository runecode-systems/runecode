---
schema_version: 1
id: DEC-2026-002-verified-assurance-and-assurance-path
title: Verified Assurance Adoption and Assurance Path
originating_changes:
  - CHG-2026-001-57d6-agent-os-to-runecontext-migration-umbrella
related_changes:
  - CHG-2026-046-a91d-runecontext-verified-project-substrate-compatibility-lifecycle-v0
  - CHG-2026-002-33c5-git-gateway-commit-push-pr
  - CHG-2026-003-b567-audit-log-v0-verify
  - CHG-2026-004-acdb-artifact-store-data-classes-v0
  - CHG-2026-005-cfd0-crypto-key-management-v0
  - CHG-2026-006-84f0-audit-anchoring-v0
  - CHG-2026-007-2315-policy-engine-v0
  - CHG-2026-008-62e1-broker-local-api-v0
  - CHG-2026-009-1672-launcher-microvm-backend-v0
  - CHG-2026-010-54b7-container-backend-v0-explicit-opt-in
  - CHG-2026-011-7240-secretsd-model-gateway-v0
  - CHG-2026-012-f1ef-workflow-runner-workspace-roles-deterministic-gates-v0
  - CHG-2026-013-d2c9-minimal-tui-v0
  - CHG-2026-014-0c5d-approval-profiles-strict-permissive
  - CHG-2026-015-cae6-formal-spec-v0-tla-ci-model-checking
  - CHG-2026-016-8cdb-zk-proof-v0-one-narrow-proof-verify
  - CHG-2026-017-3d58-workflow-extensibility-v0
  - CHG-2026-018-5900-auth-gateway-role-v0
  - CHG-2026-019-40c5-bridge-runtime-protocol-v0
  - CHG-2026-020-4425-openai-chatgpt-subscription-provider-oauth-codex-bridge
  - CHG-2026-021-8d6d-local-ipc-protobuf-transport-v0
  - CHG-2026-022-8051-github-copilot-subscription-provider-official-runtime-bridge
  - CHG-2026-023-59ac-web-research-role
  - CHG-2026-024-acde-deps-fetch-offline-cache
  - CHG-2026-025-5679-external-audit-anchoring-v0
  - CHG-2026-026-98be-image-toolchain-signing-pipeline
  - CHG-2026-027-71ed-workflow-concurrency-v0
  - CHG-2026-028-647e-windows-microvm-runtime-support
  - CHG-2026-029-5e5e-macos-virtualization-polish
  - CHG-2026-030-98b8-isolate-attestation-v0
---

# DEC-2026-002: Verified Assurance Adoption and Assurance Path

## Status
Accepted

## Date
2026-04-03

## Context
This repository already has a RuneContext root config and baseline assurance artifact.
The migration plan requires immediate verified-mode operation and a stable assurance location.

## Decision
- This repository adopts RuneContext in `verified` mode from the start of migration.
- Assurance artifacts for this repository remain under `runecontext/assurance/`.
- There is no temporary plain-mode migration phase for canonical migration work.

## Consequences
- Migration steps must keep `runectx validate` and `runectx status` clean while in verified mode.
- Assurance history for migration work is captured natively under the active assurance path.
- Legacy `agent-os/` material may be represented as imported history when needed; it must not be rewritten as if it were native verified provenance.

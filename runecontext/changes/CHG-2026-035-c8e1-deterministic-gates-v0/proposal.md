## Summary
RuneCode enforces deterministic quality and safety gates that are planned explicitly through immutable broker-compiled `RunPlan` entries, produce typed verifiable evidence artifacts, fail closed by default, and integrate with canonical policy approval/override semantics.

## Problem
Gate behavior and evidence semantics were previously embedded in a broad combined change, limiting focused verification. It also remained unclear whether gate ordering and checkpoint placement would be explicit planning inputs or drift into runner-local conventions.

## Proposed Change
- Implement a deterministic gate framework.
- Produce hash-addressed gate evidence artifacts.
- Define failure, retry, and override semantics with audit coverage.
- Freeze a reusable typed gate contract with stable gate identity, declared inputs, explicit attempt semantics, and policy-mediated overrides.
- Bind gate ordering and placement to `RunPlan` entries compiled from operational workflow/process definitions.

## Why Now
This split keeps gate correctness and evidence production independently reviewable while preserving end-to-end workflow traceability. Freezing plan-driven gate execution now avoids later workflow work depending on hidden gate order conventions that would be expensive to unwind.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runner durable-state internals.
- Workspace role implementation details.
- Leaving gate ordering or retry behavior to runner-local or executor-local convention.

## Impact
Keeps gate determinism and evidence semantics as a dedicated feature boundary while freezing the shared gate/evidence model that later workflow features must reuse.

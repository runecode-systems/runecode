# Design

## Overview
Define the explicit web-research gateway role with strict egress controls, citation artifacts, and no workspace-derived data exposure.

## Key Decisions
- Egress is deny-by-default and policy-driven.
- Web research must not consume workspace-derived data classes.
- Fetching is hardened against SSRF/DNS rebinding (block private/reserved IP ranges; constrain redirects).
- Web destinations should use the shared typed gateway destination/allowlist model so origin identity, redirect posture, and approved operations stay canonical across gateway features.
- Web research should also stay aligned with the shared gateway operation taxonomy and shared gateway audit field set rather than evolving a web-only outbound vocabulary where the shared model is sufficient.

## Main Workstreams
- Web Research Gateway Contract
- Egress Controls + Fetch Hardening
- Citation Artifact Model
- Policy + Audit Integration

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.

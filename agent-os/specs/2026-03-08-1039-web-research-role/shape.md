# Web Research Role — Shaping Notes

## Scope

Add a dedicated web-research role with strict allowlists and citation-only outputs.

## Decisions

- Egress is deny-by-default and policy-driven.
- Web research must not consume workspace-derived data classes.
- Fetching is hardened against SSRF/DNS rebinding (block private/reserved IP ranges; constrain redirects).

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-artifact-store-data-classes-v0/`
- Product alignment: Limits egress blast radius and preserves data-flow controls.

## Standards Applied

- None yet.

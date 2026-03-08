# Deps Fetch + Offline Cache — Shaping Notes

## Scope

Add a deps-fetch role that enables offline workspace execution by producing cache artifacts.

## Decisions

- Inputs are minimal and low-sensitivity (lockfiles only).
- Outputs are read-only artifacts.

## Context

- Visuals: None.
- References: `agent-os/specs/2026-03-08-1039-artifact-store-data-classes-v0/`
- Product alignment: Preserves offline workspace roles while supporting practical builds.

## Standards Applied

- None yet.

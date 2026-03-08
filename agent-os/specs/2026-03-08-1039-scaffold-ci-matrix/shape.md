# Monorepo Scaffold + Package Boundaries (v0) — Shaping Notes

## Scope

Set up the initial monorepo structure and baseline build/test/lint commands, with explicit trust boundaries between Go components and the TS/Node scheduler.

## Decisions

- Keep Go components and the TS workflow runner in separate packages with explicit trust boundaries.
- Developer tooling and CI conventions (Nix Flake + `direnv` + `just` + GitHub Actions) live in a dedicated spec and are implemented first.

## Context

- Visuals: None.
- References: `agent-os/product/tech-stack.md`
- Product alignment: Matches the intended split (Go security kernel + Go TUI + TS LangGraph runner treated as untrusted).

## Standards Applied

- None yet.

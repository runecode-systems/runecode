# Minimal TUI v0 — Shaping Notes

## Scope

Implement a small local TUI for approvals and run/audit visibility.

## Decisions

- TUI is a separate least-privilege client; it does not embed privileged execution.
- Use Bubble Tea as the TUI framework.
- The assurance level (microVM vs container) must be prominent.

## Context

- Visuals: None.
- References: `agent-os/product/tech-stack.md`
- Product alignment: “user in the loop” approvals and audit-first UX.

## Standards Applied

- None yet.

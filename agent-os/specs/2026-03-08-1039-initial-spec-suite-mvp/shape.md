# Initial Spec Suite (MVP + Post-MVP) — Shaping Notes

## Scope

Define a small set of initial specs for RuneCode, split into MVP vs post-MVP, and update the product roadmap accordingly.

## Decisions

- MVP uses Nix Flakes for the canonical local dev environment, with `direnv` auto-entry and `just` for common developer commands.
- MVP is single-user and single-machine (no multi-user daemon or remote control plane).
- MVP uses SQLite (WAL) for durable local state and indexing (pinned SQLite version when WAL is enabled).
- MVP includes formal verification (TLA+ model checking in CI).
- MVP includes one narrow ZK proof only if a proving system can be selected with deterministic, fast verification; otherwise ZK remains a documented interface/fixture until post-MVP.
- MVP runtime targets Linux + KVM first; macOS is included in MVP only if it does not materially slow delivery.
- Windows support in MVP is enforced via CI workflows (lint/tests/integration where possible) to keep the codebase portable; Windows microVM runtime support is post-MVP.
- MVP supports both microVM and container isolation backends.
- Container backend is explicit opt-in only and must never be an automatic fallback when microVMs fail.
- MVP starts with JSON messages validated by JSON Schema, while keeping the logical object model encoding-agnostic to allow post-MVP protobuf over local IPC for on-wire RPC (gRPC optional and local-only).
- MVP uses vsock-first for isolate <-> host transport on Linux with a virtio-serial fallback; a message-level authenticated+encrypted session is always required.
- MVP UX is CLI + minimal TUI.
- Spec docs must not mention the source discovery doc filename/path; they should stand on their own.

## Context

- Visuals: None.
- References: No existing code references (repo is currently docs-only).
- Product alignment: Aligns tightly with the product mission (security-first) and tech direction (Go control plane + Go TUI + TS LangGraph runner treated as untrusted), with a clarified constraint that container isolation is opt-in only.

## Standards Applied

- product/roadmap-conventions — Applies to the `agent-os/product/roadmap.md` update.

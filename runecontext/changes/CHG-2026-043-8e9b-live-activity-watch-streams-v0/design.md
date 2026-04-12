# Design

## Overview
Define typed live activity watch streams for the alpha TUI and related operator surfaces.

## Key Decisions
- Watch families should be explicit and typed, not one ambiguous event bus.
- Stream identity, sequencing, and terminal-state clarity should remain first-class.
- The initial watch families should cover runs, approvals, and sessions.
- Logs remain supplementary rather than the primary source of live control-plane truth.

## Main Workstreams
- `RunWatchEvent` family
- `ApprovalWatchEvent` family
- `SessionWatchEvent` family
- Shared stream-semantics alignment

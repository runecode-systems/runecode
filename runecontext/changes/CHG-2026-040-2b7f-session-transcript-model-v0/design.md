# Design

## Overview
Define the minimal broker-visible session and transcript contracts the alpha TUI should depend on for its first-class chat route.

## Key Decisions
- Session identity should be canonical and broker-visible.
- Transcript turns/messages should be ordered and linked to one session identity.
- Session interactions should use typed request/response or stream contracts rather than ad hoc client-local conventions.
- Turns may link to runs, approvals, artifacts, and audit references where those relationships exist.
- The model should stay minimal enough for alpha.3 while avoiding a client-local-only foundation.

## Main Workstreams
- Session identity model
- Transcript turn/message model
- Session interaction request/response model
- Linked-reference model for related control-plane objects

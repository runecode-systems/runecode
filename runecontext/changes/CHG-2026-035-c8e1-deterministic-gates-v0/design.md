# Design

## Overview
Implement deterministic gate execution with explicit, auditable evidence outputs and fail-closed semantics.

## Key Decisions
- Gates are deterministic and produce typed evidence artifacts.
- Gate failures fail the run by default.
- Any override requires explicit approval and audit events.
- Gate overrides should be modeled as canonical policy actions with typed approval payloads and shared reason-code semantics rather than as feature-local override exceptions.

## Main Workstreams
- Gate framework and execution order.
- Evidence artifact schema and retention linkage.
- Retry and override policy integration.

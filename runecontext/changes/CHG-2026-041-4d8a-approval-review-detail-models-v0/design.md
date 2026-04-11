# Design

## Overview
Define broker-projected approval review detail models that make approval review explainable without relying on client payload scraping or heuristic inference.

## Key Decisions
- Approval review models should expose `policy_reason_code` directly.
- Approval review models should expose binding kind explicitly.
- Structured explanation should replace dependence on one prose field for approval effect understanding.
- Stale, superseded, expired, and consumed semantics should be surfaced through typed fields and reason codes.
- The model should preserve the project’s distinction between exact-action approvals and stage sign-off.

## Main Workstreams
- Approval detail model
- Structured effect and blocked-scope model
- Lifecycle and stale/supersession reason model

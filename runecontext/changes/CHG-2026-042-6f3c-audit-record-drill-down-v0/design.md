# Design

## Overview
Define broker-owned typed audit record drill-down surfaces for the alpha TUI.

## Key Decisions
- Audit record drill-down should use broker-owned derived views.
- Timeline entries should carry stable record identity sufficient for drill-down.
- Audit drill-down should not expose ledger-private file structure.
- The model should support linked inspection with related approvals, artifacts, and verification posture where relevant.

## Main Workstreams
- Audit record detail model
- Timeline-to-record identity linkage
- Linked-reference drill-down model

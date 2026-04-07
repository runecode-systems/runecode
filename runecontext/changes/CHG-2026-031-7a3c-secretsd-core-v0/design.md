# Design

## Overview
Implement the reusable `secretsd` foundation for secret storage and lease management as a standalone feature boundary.

## Key Decisions
- Long-lived secrets are stored only in `secretsd`; other components use leases only.
- Secrets storage fails closed by default when secure key storage is unavailable.
- Secret values are never accepted via CLI args or environment variables.
- Lease issuance is short-lived, scope-bound, and fully audited.
- Local `secretsd` health/readiness is primarily a daemon/supervision surface; user-facing operational posture should converge through broker operator-facing summaries rather than a separate long-term client API.

## Main Workstreams
- Storage and key posture policy.
- Lease lifecycle rules.
- Safe secret onboarding/import flow.
- Local health and metrics surfaces.

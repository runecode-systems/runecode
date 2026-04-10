# Design

## Overview
Implement the untrusted runner orchestration and durable state authority for secure, resumable runs.

## Key Decisions
- Runner is untrusted and never directly executes privileged operations.
- Runner persistence stores control-plane state only.
- Pause/resume and crash recovery rely on durable state transitions.
- Pending approval blocks only the exact bound scope; unrelated eligible work may continue.
- Multiple pending approvals may coexist and survive restarts.
- Runner-internal durable state remains non-canonical and outside the cryptographic trust root unless exported into canonical protocol objects.
- All real execution remains brokered and policy-authorized.
- Broker-facing run and approval summaries are shared operator-facing contracts; runner state must align with them rather than inventing a second lifecycle vocabulary.
- Runner durable state may retain additional orchestration detail, but broker-visible run truth must remain an explicit translation into authoritative or advisory public fields.
- Authoritative backend/runtime facts come from launcher -> broker projection rather than from runner-local inference.
- Runner must not flatten backend kind, runtime isolation assurance, provisioning/binding posture, and audit posture into one local status string.
- Runner approval wait semantics must preserve the policy distinction between exact-action approvals and stage sign-off, including supersession when the bound stage summary hash changes.

## Main Workstreams
- Runner contract and packaging constraints.
- Durable state schema and migration rules.
- Propose-to-attest execution loop integration.

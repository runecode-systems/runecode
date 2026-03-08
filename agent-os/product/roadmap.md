# Product Roadmap

Phased development plan with prioritized features.

This is the canonical view of what is planned next (as specs) and what has shipped (as releases). Move items from "Upcoming Features" to "Completed Features" when the corresponding version is released.

## Upcoming Features

### vNext (Planned)

- [ ] Spec title (`agent-os/specs/YYYY-MM-DD-HHMM-spec-slug/`)
  - Short description of the user-visible outcome.

## Unscheduled (Needs Specs)

- [ ] Isolated execution model
  - Role-based isolates and strong blast-radius reduction.
- [ ] Signed, immutable capability manifests per stage
  - No escalation-in-place.
- [ ] Structured, auditable cross-isolate communication
  - Hash-addressed artifacts.
- [ ] Tamper-evident audit log
  - Signed events, prompts/responses, tool calls, diffs, test logs.
- [ ] Deterministic gates
  - Build/test/lint/security/policy gates that produce evidence artifacts.
- [ ] Local TUI for run control
  - Approvals, diffs, artifacts, audit timeline.
- [ ] Networked gateways split out
  - model-gateway, git-gateway, web-research with strict allowlists.

## Completed Features

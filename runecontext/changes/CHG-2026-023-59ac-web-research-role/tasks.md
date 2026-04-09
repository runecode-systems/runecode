# Tasks

## Web Research Gateway Contract

- [ ] Define the dedicated web-research gateway role.
- [ ] Keep web research isolated from workspace roles and workspace-derived data classes.
- [ ] Model web destinations through the shared typed `DestinationDescriptor` / allowlist-entry pattern.

## Egress Controls + Fetch Hardening

- [ ] Keep egress deny-by-default and policy-driven.
- [ ] Harden fetching against SSRF and DNS rebinding.
- [ ] Block private and reserved IP ranges and constrain redirects.
- [ ] Keep redirect handling aligned with the shared rule that redirects may only target separately allowlisted destinations.

## Citation Artifact Model

- [ ] Define citation artifacts and related evidence objects for fetched material.

## Policy + Audit Integration

- [ ] Keep web research as an explicit approved egress posture.
- [ ] Record fetch targets, bytes, timing, and outcomes without expanding the trust boundary.

## Acceptance Criteria

- [ ] Web research stays behind an explicit gateway role with strict allowlists.
- [ ] Workspace-derived data does not flow into web research egress.
- [ ] Citation artifacts remain auditable and reviewable.

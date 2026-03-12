# Web Research Role — Post-MVP

User-visible outcome: RuneCode can perform controlled web research in a dedicated gateway role with deny-by-default egress and explicit allowlist patterns, producing citation artifacts.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-web-research-role/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Egress Policy Model

- Support explicit domains and constrained suffix-wildcard patterns.
- Add bundle expansion support in the data model (even if MVP ships without bundles).

Parallelization: can be implemented in parallel with the policy engine's gateway allowlist model; depends on stable destination descriptor schemas.

## Task 3: Crawling + Safety Limits

- Implement crawler behavior bounded by allowlist.
- Enforce quotas (requests/bytes/time) and log all outbound requests.
- SSRF / DNS rebinding protections (required):
  - block private/link-local/loopback/reserved IP ranges for both IPv4 and IPv6 (including IPv4-mapped IPv6)
  - constrain or disable redirects by default; if enabled, validate every hop and never follow redirects to out-of-policy hosts
  - restrict schemes to `https://` (and `http://` only if explicitly allowed)
  - enforce timeouts and response size limits
  - apply content-type allowlists for stored citation excerpts

Parallelization: crawler implementation can proceed in parallel with artifact store + audit work; coordinate with broker limits and schema-defined citation artifacts.

## Task 4: Outputs as Artifacts

- Emit `web_citations` artifacts only (URLs + quoted excerpts).
- Ensure no workspace-derived data classes flow into this role.

Parallelization: can be implemented in parallel with the artifact store (data classes + flow matrix); depends on stable `web_citations` artifact schema.

## Acceptance Criteria

- Out-of-policy URLs are blocked and reported.
- Outputs are auditably stored as artifacts.

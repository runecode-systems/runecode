# Web Research Role — Post-MVP

User-visible outcome: RuneCode can perform controlled web research in a dedicated role with deny-by-default egress and explicit allowlist patterns, producing citation artifacts.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-web-research-role/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Egress Policy Model

- Support explicit domains and constrained suffix-wildcard patterns.
- Add bundle expansion support in the data model (even if MVP ships without bundles).

## Task 3: Crawling + Safety Limits

- Implement crawler behavior bounded by allowlist.
- Enforce quotas (requests/bytes/time) and log all outbound requests.
- SSRF / DNS rebinding protections (required):
  - block private/link-local/reserved IP ranges (e.g., RFC1918, loopback, link-local)
  - constrain or disable redirects by default; never follow redirects to out-of-policy hosts
  - restrict schemes to `https://` (and `http://` only if explicitly allowed)
  - enforce timeouts and response size limits
  - apply content-type allowlists for stored citation excerpts

## Task 4: Outputs as Artifacts

- Emit `web_citations` artifacts only (URLs + quoted excerpts).
- Ensure no workspace-derived data classes flow into this role.

## Acceptance Criteria

- Out-of-policy URLs are blocked and reported.
- Outputs are auditably stored as artifacts.

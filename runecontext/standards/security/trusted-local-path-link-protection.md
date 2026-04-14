---
schema_version: 1
id: security/trusted-local-path-link-protection
title: Trusted Local Path Link Protection
status: active
suggested_context_bundles:
    - go-control-plane
---

# Trusted Local Path Link Protection

When trusted RuneCode services open operator-supplied files or trusted local state roots:

- Reject symlink or reparse-point path components when resolving trusted roots or operator-provided paths
- When the final target must already exist, open it with no-follow protection for the final path component and verify the opened handle still refers to the validated object
- Validate directory-versus-file expectations explicitly and fail closed on type mismatches or target replacement during validation and open
- Allow a non-existent trailing path only when the caller is establishing a trusted local root and all existing parent components already passed link checks
- Keep Linux, macOS, and Windows protection explicit; use platform primitives such as `O_NOFOLLOW` and reparse-point checks rather than assuming one portability layer preserves the invariant
- Reuse shared trusted helpers for link-safe validation instead of ad hoc per-command path parsing for secrets, audit, artifact, runtime, or other sensitive local state
- Do not allow link-following to widen trusted file access into user-controlled or workspace-controlled locations
- Tests should cover linked parent directories, linked final targets, type mismatches, non-existent creation paths, and replace-during-open races when relevant

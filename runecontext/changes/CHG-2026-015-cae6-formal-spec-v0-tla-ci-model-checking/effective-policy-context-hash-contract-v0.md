# Effective Policy Context Hash Contract (v0)

This note closes the `manifest_hash` contract-freeze obligation owned by `CHG-2026-015` by pointing the formal model and stage-sign-off binding work at the already-canonical policy foundation implemented in `CHG-2026-007` and enforced by current trusted runtime code.

## Canonical Meaning

- `manifest_hash` is the digest of the canonical compiled effective policy context.
- It is not the digest of one raw source manifest.
- The digest is computed from RFC 8785 JCS canonical bytes of the compiled `EffectivePolicyContext` object.

## Authoritative Construction

The authoritative implementation is in:

- `internal/policyengine/compile_context.go`

Current compile flow:

1. Decode and validate the active trusted manifest inputs.
2. Compute the active capability sets and active allowlist references.
3. Build one canonical `EffectivePolicyContext` object.
4. Populate `policy_input_hashes` with the contributing immutable input digests.
5. Compute `manifest_hash` as `canonicalHashValue(context)`.

## Required Inputs

The compiled effective policy context contains, at minimum:

- fixed invariants
- active role family and role kind
- approval profile
- active role manifest hash
- active run capability manifest hash
- active stage capability manifest hash when present
- role, run, and stage manifest signer identities when present
- role, run, and stage capability sets
- effective capabilities after precedence/intersection
- active allowlist refs
- policy input hashes
- evaluation rule-set hash/schema when present

## Contributing Input Hashes

`policy_input_hashes` are the immutable digest identities that contributed to the compiled context. The current implementation includes:

- role manifest digest
- run capability manifest digest
- stage capability manifest digest when present
- active signed allowlist digests
- evaluation rule-set digest when present

These remain explicit alongside `manifest_hash` so approval, audit, and formal-model consumers can distinguish the compiled context digest from its source inputs.

## Formal-Model Contract

For `v0`, the TLA+ model treats `manifest_hash` and `policy_input_hashes` as opaque deterministic tokens.

That abstraction is intentional and does not relax the byte-level runtime contract above. The formal model depends on the frozen meaning that:

- `manifest_hash` identifies the compiled effective policy context in force
- stage sign-off and gate override bindings use that same trusted policy-context identity
- drift in source-manifest interpretation must be resolved in the policy foundation, not via feature-local hashing rules

## Governing References

- `runecontext/standards/security/policy-evaluation-foundations.md`
- `runecontext/changes/CHG-2026-007-2315-policy-engine-v0/design.md`
- `runecontext/standards/global/protocol-canonicalization-profile.md`
- `internal/policyengine/types.go`
- `internal/policyengine/compile_context.go`

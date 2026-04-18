## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/approval-binding-and-verifier-identity.md`
- `standards/global/control-plane-api-contract-shape.md`

## Resolution Notes
Expanded to include the shared policy-foundation, approval-binding, and control-plane contract standards so profile semantics inherit canonical action, approval, and hard-floor models.

This now explicitly includes the reviewed rule that exact-action hard floors such as `git_remote_ops` remain outside profile-local batching or milestone approval semantics.

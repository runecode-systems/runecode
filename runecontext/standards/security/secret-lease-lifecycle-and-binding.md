---
schema_version: 1
id: security/secret-lease-lifecycle-and-binding
title: Secret Lease Lifecycle And Binding
status: active
suggested_context_bundles:
    - go-control-plane
    - protocol-foundation
---

# Secret Lease Lifecycle And Binding

When trusted RuneCode services persist, issue, renew, revoke, or consume leased access to secret material:

- Treat `secretsd` as the only long-lived secret store; trusted services outside `secretsd` may hold leased material only transiently and must not become a second long-lived custody surface
- Use the shared typed `SecretLease` family as the canonical contract for persisted secrets and future derived short-lived tokens; do not invent daemon-private substitute lease objects for boundary-visible identity or lifecycle
- Bind every lease to the exact `consumer_id`, `role_kind`, and target `scope`; issue, renew, revoke, and retrieve must fail closed when any bound identity field changes
- Issue, renew, and revoke by stable `lease_id`; do not treat `secret_ref` alone as sufficient authority for lifecycle mutation
- Keep leases short-lived by default and enforce a trusted TTL cap even when callers request longer durations
- Preserve lifecycle truth explicitly: active, expiry, and revocation outcomes must survive restart and restore without being recomputed heuristically from transient process state
- Retrieval of secret material remains a trusted local delivery path; raw secret bytes must not become normal boundary-visible protocol objects, log content, CLI args, or environment variables
- Keep audit and policy linkage for leases explicit and durable enough to preserve prior deny, revoke, and expiry outcomes across recovery
- Startup and recovery fail closed unless secret metadata, secret-material integrity, lease state, and revocation state can be reconstructed consistently enough to enforce prior bindings
- Tests should cover bound-identity mismatch, TTL capping, renewal, revocation, retrieval denial after revoke or expiry, and restart-time fail-closed recovery

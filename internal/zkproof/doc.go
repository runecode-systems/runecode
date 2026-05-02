// Package zkproof contains trusted proof-statement compilation foundations.
//
// The v0 statement family in this package is intentionally narrow and audit-bound:
// audit.isolate_session_bound.attested_runtime_membership.v0.
//
// Proving-library details are intentionally isolated inside this internal package
// boundary so scheme-specific backend choices (for example gnark/Groth16 in v0)
// do not leak into broader trusted control-plane contracts.
//
// Production proof generation for real events depends on CHG-2026-030-98b8-
// isolate-attestation-v0 emitting eligible attested isolate_session_bound events.
package zkproof

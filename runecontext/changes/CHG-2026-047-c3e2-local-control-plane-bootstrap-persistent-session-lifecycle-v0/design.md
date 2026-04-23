# Design

## Overview
Add a coherent local bootstrap and reconnect model on top of the existing brokered control plane.

## Key Decisions
- Bootstrap and supervision remain local product mechanics; they must not become a second public authority surface.
- Broker remains the canonical operator-facing surface for readiness, version, session, run, and project-substrate posture truth.
- Sessions and runs must persist beyond the life of the TUI so clients can detach and later reconnect safely.
- Attach, detach, and reconnect flows should be defined in topology-neutral terms so later Windows and macOS service implementations do not require a contract rewrite.
- Degraded or blocked local-service posture must be explicit and broker-projected rather than inferred from client-local heuristics.
- Project-substrate discovery and compatibility evaluation must remain read-only during start, attach, reconnect, and status flows; bootstrap must not silently initialize, upgrade, or rewrite repository substrate.
- Blocked repository substrate posture must route clients to broker-owned diagnostics and remediation flows rather than implicit bootstrap repair.
- `v0` freezes one local RuneCode product instance per authoritative repository root.
- The canonical user-facing entrypoint for product lifecycle flows should be a new top-level `runecode` command rather than continuing to expose `runecode-broker serve-local` and `runecode-tui` startup ordering as the normal user model.
- Product lifecycle posture should be a dedicated broker-owned typed surface rather than being inferred from `Readiness.ready`, `VersionInfo`, local socket availability, or client-local bootstrap heuristics.
- Reconnect remains available for inspection and remediation even when current repository project-substrate posture blocks normal managed execution.
- Session object lifecycle, session work posture, and client attachment state must remain distinct concepts; client presence is not canonical lifecycle truth.
- Public lifecycle and attach semantics should target the logical RuneCode product instance for a repo rather than the exact process graph used by a given platform implementation.

## Canonical User Command Surface

`CHG-2026-047` is the point where RuneCode stops presenting as manual component assembly and starts presenting as one attachable local product. The canonical user surface should reflect that now rather than after bootstrap logic has already spread across `runecode-tui` and `runecode-broker`.

### Required Top-Level Command

- Add a new canonical user-facing command: `runecode`
- Bare `runecode` means `attach`
- Product-level lifecycle flows should include at least:
  - `runecode`
  - `runecode attach`
  - `runecode start`
  - `runecode status`
  - `runecode stop`
  - `runecode restart`

### Command Semantics

- `runecode` / `runecode attach`
  - resolve the authoritative repo root
  - ensure the repo-scoped local product instance exists and is attachable
  - attach the TUI as a thin client of broker-owned state
- `runecode start`
  - ensure the repo-scoped local product instance exists and reaches attachable broker posture without opening the TUI
- `runecode status`
  - must be non-starting
  - if a live broker is reachable, report broker-owned lifecycle posture plus project-substrate posture
  - if no live broker is reachable, report only the private bootstrap-local fact that no live product instance is reachable
- `runecode stop`
  - request clean stop of the repo-scoped local product instance
- `runecode restart`
  - request clean stop and then ensure a new attachable product instance for the same authoritative repo root

### Existing Binaries After This Change

- `runecode-broker`, `runecode-launcher`, and similar binaries remain valid low-level plumbing, admin, and dev entrypoints
- they must not remain the normal semantic source of user-facing product bootstrap or attach behavior
- future user-facing CLI ergonomics should wrap broker-owned typed flows behind `runecode` rather than expanding `runecode-broker` as the long-term operator surface

## Broker Lifecycle And Project-Substrate Posture

- Local product lifecycle posture and project-substrate posture are related but distinct surfaces.
- Broker readiness and version surfaces may include summary project-substrate signals, but the canonical project-substrate posture remains on its own dedicated broker-owned typed surface.
- Attachable clients should depend on broker-projected product lifecycle posture plus broker-projected project-substrate posture rather than inferring one from the other.
- If local services are healthy but repository substrate posture is blocked, RuneCode should attach cleanly into a diagnostics/remediation-only posture rather than pretending the product is fully ready.

### Dedicated Product Lifecycle Surface

This feature should add one broker-owned typed product lifecycle posture surface instead of continuing to overload `Readiness` and `VersionInfo` with attach semantics.

That surface should, at minimum, project:
- stable product instance identity for the repo-scoped RuneCode product instance
- lifecycle generation or equivalent restart identity so clients can distinguish reconnect-to-same-instance vs reconnect-after-restart
- normalized lifecycle posture for attachable clients
- whether attach mode is:
  - full normal-operation attach
  - diagnostics/remediation-only attach
- stable reason codes explaining degraded or blocked lifecycle posture
- small broker-owned summary cues for active sessions/runs where useful for attach UX

`Readiness` should remain a subsystem-health summary.

`VersionInfo` should remain build, bundle, and compatibility diagnostics.

`ProjectSubstratePostureGet` remains the canonical repository compatibility and remediation surface.

Clients must not infer lifecycle truth solely from:
- `Readiness.ready`
- `VersionInfo`
- local socket presence
- pidfile or lockfile presence
- transport reconnect success alone

### Attachability vs Normal Operation

This change should freeze a hard distinction between:
- attachability to a live broker-owned product instance
- permission for normal managed operation under current repository project-substrate posture

Required behavior:
- if the broker is healthy and reachable, attach may still succeed even when repository project-substrate posture blocks normal operation
- in that case the broker must project diagnostics/remediation-only posture explicitly
- clients may inspect sessions, runs, transcripts, approvals, artifacts, and audit state in that posture
- execution-sensitive actions remain blocked through broker-owned contracts until project-substrate posture returns to a supported normal-operation state

## Lifecycle Model

- Local product lifecycle should cover at least:
  - start and attach
  - detach while work continues
  - reconnect to active sessions and runs
  - blocked or degraded readiness states
  - clean stop and restart
- Local supervision detail may remain implementation-local, but attachable clients should only depend on broker-visible posture, canonical session/run state, and explicit project-substrate posture.

### Repo-Scoped Product Instance Model

`v0` should define one local RuneCode product instance per authoritative repository root.

Required properties:
- authoritative repo root selection happens before local product instance resolution
- all runtime directories, socket names, pidfiles, locks, and related local mechanics are derived from that repo-scoped product instance
- those mechanics remain implementation-private and non-authoritative
- broker handshake must confirm the client attached to the expected repo-scoped product instance rather than merely some reachable local broker

The following must not become semantic identity:
- local usernames
- socket paths
- runtime directories
- state-root paths
- host-local counters

This preserves alignment with `CHG-2026-046-a91d-runecontext-verified-project-substrate-compatibility-lifecycle-v0`, where repository project-substrate posture and identity are tied to one authoritative repository root.

### Private Bootstrap / Resolver Layer

This change should introduce one trusted internal bootstrap and attach layer shared by the canonical `runecode` command and by thin clients that need local product attachment.

That private layer should be responsible for:
- resolving the authoritative repo root
- deriving repo-scoped local runtime mechanics
- detecting stale local runtime artifacts safely
- ensuring a live broker exists for the repo-scoped product instance
- validating that the reachable broker matches the expected repo-scoped instance
- waiting for broker lifecycle posture to become attachable or diagnostics-only attachable
- returning the broker connection or connection config to the caller

That private layer must not become:
- a second public lifecycle API
- a second source of readiness truth
- a second source of project-substrate truth
- a second source of session/run truth

The following local artifacts are advisory mechanics only:
- pidfiles
- lockfiles
- runtime directories
- socket files
- launcher-private supervision state

They may aid local recovery and bootstrap, but broker handshake and broker-owned lifecycle posture remain authoritative for operator-facing behavior.

### Process Topology Neutrality

The public lifecycle model must target the logical RuneCode product instance for a repo rather than the exact process topology of the current platform implementation.

This is required because today:
- broker currently opens some trusted services as libraries/in-process mechanics
- launcher already has a separate service realization path
- future Windows and macOS realizations may prefer service-manager patterns that differ from Linux-first local spawning

Therefore:
- user-facing lifecycle flows must not expose the current daemon graph as the product model
- attach/start/stop/restart semantics must remain stable even if the local trusted implementation shifts between in-process helpers, sibling daemons, or managed services

## Main Workstreams
- Local Bootstrap and Supervision Entry Flows.
- Persistent Session Catalog and Reconnect Semantics.
- Broker-Protected Lifecycle and Project-Substrate Posture Projection.
- TUI and CLI Attach/Detach UX.

## Implementation Strategy

This feature should be implemented as a foundation-first change set rather than as a narrow MVP shortcut.

Recommended sequence:

1. Freeze the repo-scoped product-instance model and the public/private lifecycle boundary in protocol, broker, and CLI planning.
2. Add the trusted private bootstrap/resolver layer that selects authoritative repo root, derives repo-scoped local runtime mechanics, ensures a live broker exists, and validates broker identity before attach.
3. Add the dedicated broker-owned typed product lifecycle posture surface, including stable product instance identity, restart identity or lifecycle generation, attach mode, lifecycle posture, and stable reason codes.
4. Extend session summary/read-model planning with the minimal distinct broker-projected work-posture surface needed for attach/reconnect UX, without collapsing session object lifecycle and client presence into one field.
5. Introduce the new canonical top-level `runecode` command over the shared bootstrap/resolver and broker lifecycle posture model.
6. Repoint TUI startup and CLI recovery flows to the canonical `runecode` product lifecycle path instead of preserving manual `runecode-broker serve-local` sequencing as the normal-user workflow.
7. Keep existing low-level binaries as plumbing/admin/dev entrypoints rather than removing them prematurely or turning them into the long-term semantic source of user lifecycle behavior.
8. Verify repo-scoped instance identity, stale-runtime-artifact recovery, diagnostics-only attach on blocked repository substrate posture, non-starting `runecode status`, durable session/run survival across TUI close and product restart, and topology-neutral contract preservation.

The key design goal in this sequence is to avoid layering a new canonical user command on top of unresolved or transport-shaped semantics. The product command, bootstrap layer, broker lifecycle posture, and repo-scoped instance model should be introduced as one coherent foundation.

## Session And Reconnect Model

### Persistent Session Foundation

This change should build on the existing broker-owned durable session state rather than inventing client-local reconnect semantics.

The feature should preserve and extend the following principles:
- sessions remain broker-visible canonical objects
- runs remain broker-visible canonical objects
- TUI and CLI clients are attachable views over broker-owned truth
- local workbench persistence remains convenience-only and non-authoritative

### Distinct Session Concepts

The feature should explicitly separate:
- session object lifecycle
- projected session work posture
- client attachment or presence state

Required implications:
- client attachment is never canonical lifecycle truth
- whether the TUI is open, closed, or reattached must not become broker-owned session authority
- local recents, pins, last-opened session, and layout memory remain client convenience state only

### Session Summary Evolution

The current session summary foundation already carries object status, last activity, counts, and incomplete-turn state. This feature should extend that summary minimally with broker-projected work posture rather than overloading existing fields.

Recommended direction:
- keep session object `status` narrow to the lifecycle of the session object itself
- add a distinct projected work posture field and stable reason code(s) where needed for operator UX

That projected work posture should be able to distinguish high-level operator cues such as:
- idle
- running
- waiting
- blocked
- degraded
- failed

The exact vocabulary may be refined during schema work, but the separation of concepts should be frozen now.

### Reconnect Rules Frozen By This Change

This change should freeze reconnect rules at the product-lifecycle layer, not the execution-resume layer.

Required behavior in this change:
- sessions and linked runs remain inspectable after TUI close
- sessions and linked runs remain inspectable after local product restart, provided their broker-owned durable state remains available
- reconnect must depend on broker-owned state, watch streams, and canonical identities rather than client-local reconstruction
- if current project-substrate posture blocks normal operation, reconnect remains available in diagnostics/remediation-only posture

This change does not need to freeze the final execution-resume policy for drifted project context.

That remains the responsibility of `CHG-2026-048-6b7a-session-execution-orchestration-v0`, especially for:
- whether a blocked or drifted session may resume execution automatically
- how turn continuation binds to project-substrate snapshot identity
- how execution-sensitive reconnect behaves after project-context drift

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.

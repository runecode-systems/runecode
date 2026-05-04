# Design

## Overview
Add a trusted cross-machine evidence replication and restore foundation on top of the verification plane.

The goal is not to invent a second canonical store. The goal is to preserve the existing authority model while making evidence durable, repairable, and portable across machines that may lose space, power, or their entire local disk.

## Key Decisions
- Local canonical evidence on each node remains authoritative for what that node observed and persisted; remote stores do not become a second truth surface.
- The primary replicated units are immutable canonical evidence objects and signed replication checkpoint manifests, not exported bundles alone.
- Verifier-friendly evidence bundles and signed export manifests are replicated when they are explicitly created, but they remain derivative export artifacts rather than replication authority.
- Replication authority stays in trusted broker and auditd code. The runner remains uninvolved in replication truth, restore admission, or durability gating.
- An optional trusted helper may handle queued upload, download, retry, and anti-entropy work, but it must remain subordinate to broker and auditd authority rather than becoming a second public control surface.
- Remote targets use typed S3-compatible descriptors with tenant and project scoped namespace layout, but remote object paths and prefixes remain storage mechanics only.
- Signed replication checkpoints, not remote bucket listings, are the authoritative coordination skeleton for cross-machine durability, fetch-on-miss, and repair.
- Historical local evidence may be garbage-collected only after trusted code confirms required remote durability.
- Nodes keep a compact local checkpoint and index skeleton so ordinary new work does not require re-downloading full historical evidence.
- Publication-sensitive actions require a pre-action durability barrier: the evidence that justifies the action must be sealed or checkpointed and durably replicated to the healthy replica set before execution.
- Publication-sensitive actions also require durable prepare and execute plus reconcile recovery so crashes after remote mutation do not create unverifiable gaps.
- RuneCode must never introduce a permanent lower-assurance publication path for degraded-state changes. Surviving degraded-state edits may be captured only as non-authoritative recovery seeds that feed a new healthy reimplementation run.
- Healthy self-healing durability requires two independent remote targets. A single remote target may remain supported as a basic degraded durability posture, not the healthy posture.
- One architecture must hold across constrained devices and larger installations; only worker counts, cache size, and target count may vary.

## Trust Model

### Canonical Local Authority
- `auditd` owns canonical evidence persistence, checkpoint creation, restore admission, and GC eligibility.
- `broker` owns operator-facing posture, policy gating, publication barriers, and typed APIs.
- `secretsd` owns remote-target credential storage and short-lived lease issuance for authenticated remote replication when required.
- The TUI and CLI remain thin clients of broker-owned truth.
- The runner remains untrusted and must not become replication or restore authority.

### Remote Durability Substrate
Remote object stores are treated as durable but not authoritative.

They may:
- omit objects
- reorder listings
- surface stale listings
- lose data
- expose mutable metadata or timestamps that are not trustworthy as semantic inputs

Therefore RuneCode must verify remote content by:
- immutable digest identity
- signed checkpoint linkage
- typed target identity
- local trusted admission rules

No verification or restore rule may depend on bucket listing order, mutable marker files, or path naming as semantic authority.

## Replicated Object Families

### Required Primary Replication Units
- sealed audit segments
- audit segment seal envelopes
- receipt sidecars
- verification report sidecars
- runtime and attestation evidence sidecars
- external anchor evidence and sidecars
- verifier and support objects required for later verification recomputation
- imported-evidence authoritative payloads when later import lanes exist
- signed replication checkpoint manifests

### Replicated Derivative Export Objects
- signed bundle manifests
- exported verifier-friendly evidence bundles

These export objects should be replicated when they are explicitly created or policy requires their preservation, but they must not replace primary object replication.

## Replication Checkpoint Model

### Why A New Checkpoint Family Exists
Bundle manifests are export artifacts for independent verifiers. They are not the right long-lived coordination primitive for internal durability, anti-entropy, and thin-local restore.

This change should therefore introduce a distinct signed replication checkpoint family.

### Checkpoint Requirements
Each checkpoint should bind at least:
- project identity
- repo-scoped product instance identity or equivalent node-scoped emitter identity
- checkpoint digest and previous-checkpoint link when present
- created-at timestamp
- scope of evidence covered
- root digests and seal references needed to prove the covered history boundary
- included object digests grouped by family or shard
- verification-support object digests required for later recomputation when relevant
- durability target set or durability policy context used for evaluation

### Checkpoint Semantics
- checkpoints are append-only and signed
- checkpoints describe a consistent immutable evidence set boundary
- checkpoints are the trusted skeleton used for anti-entropy, completeness review, local thinning eligibility, and fetch-on-miss repair
- checkpoints must remain valid even when some bulky local historical objects have been GC'd on the node that produced them

## Remote Target Model

### Typed Replica Target Descriptor
Remote durability targets should use a provider-neutral typed descriptor that binds at least:
- target kind
- tenant identity
- project identity
- bucket or account identity
- namespace or prefix contract
- immutable object keying rules
- required transport security posture
- auth requirement posture

### Namespace Layout
The remote layout should support tenant and project scoped organization, for example:

- `tenants/{tenant_identity}/projects/{project_identity}/objects/{family}/{digest}`
- `tenants/{tenant_identity}/projects/{project_identity}/checkpoints/{checkpoint_digest}.json`
- `tenants/{tenant_identity}/projects/{project_identity}/exports/manifests/{manifest_digest}.json`
- `tenants/{tenant_identity}/projects/{project_identity}/exports/bundles/{bundle_digest}.tar`

These are storage mechanics only. The authoritative identity remains the typed descriptor plus object digest and signed checkpoint content.

## Local Storage Tiers

### Hot Local Evidence
Must remain local while active:
- open segment or equivalent write-ahead evidence buffer
- active run evidence
- unreplicated backlog
- durable publication prepare records awaiting execute or reconcile
- any object not yet admitted into a durable checkpoint

### Compact Local Skeleton
Should remain local even on thin-history nodes:
- signed replication checkpoints
- sparse evidence index or checkpoint summaries
- enough metadata to know what remote evidence exists and what local history has been thinned
- local product-instance and project identity state needed for safe new work and targeted repair

### Hydrated Historical Evidence
- bulky historical canonical objects may be present temporarily
- they may be fetched on demand for inspection, verification, or repair
- they may be GC'd again after use once durability rules remain satisfied

This split is what allows developer machines to free large amounts of space without losing the ability to start new work safely.

## Durability Posture

### Posture Levels
This change should define at least:
- `healthy`
- `remote_durability_degraded`
- `local_capture_unhealthy`

### Meaning
- `healthy`: local capture is healthy and required remote durability targets are satisfied
- `remote_durability_degraded`: local capture is healthy, but required healthy remote durability targets are not satisfied
- `local_capture_unhealthy`: local evidence capture itself cannot be trusted to persist mutation-bearing work safely

### Healthy Requirement
Healthy self-healing durability should require two independent remote targets.

One remote target may remain supported for operators who accept lower durability, but that posture must be explicit and degraded.

## GC And Thin-Local Rules

### Historical GC Eligibility
Historical local canonical evidence may be fully GC'd only when all are true:
- the evidence is covered by a signed replication checkpoint
- the checkpoint and all referenced required objects are durably replicated to the configured minimum GC durability target set
- no active run, prepare record, or pending reconcile still depends on those local copies
- local checkpoint skeleton state remains sufficient to prove what was removed and what can be restored later

### GC Rules
- open segments are never historical GC candidates
- unreplicated objects are never GC candidates
- missing or ambiguous replication confirmation fails closed
- local GC itself should emit canonical meta-audit evidence

## Fetch-On-Miss, Restore, And Repair

### Fetch-On-Miss
When verification or operator inspection requires an object not held locally:
1. consult local checkpoint skeleton and sparse index
2. resolve the required object digests from signed checkpoints
3. fetch missing immutable objects from configured remote targets
4. verify digests, signatures, and checkpoint membership
5. admit the objects locally through trusted restore or import rules
6. rebuild or refresh derived indexes as needed

### Restore
Restore should never mean replacing local authority with whatever the remote bucket currently contains.

Restore means:
- fetch the requested immutable objects and checkpoints
- verify them against typed target identity and signed checkpoint content
- admit them locally as imported or restored canonical evidence through trusted rules
- rebuild derived indexes deterministically
- emit import or restore meta-audit evidence

### Anti-Entropy
Anti-entropy should be checkpoint-driven rather than listing-driven:
- compare signed checkpoints and expected object sets
- detect missing local or remote objects
- repair from any surviving durable source
- keep bucket listing, timestamps, or mutable markers advisory only

## Publication-Sensitive Durability Barrier

### Hard-Floor Actions
This change should treat at least these as publication-sensitive:
- git remote push
- tag publication
- pull request creation when it is the reviewed remote mutation act
- other later remote-state mutation or publication actions classified into the same hard-floor lane

### Required Sequence
Before a publication-sensitive action executes, RuneCode must:
1. seal or checkpoint the evidence that justifies the action
2. create a signed replication checkpoint manifest for that evidence boundary
3. replicate the checkpoint and referenced required immutable objects to the healthy replica set
4. confirm durability against the healthy publication rule
5. persist a durable exact-action prepare record bound to:
   - repository identity
   - target refs or equivalent target identity
   - referenced patch or input digests
   - expected result tree hash or equivalent expected outcome identity
   - canonical action request hash
   - bound evidence checkpoint digest
6. only then execute the remote mutation
7. persist outcome evidence and a new post-action checkpoint
8. reconcile and replicate post-action evidence before reporting the action as fully complete

### Why The Barrier Is Not Sufficient Alone
Even with a pre-action durability barrier, a machine may fail immediately after the remote mutation executes.

Therefore the design still requires durable prepare and execute plus reconcile semantics rather than relying only on a pre-action flush.

## Degraded-State Recovery Seeds

### What Is Forbidden
RuneCode must not introduce a permanent lower-assurance publication path for degraded-state changes.

That means:
- no break-glass publication of inadequately evidenced degraded-state work in this change
- no durable exception lane that treats degraded-state self-signoff as equivalent to healthy reviewed publication

### Recovery Model
If degraded-state edits survive outside a healthy evidentiary run, RuneCode may capture them only as a recovery seed, such as:
- diff against a bound base tree or commit
- surviving file snapshots
- operator description of intent
- references to approved specs or change docs when available

That recovery seed is:
- non-authoritative operator input
- not canonical historical proof of the degraded period
- not directly publishable

The publishable outcome must come from a fresh healthy audited run that re-creates the intended change through normal reviewed workflow, approval, and evidence rules.

## Optional Trusted Helper
An optional helper may be added for:
- upload queues
- download queues
- bounded concurrency
- retry and backoff
- anti-entropy scheduling

Rules:
- it remains in the trusted domain
- it exposes no second public authority API
- it executes only broker and auditd-owned work
- it does not become the source of replication truth, restore admissibility, or policy decisions

## Performance And Scaling
- keep one architecture across constrained and scaled environments
- stream large object upload and download where possible
- keep checkpoint generation and admission incremental
- avoid holding audit-ledger locks during remote I/O
- use bounded workers and explicit retries
- keep local skeleton state compact enough for storage-constrained developer machines
- keep full rebuild and full verification possible from canonical evidence plus replicated checkpoints alone

## Main Workstreams
- Replication Checkpoint Object Model
- Typed Remote Replica Target Descriptors
- Thin-Local Retention And GC Rules
- Fetch-On-Miss Restore And Anti-Entropy Repair
- Publication-Sensitive Durability Barrier And Recovery
- Meta-Audit And Durability Posture Reporting

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, typed contracts, or trusted state, the plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.

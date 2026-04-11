# Tasks

## Session Identity

- [ ] Define canonical session identity for the broker-visible session model.
- [ ] Define session summary/detail expectations sufficient for the alpha TUI chat route.

## Transcript Model

- [ ] Define ordered transcript turn/message contracts.
- [ ] Define how transcript items link to related runs, approvals, artifacts, and audit references.

## Interaction Model

- [ ] Define typed send-message request/response or equivalent broker-mediated session interaction.
- [ ] Keep the contract suitable for later session watch-stream work.

## Acceptance Criteria

- [ ] The alpha TUI can depend on a canonical session/transcript model rather than client-local-only state.
- [ ] The model is minimal but does not block later multi-session work.

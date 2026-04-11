# Tasks

## Session Identity

- [x] Define canonical session identity for the broker-visible session model.
- [x] Define session summary/detail expectations sufficient for the alpha TUI chat route.

## Transcript Model

- [x] Define ordered transcript turn/message contracts.
- [x] Define how transcript items link to related runs, approvals, artifacts, and audit references.

## Interaction Model

- [x] Define typed send-message request/response or equivalent broker-mediated session interaction.
- [x] Keep the contract suitable for later session watch-stream work.

## Acceptance Criteria

- [x] The alpha TUI can depend on a canonical session/transcript model rather than client-local-only state.
- [x] The model is minimal but does not block later multi-session work.

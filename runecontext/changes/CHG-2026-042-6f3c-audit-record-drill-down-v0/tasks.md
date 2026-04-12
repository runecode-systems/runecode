# Tasks

## Audit Record Detail

- [x] Define typed audit record drill-down reads (`AuditRecordGetRequest`/`AuditRecordGetResponse`; fixtures `audit-record-get-request.valid-basic`, `audit-record-get-response.valid-detail`).
- [x] Define the minimum detail surface needed for the alpha TUI audit route (`AuditRecordDetail`; fixture `audit-record-detail.valid-basic`).

## Identity And Linking

- [x] Define stable record identity linkage from timeline views into drill-down reads (`AuditTimelineResponse.$defs.viewEntry.record_digest` + `AuditRecordGetRequest.record_digest`; fixture `audit-timeline-response.valid-linked-records`).
- [x] Define linked-reference expectations for related approvals, artifacts, and verification posture where useful (`linked_references` + `verification_posture`; fixtures `audit-timeline-response.valid-linked-records`, `audit-record-detail.invalid-approval-link-format`).

## Acceptance Criteria

- [x] The alpha TUI can inspect audit record detail through typed broker-owned reads (`AuditRecordGetRequest`/`AuditRecordGetResponse` + `AuditRecordDetail`; enforced by fixture-manifest validation in `go test ./internal/protocolschema` and `node --test runner/scripts/protocol-fixtures.test.js`).
- [x] Audit drill-down does not require direct ledger or daemon-private storage access (`AuditRecordDetail`/`AuditRecordGetResponse` are closed typed projections with `additionalProperties: false`; enforced by fixture `audit-record-get-response.invalid-daemon-private-storage-access` expecting schema failure).

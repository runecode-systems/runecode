# Verification

## Planned Checks
- `runectx validate --json`
- `go test ./internal/protocolschema`
- `cd runner && node --test scripts/protocol-fixtures.test.js`

## Verification Notes
- Typed broker-owned drill-down is explicitly covered by protocol artifacts and fixtures:
  - `protocol/schemas/objects/AuditRecordGetRequest.schema.json`
  - `protocol/schemas/objects/AuditRecordGetResponse.schema.json`
  - `protocol/schemas/objects/AuditRecordDetail.schema.json`
  - `protocol/fixtures/schema/audit-record-get-request.valid-basic.json`
  - `protocol/fixtures/schema/audit-record-get-response.valid-detail.json`
- Timeline-to-detail linkage is explicitly covered by `record_digest` parity in:
  - `protocol/schemas/objects/AuditTimelineResponse.schema.json` (`$defs.viewEntry.record_digest`)
  - `protocol/schemas/objects/AuditRecordGetRequest.schema.json` (`record_digest`)
  - `protocol/fixtures/schema/audit-timeline-response.valid-linked-records.json`
- No direct ledger/daemon-private storage requirement is enforced fail-closed via schema closure (`additionalProperties: false`) and negative fixture:
  - `protocol/fixtures/schema/audit-record-get-response.invalid-daemon-private-storage-access.json` (`expect_valid: false` in fixture manifest)

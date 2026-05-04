package auditd

type ledgerState struct {
	SchemaVersion                int    `json:"schema_version"`
	LedgerIdentity               string `json:"ledger_identity,omitempty"`
	CurrentOpenSegmentID         string `json:"current_open_segment_id"`
	NextSegmentNumber            int64  `json:"next_segment_number"`
	OpenFrameCount               int    `json:"open_frame_count"`
	LastSealEnvelopeDigest       string `json:"last_seal_envelope_digest,omitempty"`
	LastSealedSegmentID          string `json:"last_sealed_segment_id,omitempty"`
	LastVerificationReportDigest string `json:"last_verification_report_digest,omitempty"`
	RecoveryComplete             bool   `json:"recovery_complete"`
	LastIndexedRecordCount       int    `json:"last_indexed_record_count"`
}

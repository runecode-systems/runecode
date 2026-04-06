package auditd

import "github.com/runecode-ai/runecode/internal/trustpolicy"

type AppendResult struct {
	SegmentID    string             `json:"segment_id"`
	RecordDigest trustpolicy.Digest `json:"record_digest"`
	ByteLength   int64              `json:"byte_length"`
	FrameCount   int                `json:"frame_count"`
}

type SealResult struct {
	SegmentID          string             `json:"segment_id"`
	SealEnvelopeDigest trustpolicy.Digest `json:"seal_envelope_digest"`
	NextOpenSegmentID  string             `json:"next_open_segment_id"`
}

type VerificationResult struct {
	SegmentID    string                                     `json:"segment_id"`
	ReportDigest trustpolicy.Digest                         `json:"report_digest"`
	Report       trustpolicy.AuditVerificationReportPayload `json:"report"`
}

type TimelinePointer struct {
	SegmentID       string `json:"segment_id"`
	FrameIndex      int    `json:"frame_index"`
	RecordDigest    string `json:"record_digest"`
	EmitterStreamID string `json:"emitter_stream_id"`
	Sequence        int64  `json:"seq"`
	OccurredAt      string `json:"occurred_at"`
	RunID           string `json:"run_id,omitempty"`
}

type derivedIndex struct {
	BuiltAt      string            `json:"built_at"`
	TotalRecords int               `json:"total_records"`
	RunTimeline  []TimelinePointer `json:"run_timeline"`
}

type ledgerState struct {
	SchemaVersion                int    `json:"schema_version"`
	CurrentOpenSegmentID         string `json:"current_open_segment_id"`
	NextSegmentNumber            int64  `json:"next_segment_number"`
	OpenFrameCount               int    `json:"open_frame_count"`
	LastSealEnvelopeDigest       string `json:"last_seal_envelope_digest,omitempty"`
	LastSealedSegmentID          string `json:"last_sealed_segment_id,omitempty"`
	LastVerificationReportDigest string `json:"last_verification_report_digest,omitempty"`
	RecoveryComplete             bool   `json:"recovery_complete"`
	LastIndexedRecordCount       int    `json:"last_indexed_record_count"`
}

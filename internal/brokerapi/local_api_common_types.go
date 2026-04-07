package brokerapi

import (
	"io"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

const (
	artifactSummarySchemaPath              = "objects/ArtifactSummary.schema.json"
	brokerReadinessSchemaPath              = "objects/BrokerReadiness.schema.json"
	brokerVersionInfoSchemaPath            = "objects/BrokerVersionInfo.schema.json"
	artifactListRequestSchemaPath          = "objects/ArtifactListRequest.schema.json"
	artifactListResponseSchemaPath         = "objects/ArtifactListResponse.schema.json"
	artifactHeadRequestSchemaPath          = "objects/ArtifactHeadRequest.schema.json"
	artifactHeadResponseSchemaPath         = "objects/ArtifactHeadResponse.schema.json"
	artifactReadRequestSchemaPath          = "objects/ArtifactReadRequest.schema.json"
	artifactStreamEventSchemaPath          = "objects/ArtifactStreamEvent.schema.json"
	logStreamEventSchemaPath               = "objects/LogStreamEvent.schema.json"
	logStreamRequestSchemaPath             = "objects/LogStreamRequest.schema.json"
	readinessGetRequestSchemaPath          = "objects/ReadinessGetRequest.schema.json"
	readinessGetResponseSchemaPath         = "objects/ReadinessGetResponse.schema.json"
	versionInfoGetRequestSchemaPath        = "objects/VersionInfoGetRequest.schema.json"
	versionInfoGetResponseSchemaPath       = "objects/VersionInfoGetResponse.schema.json"
	auditTimelineRequestSchemaPath         = "objects/AuditTimelineRequest.schema.json"
	auditTimelineResponseSchemaPath        = "objects/AuditTimelineResponse.schema.json"
	auditVerificationGetRequestSchemaPath  = "objects/AuditVerificationGetRequest.schema.json"
	auditVerificationGetResponseSchemaPath = "objects/AuditVerificationGetResponse.schema.json"
)

type ArtifactSummary struct {
	SchemaID             string                      `json:"schema_id"`
	SchemaVersion        string                      `json:"schema_version"`
	Reference            artifacts.ArtifactReference `json:"reference"`
	CreatedAt            string                      `json:"created_at"`
	CreatedByRole        string                      `json:"created_by_role"`
	RunID                string                      `json:"run_id,omitempty"`
	StageID              string                      `json:"stage_id,omitempty"`
	StepID               string                      `json:"step_id,omitempty"`
	ApprovalOfDigest     string                      `json:"approval_of_digest,omitempty"`
	ApprovalDecisionHash string                      `json:"approval_decision_hash,omitempty"`
}

type BrokerReadiness struct {
	SchemaID                  string `json:"schema_id"`
	SchemaVersion             string `json:"schema_version"`
	Ready                     bool   `json:"ready"`
	LocalOnly                 bool   `json:"local_only"`
	ConsumptionChannel        string `json:"consumption_channel"`
	RecoveryComplete          bool   `json:"recovery_complete"`
	AppendPositionStable      bool   `json:"append_position_stable"`
	CurrentSegmentWritable    bool   `json:"current_segment_writable"`
	VerifierMaterialAvailable bool   `json:"verifier_material_available"`
	DerivedIndexCaughtUp      bool   `json:"derived_index_caught_up"`
}

type BrokerVersionInfo struct {
	SchemaID                    string   `json:"schema_id"`
	SchemaVersion               string   `json:"schema_version"`
	ProductVersion              string   `json:"product_version"`
	BuildRevision               string   `json:"build_revision"`
	BuildTime                   string   `json:"build_time"`
	ProtocolBundleVersion       string   `json:"protocol_bundle_version"`
	ProtocolBundleManifestHash  string   `json:"protocol_bundle_manifest_hash"`
	APIFamily                   string   `json:"api_family"`
	APIVersion                  string   `json:"api_version"`
	SupportedTransportEncodings []string `json:"supported_transport_encodings"`
}

type LocalArtifactListRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Cursor        string `json:"cursor,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	Order         string `json:"order,omitempty"`
	RunID         string `json:"run_id,omitempty"`
	StepID        string `json:"step_id,omitempty"`
	DataClass     string `json:"data_class,omitempty"`
}

type LocalArtifactListResponse struct {
	SchemaID      string            `json:"schema_id"`
	SchemaVersion string            `json:"schema_version"`
	RequestID     string            `json:"request_id"`
	Order         string            `json:"order"`
	Artifacts     []ArtifactSummary `json:"artifacts"`
	NextCursor    string            `json:"next_cursor,omitempty"`
}

type LocalArtifactHeadRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Digest        string `json:"digest"`
}

type LocalArtifactHeadResponse struct {
	SchemaID      string          `json:"schema_id"`
	SchemaVersion string          `json:"schema_version"`
	RequestID     string          `json:"request_id"`
	Artifact      ArtifactSummary `json:"artifact"`
}

type ArtifactReadRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Digest        string `json:"digest"`
	ProducerRole  string `json:"producer_role"`
	ConsumerRole  string `json:"consumer_role"`
	ManifestOptIn bool   `json:"manifest_opt_in,omitempty"`
	DataClass     string `json:"data_class,omitempty"`
	RangeStart    *int64 `json:"range_start,omitempty"`
	RangeEnd      *int64 `json:"range_end,omitempty"`
	StreamID      string `json:"stream_id,omitempty"`
	ChunkBytes    int    `json:"chunk_bytes,omitempty"`
}

type ArtifactReadHandle struct {
	RequestID  string
	Digest     string
	DataClass  artifacts.DataClass
	StreamID   string
	ChunkBytes int
	Reader     io.ReadCloser
}

type ArtifactStreamEvent struct {
	SchemaID       string         `json:"schema_id"`
	SchemaVersion  string         `json:"schema_version"`
	StreamID       string         `json:"stream_id"`
	RequestID      string         `json:"request_id"`
	Seq            int64          `json:"seq"`
	EventType      string         `json:"event_type"`
	Digest         string         `json:"digest,omitempty"`
	DataClass      string         `json:"data_class,omitempty"`
	ChunkBase64    string         `json:"chunk_base64,omitempty"`
	ChunkBytes     int            `json:"chunk_bytes,omitempty"`
	EOF            bool           `json:"eof,omitempty"`
	Terminal       bool           `json:"terminal,omitempty"`
	TerminalStatus string         `json:"terminal_status,omitempty"`
	Error          *ProtocolError `json:"error,omitempty"`
}

type LogStreamEvent struct {
	SchemaID       string         `json:"schema_id"`
	SchemaVersion  string         `json:"schema_version"`
	StreamID       string         `json:"stream_id"`
	RequestID      string         `json:"request_id"`
	Seq            int64          `json:"seq"`
	EventType      string         `json:"event_type"`
	RunID          string         `json:"run_id,omitempty"`
	RoleInstanceID string         `json:"role_instance_id,omitempty"`
	Cursor         string         `json:"cursor,omitempty"`
	Timestamp      string         `json:"timestamp,omitempty"`
	Level          string         `json:"level,omitempty"`
	Message        string         `json:"message,omitempty"`
	Terminal       bool           `json:"terminal,omitempty"`
	TerminalStatus string         `json:"terminal_status,omitempty"`
	Error          *ProtocolError `json:"error,omitempty"`
}

type logStreamRecord struct {
	RunID          string
	RoleInstanceID string
	Cursor         string
	Timestamp      string
	Level          string
	Message        string
}

type LogStreamRequest struct {
	SchemaID       string `json:"schema_id"`
	SchemaVersion  string `json:"schema_version"`
	RequestID      string `json:"request_id"`
	StreamID       string `json:"stream_id"`
	RunID          string `json:"run_id,omitempty"`
	RoleInstanceID string `json:"role_instance_id,omitempty"`
	StartCursor    string `json:"start_cursor,omitempty"`
	Follow         bool   `json:"follow"`
	IncludeBacklog bool   `json:"include_backlog"`
}

type ReadinessGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type ReadinessGetResponse struct {
	SchemaID      string          `json:"schema_id"`
	SchemaVersion string          `json:"schema_version"`
	RequestID     string          `json:"request_id"`
	Readiness     BrokerReadiness `json:"readiness"`
}

type VersionInfoGetRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type VersionInfoGetResponse struct {
	SchemaID      string            `json:"schema_id"`
	SchemaVersion string            `json:"schema_version"`
	RequestID     string            `json:"request_id"`
	VersionInfo   BrokerVersionInfo `json:"version_info"`
}

func toArtifactSummary(record artifacts.ArtifactRecord) ArtifactSummary {
	return ArtifactSummary{SchemaID: "runecode.protocol.v0.ArtifactSummary", SchemaVersion: "0.1.0", Reference: record.Reference, CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339), CreatedByRole: record.CreatedByRole, RunID: record.RunID, StageID: stageIDForArtifactSummary(record), StepID: record.StepID, ApprovalOfDigest: record.ApprovalOfDigest, ApprovalDecisionHash: record.ApprovalDecisionHash}
}

func stageIDForArtifactSummary(record artifacts.ArtifactRecord) string {
	if record.RunID == "" {
		return ""
	}
	return "artifact_flow"
}

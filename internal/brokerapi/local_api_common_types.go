package brokerapi

import (
	"context"
	"io"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
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
	RequestCtx context.Context
	Cancel     context.CancelFunc
	Release    func()
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
	SchemaID       string             `json:"schema_id"`
	SchemaVersion  string             `json:"schema_version"`
	RequestID      string             `json:"request_id"`
	StreamID       string             `json:"stream_id"`
	RunID          string             `json:"run_id,omitempty"`
	RoleInstanceID string             `json:"role_instance_id,omitempty"`
	StartCursor    string             `json:"start_cursor,omitempty"`
	Follow         bool               `json:"follow"`
	IncludeBacklog bool               `json:"include_backlog"`
	RequestCtx     context.Context    `json:"-"`
	Cancel         context.CancelFunc `json:"-"`
	Release        func()             `json:"-"`
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

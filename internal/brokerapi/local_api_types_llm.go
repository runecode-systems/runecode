package brokerapi

import (
	"context"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type LLMInvokeRequest struct {
	SchemaID      string              `json:"schema_id"`
	SchemaVersion string              `json:"schema_version"`
	RequestID     string              `json:"request_id"`
	RunID         string              `json:"run_id"`
	LLMRequest    any                 `json:"llm_request"`
	RequestDigest *trustpolicy.Digest `json:"request_digest,omitempty"`
}

type LLMInvokeResponse struct {
	SchemaID      string             `json:"schema_id"`
	SchemaVersion string             `json:"schema_version"`
	RequestID     string             `json:"request_id"`
	RunID         string             `json:"run_id"`
	RequestDigest trustpolicy.Digest `json:"request_digest"`
	Response      any                `json:"response"`
}

type LLMStreamRequest struct {
	SchemaID      string              `json:"schema_id"`
	SchemaVersion string              `json:"schema_version"`
	RequestID     string              `json:"request_id"`
	RunID         string              `json:"run_id"`
	StreamID      string              `json:"stream_id"`
	LLMRequest    any                 `json:"llm_request"`
	RequestDigest *trustpolicy.Digest `json:"request_digest,omitempty"`
	Follow        bool                `json:"follow"`
	RequestCtx    context.Context     `json:"-"`
	Cancel        context.CancelFunc  `json:"-"`
	Release       func()              `json:"-"`
}

type LLMStreamEnvelope struct {
	SchemaID      string             `json:"schema_id"`
	SchemaVersion string             `json:"schema_version"`
	RequestID     string             `json:"request_id"`
	RunID         string             `json:"run_id"`
	RequestDigest trustpolicy.Digest `json:"request_digest"`
	Events        []LLMStreamAny     `json:"events"`
}

type LLMStreamAny map[string]any

type llmExecutionBinding struct {
	RequestDigest  trustpolicy.Digest
	RequestHash    trustpolicy.Digest
	ResponseHash   trustpolicy.Digest
	LeaseID        string
	ProviderID     string
	ProviderFamily string
	AdapterKind    string
	PolicyRef      string
	StartedAt      time.Time
	CompletedAt    time.Time
	OutboundBytes  int64
}

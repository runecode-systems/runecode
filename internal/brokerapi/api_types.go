package brokerapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

const (
	defaultRequestIDFallback = "invalid_request"

	brokerArtifactListRequestSchemaPath  = "objects/BrokerArtifactListRequest.schema.json"
	brokerArtifactListResponseSchemaPath = "objects/BrokerArtifactListResponse.schema.json"
	brokerArtifactHeadRequestSchemaPath  = "objects/BrokerArtifactHeadRequest.schema.json"
	brokerArtifactHeadResponseSchemaPath = "objects/BrokerArtifactHeadResponse.schema.json"
	brokerArtifactPutRequestSchemaPath   = "objects/BrokerArtifactPutRequest.schema.json"
	brokerArtifactPutResponseSchemaPath  = "objects/BrokerArtifactPutResponse.schema.json"
	brokerErrorResponseSchemaPath        = "objects/BrokerErrorResponse.schema.json"
	errorEnvelopeSchemaVersion           = "0.3.0"
	errorResponseSchemaVersion           = "0.1.0"
)

type Limits struct {
	MaxMessageBytes        int
	MaxStructuralDepth     int
	MaxArrayLength         int
	MaxObjectProperties    int
	MaxInFlightPerClient   int
	MaxInFlightPerLane     int
	DefaultRequestDeadline time.Duration
	MaxStreamChunkBytes    int
	StreamIdleTimeout      time.Duration
	MaxResponseStreamBytes int
}

func DefaultLimits() Limits {
	return Limits{
		MaxMessageBytes:        1 << 20,
		MaxStructuralDepth:     64,
		MaxArrayLength:         10_000,
		MaxObjectProperties:    1_000,
		MaxInFlightPerClient:   64,
		MaxInFlightPerLane:     32,
		DefaultRequestDeadline: 30 * time.Second,
		MaxStreamChunkBytes:    64 << 10,
		StreamIdleTimeout:      15 * time.Second,
		MaxResponseStreamBytes: 16 << 20,
	}
}

type APIConfig struct {
	Limits Limits
}

func (c APIConfig) withDefaults() APIConfig {
	defaults := DefaultLimits()
	if c.Limits.MaxMessageBytes <= 0 {
		c.Limits.MaxMessageBytes = defaults.MaxMessageBytes
	}
	if c.Limits.MaxStructuralDepth <= 0 {
		c.Limits.MaxStructuralDepth = defaults.MaxStructuralDepth
	}
	if c.Limits.MaxArrayLength <= 0 {
		c.Limits.MaxArrayLength = defaults.MaxArrayLength
	}
	if c.Limits.MaxObjectProperties <= 0 {
		c.Limits.MaxObjectProperties = defaults.MaxObjectProperties
	}
	if c.Limits.MaxInFlightPerClient <= 0 {
		c.Limits.MaxInFlightPerClient = defaults.MaxInFlightPerClient
	}
	if c.Limits.MaxInFlightPerLane <= 0 {
		c.Limits.MaxInFlightPerLane = defaults.MaxInFlightPerLane
	}
	if c.Limits.DefaultRequestDeadline <= 0 {
		c.Limits.DefaultRequestDeadline = defaults.DefaultRequestDeadline
	}
	if c.Limits.MaxStreamChunkBytes <= 0 {
		c.Limits.MaxStreamChunkBytes = defaults.MaxStreamChunkBytes
	}
	if c.Limits.StreamIdleTimeout <= 0 {
		c.Limits.StreamIdleTimeout = defaults.StreamIdleTimeout
	}
	if c.Limits.MaxResponseStreamBytes <= 0 {
		c.Limits.MaxResponseStreamBytes = defaults.MaxResponseStreamBytes
	}
	return c
}

type RequestContext struct {
	RequestID    string
	ClientID     string
	LaneID       string
	Deadline     *time.Time
	AdmissionErr error
}

type ArtifactListRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type ArtifactListResponse struct {
	SchemaID      string                     `json:"schema_id"`
	SchemaVersion string                     `json:"schema_version"`
	RequestID     string                     `json:"request_id"`
	Artifacts     []artifacts.ArtifactRecord `json:"artifacts"`
}

type ArtifactHeadRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Digest        string `json:"digest"`
}

type ArtifactHeadResponse struct {
	SchemaID      string                   `json:"schema_id"`
	SchemaVersion string                   `json:"schema_version"`
	RequestID     string                   `json:"request_id"`
	Artifact      artifacts.ArtifactRecord `json:"artifact"`
}

type ArtifactPutRequest struct {
	SchemaID              string `json:"schema_id"`
	SchemaVersion         string `json:"schema_version"`
	RequestID             string `json:"request_id"`
	PayloadBase64         string `json:"payload_base64"`
	ContentType           string `json:"content_type"`
	DataClass             string `json:"data_class"`
	ProvenanceReceiptHash string `json:"provenance_receipt_hash"`
	CreatedByRole         string `json:"created_by_role,omitempty"`
	RunID                 string `json:"run_id,omitempty"`
	StepID                string `json:"step_id,omitempty"`
}

type ArtifactPutResponse struct {
	SchemaID      string                      `json:"schema_id"`
	SchemaVersion string                      `json:"schema_version"`
	RequestID     string                      `json:"request_id"`
	Artifact      artifacts.ArtifactReference `json:"artifact"`
}

type ProtocolError struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	Code          string `json:"code"`
	Category      string `json:"category"`
	Retryable     bool   `json:"retryable"`
	Message       string `json:"message"`
}

type ErrorResponse struct {
	SchemaID      string        `json:"schema_id"`
	SchemaVersion string        `json:"schema_version"`
	RequestID     string        `json:"request_id"`
	Error         ProtocolError `json:"error"`
}

func DefaultArtifactListRequest(requestID string) ArtifactListRequest {
	return ArtifactListRequest{
		SchemaID:      "runecode.protocol.v0.BrokerArtifactListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
	}
}

func DefaultArtifactHeadRequest(requestID string, digest string) ArtifactHeadRequest {
	return ArtifactHeadRequest{
		SchemaID:      "runecode.protocol.v0.BrokerArtifactHeadRequest",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Digest:        digest,
	}
}

func DefaultArtifactPutRequest(requestID string, payload []byte, contentType string, dataClass string, provenanceHash string, createdByRole string, runID string, stepID string) ArtifactPutRequest {
	return ArtifactPutRequest{
		SchemaID:              "runecode.protocol.v0.BrokerArtifactPutRequest",
		SchemaVersion:         "0.1.0",
		RequestID:             requestID,
		PayloadBase64:         base64.StdEncoding.EncodeToString(payload),
		ContentType:           contentType,
		DataClass:             dataClass,
		ProvenanceReceiptHash: provenanceHash,
		CreatedByRole:         createdByRole,
		RunID:                 runID,
		StepID:                stepID,
	}
}

func defaultArtifactListResponse(requestID string, artifactsList []artifacts.ArtifactRecord) ArtifactListResponse {
	return ArtifactListResponse{
		SchemaID:      "runecode.protocol.v0.BrokerArtifactListResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Artifacts:     artifactsList,
	}
}

func defaultArtifactHeadResponse(requestID string, record artifacts.ArtifactRecord) ArtifactHeadResponse {
	return ArtifactHeadResponse{
		SchemaID:      "runecode.protocol.v0.BrokerArtifactHeadResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Artifact:      record,
	}
}

func defaultArtifactPutResponse(requestID string, ref artifacts.ArtifactReference) ArtifactPutResponse {
	return ArtifactPutResponse{
		SchemaID:      "runecode.protocol.v0.BrokerArtifactPutResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Artifact:      ref,
	}
}

func toErrorResponse(requestID string, code string, category string, retryable bool, message string) ErrorResponse {
	if requestID == "" {
		requestID = defaultRequestIDFallback
	}
	return ErrorResponse{
		SchemaID:      "runecode.protocol.v0.BrokerErrorResponse",
		SchemaVersion: errorResponseSchemaVersion,
		RequestID:     requestID,
		Error: ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: errorEnvelopeSchemaVersion,
			Code:          code,
			Category:      category,
			Retryable:     retryable,
			Message:       message,
		},
	}
}

func validateJSONEnvelope(value any, schemaPath string) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return artifacts.ValidateObjectPayloadAgainstSchema(b, schemaPath)
}

func validateMessageLimits(value any, limits Limits) error {
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	if len(b) > limits.MaxMessageBytes {
		return fmt.Errorf("message size %d exceeds max %d", len(b), limits.MaxMessageBytes)
	}
	decoded := any(nil)
	if err := json.Unmarshal(b, &decoded); err != nil {
		return fmt.Errorf("decode message: %w", err)
	}
	if err := validateStructuralComplexity(decoded, limits, 1); err != nil {
		return err
	}
	return nil
}

func validateStructuralComplexity(value any, limits Limits, depth int) error {
	if depth > limits.MaxStructuralDepth {
		return fmt.Errorf("message depth %d exceeds max %d", depth, limits.MaxStructuralDepth)
	}
	switch typed := value.(type) {
	case map[string]any:
		return validateMapComplexity(typed, limits, depth)
	case []any:
		return validateArrayComplexity(typed, limits, depth)
	}
	return nil
}

func validateMapComplexity(value map[string]any, limits Limits, depth int) error {
	if len(value) > limits.MaxObjectProperties {
		return fmt.Errorf("object property count %d exceeds max %d", len(value), limits.MaxObjectProperties)
	}
	return validateChildrenComplexity(mapValues(value), limits, depth)
}

func validateArrayComplexity(value []any, limits Limits, depth int) error {
	if len(value) > limits.MaxArrayLength {
		return fmt.Errorf("array length %d exceeds max %d", len(value), limits.MaxArrayLength)
	}
	return validateChildrenComplexity(value, limits, depth)
}

func validateChildrenComplexity(children []any, limits Limits, depth int) error {
	for _, child := range children {
		if err := validateStructuralComplexity(child, limits, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func mapValues(value map[string]any) []any {
	out := make([]any, 0, len(value))
	for _, child := range value {
		out = append(out, child)
	}
	return out
}

type inFlightGate struct {
	mu             sync.Mutex
	limits         Limits
	perClientCount map[string]int
	perLaneCount   map[string]int
}

var errInFlightLimitExceeded = errors.New("in-flight limit exceeded")

func newInFlightGate(limits Limits) *inFlightGate {
	return &inFlightGate{
		limits:         limits,
		perClientCount: map[string]int{},
		perLaneCount:   map[string]int{},
	}
}

func (g *inFlightGate) acquire(clientID string, laneID string) (func(), error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if clientID == "" {
		clientID = "default-client"
	}
	if laneID == "" {
		laneID = "default-lane"
	}
	if g.perClientCount[clientID] >= g.limits.MaxInFlightPerClient {
		return nil, fmt.Errorf("%w: client %q has %d active, max %d", errInFlightLimitExceeded, clientID, g.perClientCount[clientID], g.limits.MaxInFlightPerClient)
	}
	if g.perLaneCount[laneID] >= g.limits.MaxInFlightPerLane {
		return nil, fmt.Errorf("%w: lane %q has %d active, max %d", errInFlightLimitExceeded, laneID, g.perLaneCount[laneID], g.limits.MaxInFlightPerLane)
	}
	g.perClientCount[clientID]++
	g.perLaneCount[laneID]++
	released := false
	return func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		if released {
			return
		}
		released = true
		g.perClientCount[clientID]--
		if g.perClientCount[clientID] <= 0 {
			delete(g.perClientCount, clientID)
		}
		g.perLaneCount[laneID]--
		if g.perLaneCount[laneID] <= 0 {
			delete(g.perLaneCount, laneID)
		}
	}, nil
}

func withDefaultDeadline(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := parent.Deadline(); ok {
		return parent, func() {}
	}
	return context.WithTimeout(parent, timeout)
}

func withRequestDeadline(parent context.Context, meta RequestContext, fallback time.Duration) (context.Context, context.CancelFunc) {
	if meta.Deadline != nil {
		return context.WithDeadline(parent, *meta.Deadline)
	}
	return withDefaultDeadline(parent, fallback)
}

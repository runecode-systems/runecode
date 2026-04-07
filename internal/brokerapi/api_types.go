package brokerapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

type ArtifactListRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
}

type ArtifactListResponse struct {
	SchemaID      string                        `json:"schema_id"`
	SchemaVersion string                        `json:"schema_version"`
	RequestID     string                        `json:"request_id"`
	Artifacts     []artifacts.ArtifactReference `json:"artifacts"`
}

type ArtifactHeadRequest struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	RequestID     string `json:"request_id"`
	Digest        string `json:"digest"`
}

type ArtifactHeadResponse struct {
	SchemaID      string                      `json:"schema_id"`
	SchemaVersion string                      `json:"schema_version"`
	RequestID     string                      `json:"request_id"`
	Artifact      artifacts.ArtifactReference `json:"artifact"`
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
	refs := make([]artifacts.ArtifactReference, 0, len(artifactsList))
	for _, rec := range artifactsList {
		refs = append(refs, rec.Reference)
	}
	return ArtifactListResponse{
		SchemaID:      "runecode.protocol.v0.BrokerArtifactListResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Artifacts:     refs,
	}
}

func defaultArtifactHeadResponse(requestID string, record artifacts.ArtifactRecord) ArtifactHeadResponse {
	return ArtifactHeadResponse{
		SchemaID:      "runecode.protocol.v0.BrokerArtifactHeadResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Artifact:      record.Reference,
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

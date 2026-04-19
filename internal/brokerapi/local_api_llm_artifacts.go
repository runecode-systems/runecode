package brokerapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type llmRequestInputArtifactsEnvelope struct {
	InputArtifacts []llmRequestInputArtifact `json:"input_artifacts"`
}

type llmRequestInputArtifact struct {
	Digest                trustpolicy.Digest  `json:"digest"`
	SizeBytes             int64               `json:"size_bytes"`
	ContentType           string              `json:"content_type"`
	DataClass             artifacts.DataClass `json:"data_class"`
	ProvenanceReceiptHash trustpolicy.Digest  `json:"provenance_receipt_hash"`
}

func (s *Service) storeLLMOutputArtifact(requestID, runID, text string) (artifacts.ArtifactReference, *ErrorResponse) {
	ref, err := s.Put(artifacts.PutRequest{Payload: []byte(text), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: "sha256:" + strings.Repeat("d", 64), CreatedByRole: "model-gateway", TrustedSource: false, RunID: runID, StepID: "llm-output"})
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return artifacts.ArtifactReference{}, &errOut
	}
	return ref, nil
}

func (s *Service) buildCanonicalLLMResponseFromOutput(requestID string, requestHash trustpolicy.Digest, outputRef artifacts.ArtifactReference) (map[string]any, trustpolicy.Digest, *ErrorResponse) {
	responseObj, err := llmResponseObject(requestHash, outputRef)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return nil, trustpolicy.Digest{}, &errOut
	}
	if err := validateJSONEnvelope(responseObj, llmResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return nil, trustpolicy.Digest{}, &errOut
	}
	responseDigest, err := canonicalDigestForValue(responseObj)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return nil, trustpolicy.Digest{}, &errOut
	}
	return responseObj, responseDigest, nil
}

func decodeLLMInputArtifactRefs(llmReq any) ([]artifacts.ArtifactReference, error) {
	raw, err := json.Marshal(llmReq)
	if err != nil {
		return nil, fmt.Errorf("decode llm_request input_artifacts failed: %w", err)
	}
	envelope := llmRequestInputArtifactsEnvelope{}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode llm_request input_artifacts failed: %w", err)
	}
	refs := make([]artifacts.ArtifactReference, 0, len(envelope.InputArtifacts))
	for i := range envelope.InputArtifacts {
		ref, err := envelope.InputArtifacts[i].artifactReference()
		if err != nil {
			return nil, fmt.Errorf("input_artifacts[%d]: %w", i, err)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func (a llmRequestInputArtifact) artifactReference() (artifacts.ArtifactReference, error) {
	digestIdentity, err := a.Digest.Identity()
	if err != nil {
		return artifacts.ArtifactReference{}, fmt.Errorf("digest invalid: %w", err)
	}
	provenanceIdentity, err := a.ProvenanceReceiptHash.Identity()
	if err != nil {
		return artifacts.ArtifactReference{}, fmt.Errorf("provenance_receipt_hash invalid: %w", err)
	}
	return artifacts.ArtifactReference{
		Digest:                digestIdentity,
		SizeBytes:             a.SizeBytes,
		ContentType:           strings.TrimSpace(a.ContentType),
		DataClass:             a.DataClass,
		ProvenanceReceiptHash: provenanceIdentity,
	}, nil
}

func (s *Service) bindLLMRequestToArtifacts(requestID, runID string, expectedDigest *trustpolicy.Digest, llmReq any) (llmExecutionBinding, artifacts.ArtifactReference, *ErrorResponse) {
	runID, reqDigest, errResp := s.resolveLLMRequestArtifact(requestID, runID, expectedDigest, llmReq)
	if errResp != nil {
		return llmExecutionBinding{}, artifacts.ArtifactReference{}, errResp
	}
	inputRefs, decodeErr := decodeLLMInputArtifactRefs(llmReq)
	if decodeErr != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, decodeErr.Error())
		return llmExecutionBinding{}, artifacts.ArtifactReference{}, &errOut
	}
	if len(inputRefs) == 0 {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "llm_request input_artifacts must be non-empty")
		return llmExecutionBinding{}, artifacts.ArtifactReference{}, &errOut
	}
	primaryInputRecord, errResp := s.ensureInputArtifactsExist(requestID, runID, inputRefs)
	if errResp != nil {
		return llmExecutionBinding{}, artifacts.ArtifactReference{}, errResp
	}
	binding := llmExecutionBinding{RequestDigest: reqDigest, RequestHash: reqDigest}
	profile, err := s.providerProfileForLLMRequest(llmReq)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return llmExecutionBinding{}, artifacts.ArtifactReference{}, &errOut
	}
	binding.ProviderID = profile.ProviderProfileID
	binding.ProviderFamily = profile.ProviderFamily
	binding.AdapterKind = profile.AdapterKind
	return binding, primaryInputRecord.Reference, nil
}

func (s *Service) ensureInputArtifactsExist(requestID, runID string, refs []artifacts.ArtifactReference) (artifacts.ArtifactRecord, *ErrorResponse) {
	var primary artifacts.ArtifactRecord
	for _, ref := range refs {
		record, err := s.Head(trimArtifactRefDigest(ref))
		if err != nil {
			if errors.Is(err, artifacts.ErrArtifactNotFound) {
				errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "llm_request input_artifact digest must exist")
				return artifacts.ArtifactRecord{}, &errOut
			}
			errOut := s.errorFromStore(requestID, err)
			return artifacts.ArtifactRecord{}, &errOut
		}
		if strings.TrimSpace(record.RunID) != runID {
			errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "llm_request input_artifact run binding mismatch")
			return artifacts.ArtifactRecord{}, &errOut
		}
		if primary.Reference.Digest == "" {
			primary = record
		}
	}
	return primary, nil
}

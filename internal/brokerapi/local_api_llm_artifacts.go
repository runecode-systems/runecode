package brokerapi

import (
	"encoding/json"
	"fmt"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func llmResponseObject(requestHash trustpolicy.Digest, output artifacts.ArtifactReference) (map[string]any, error) {
	digest, err := digestFromIdentity(output.Digest)
	if err != nil {
		return nil, fmt.Errorf("llm response output digest invalid: %w", err)
	}
	provenanceDigest, err := digestFromIdentity(output.ProvenanceReceiptHash)
	if err != nil {
		return nil, fmt.Errorf("llm response output provenance digest invalid: %w", err)
	}
	return map[string]any{
		"schema_id":            "runecode.protocol.v0.LLMResponse",
		"schema_version":       "0.3.0",
		"request_hash":         requestHash,
		"output_trust_posture": "untrusted_proposal",
		"output_artifacts": []any{
			map[string]any{
				"schema_id":               "runecode.protocol.v0.ArtifactReference",
				"schema_version":          "0.3.0",
				"digest":                  digest,
				"size_bytes":              output.SizeBytes,
				"content_type":            output.ContentType,
				"data_class":              string(output.DataClass),
				"provenance_receipt_hash": provenanceDigest,
			},
		},
	}, nil
}

func decodeLLMInputArtifactRefs(llmReq any) ([]artifacts.ArtifactReference, error) {
	b, err := json.Marshal(llmReq)
	if err != nil {
		return nil, err
	}
	decoded := struct {
		InputArtifacts []struct {
			Digest trustpolicy.Digest `json:"digest"`
		} `json:"input_artifacts"`
	}{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		return nil, err
	}
	refs := make([]artifacts.ArtifactReference, 0, len(decoded.InputArtifacts))
	for _, item := range decoded.InputArtifacts {
		identity, err := item.Digest.Identity()
		if err != nil {
			return nil, err
		}
		refs = append(refs, artifacts.ArtifactReference{Digest: identity})
	}
	return refs, nil
}

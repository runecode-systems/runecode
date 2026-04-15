package brokerapi

import (
	"encoding/json"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func llmResponseObject(requestHash trustpolicy.Digest, output artifacts.ArtifactReference) map[string]any {
	return map[string]any{
		"schema_id":            "runecode.protocol.v0.LLMResponse",
		"schema_version":       "0.3.0",
		"request_hash":         requestHash,
		"output_trust_posture": "untrusted_proposal",
		"output_artifacts": []any{
			map[string]any{
				"schema_id":               "runecode.protocol.v0.ArtifactReference",
				"schema_version":          "0.3.0",
				"digest":                  digestFromIdentityOrPanic(output.Digest),
				"size_bytes":              output.SizeBytes,
				"content_type":            output.ContentType,
				"data_class":              string(output.DataClass),
				"provenance_receipt_hash": digestFromIdentityOrPanic(output.ProvenanceReceiptHash),
			},
		},
	}
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

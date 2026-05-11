package brokerapi

import (
	"fmt"

	"github.com/runecode-systems/runecode/internal/artifacts"
)

func (s *Service) persistSessionExecutionPromptArtifact(runID string, authority sessionExecutionPlanAuthority, payload []byte) (artifacts.ArtifactReference, error) {
	stepID, _, err := sessionExecutionPromptArtifactBinding(authority)
	if err != nil {
		return artifacts.ArtifactReference{}, err
	}
	ref, err := s.Put(artifacts.PutRequest{
		Payload:               payload,
		ContentType:           "text/plain; charset=utf-8",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: artifacts.DigestBytes(payload),
		CreatedByRole:         "brokerapi",
		TrustedSource:         true,
		RunID:                 runID,
		StepID:                stepID,
	})
	if err != nil {
		return artifacts.ArtifactReference{}, fmt.Errorf("persist session execution source prompt: %w", err)
	}
	return ref, nil
}

func (s *Service) persistSessionExecutionDraftTextArtifact(runID string, authority sessionExecutionPlanAuthority, payload []byte) (artifacts.ArtifactReference, error) {
	stepID, _, err := sessionExecutionDraftTextArtifactBinding(authority)
	if err != nil {
		return artifacts.ArtifactReference{}, err
	}
	ref, err := s.Put(artifacts.PutRequest{
		Payload:               payload,
		ContentType:           "text/markdown; charset=utf-8",
		DataClass:             artifacts.DataClassSpecText,
		ProvenanceReceiptHash: artifacts.DigestBytes(payload),
		CreatedByRole:         "brokerapi",
		TrustedSource:         true,
		RunID:                 runID,
		StepID:                stepID,
	})
	if err != nil {
		return artifacts.ArtifactReference{}, fmt.Errorf("persist session execution draft text: %w", err)
	}
	return ref, nil
}

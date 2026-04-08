package brokerapi

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) isTrustedVerifierArtifact(record artifacts.ArtifactRecord) (bool, error) {
	events, err := s.ReadAuditEvents()
	if err != nil {
		return false, fmt.Errorf("read audit events: %w", err)
	}
	return artifacts.IsTrustedVerifierArtifact(record, events), nil
}

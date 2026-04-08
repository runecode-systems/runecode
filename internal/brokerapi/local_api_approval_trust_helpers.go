package brokerapi

import (
	"github.com/runecode-ai/runecode/internal/artifacts"
)

func (s *Service) isTrustedVerifierArtifact(record artifacts.ArtifactRecord) bool {
	events, err := s.ReadAuditEvents()
	if err != nil {
		return false
	}
	return artifacts.IsTrustedVerifierArtifact(record, events)
}

package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-systems/runecode/internal/artifacts"
)

func (s *Service) readArtifactPayloadVerified(digest string) ([]byte, error) {
	payload, err := s.readArtifactPayload(digest)
	if err != nil {
		return nil, err
	}
	if artifacts.DigestBytes(payload) != strings.TrimSpace(digest) {
		return nil, fmt.Errorf("artifact payload digest drift for %q", strings.TrimSpace(digest))
	}
	return payload, nil
}

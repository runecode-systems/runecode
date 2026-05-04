package secretsd

import "strings"

func (s *Service) LookupSecretMetadata(secretRef string) (SecretMetadata, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	trimmed := strings.TrimSpace(secretRef)
	if trimmed == "" {
		return SecretMetadata{}, false
	}
	rec, ok := s.state.Secrets[trimmed]
	if !ok {
		return SecretMetadata{}, false
	}
	return SecretMetadata{
		SecretRef:      trimmed,
		SecretID:       rec.SecretID,
		MaterialDigest: rec.MaterialDigest,
		ImportedAt:     rec.ImportedAt,
	}, true
}

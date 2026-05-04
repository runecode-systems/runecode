package secretsd

import (
	"os"
	"strings"
)

func auditAnchorPresenceMode() string {
	mode := strings.TrimSpace(os.Getenv(envAuditAnchorPresenceMode))
	if mode == "" {
		return "os_confirmation"
	}
	return mode
}

func auditAnchorPresenceModeForService(s *Service) string {
	if s != nil {
		switch s.auditAnchorSignerProfile {
		case auditAnchorSignerProfileMetaAudit:
			return "none"
		}
	}
	return auditAnchorPresenceMode()
}

func (s *Service) AuditAnchorPresenceMode() string {
	return auditAnchorPresenceModeForService(s)
}

func (s *Service) SetAuditAnchorPresenceModeForTrustedRuntime(mode string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(mode) == "none" {
		s.auditAnchorSignerProfile = auditAnchorSignerProfileMetaAudit
		return
	}
	s.auditAnchorSignerProfile = auditAnchorSignerProfileDefault
}

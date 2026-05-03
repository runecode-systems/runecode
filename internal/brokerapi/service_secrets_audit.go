package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/secretsd"
)

func (s *Service) bindSecretsLeaseAuditHook() {
	if s == nil || s.secretsSvc == nil {
		return
	}
	s.secretsSvc.SetLeaseAuditHookForTrustedRuntime(func(event secretsd.LeaseAuditEvent) {
		runID := runIDFromLeaseScope(event.Lease.Scope)
		switch strings.TrimSpace(event.Action) {
		case "issued":
			s.persistSecretLeaseReceipt(runID, "secret_lease_issued", event.Lease)
		case "revoked":
			s.persistSecretLeaseReceipt(runID, "secret_lease_revoked", event.Lease)
		}
	})
}

func runIDFromLeaseScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if strings.HasPrefix(scope, "run:") {
		return strings.TrimPrefix(scope, "run:")
	}
	return ""
}

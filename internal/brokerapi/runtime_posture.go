package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

func runtimePostureDegraded(backendKind, isolationAssuranceLevel string) bool {
	if strings.EqualFold(strings.TrimSpace(backendKind), launcherbackend.BackendKindContainer) {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(isolationAssuranceLevel), launcherbackend.IsolationAssuranceDegraded)
}

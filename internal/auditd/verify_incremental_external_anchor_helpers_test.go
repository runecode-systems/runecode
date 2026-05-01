package auditd

import (
	"testing"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func currentVerificationContextForTest(t *testing.T, ledger *Ledger) trustpolicy.AuditVerificationInput {
	t.Helper()
	ledger.mu.Lock()
	_, input, err := ledger.currentVerificationContextLocked()
	ledger.mu.Unlock()
	if err != nil {
		t.Fatalf("currentVerificationContextLocked returned error: %v", err)
	}
	return input
}

func assertExternalAnchorVerificationInputCounts(t *testing.T, input trustpolicy.AuditVerificationInput, wantEvidence, wantSidecars int) {
	t.Helper()
	if len(input.ExternalAnchorEvidence) != wantEvidence {
		t.Fatalf("ExternalAnchorEvidence length=%d, want %d", len(input.ExternalAnchorEvidence), wantEvidence)
	}
	if len(input.ExternalAnchorSidecars) != wantSidecars {
		t.Fatalf("ExternalAnchorSidecars length=%d, want %d", len(input.ExternalAnchorSidecars), wantSidecars)
	}
}

func assertDigestListContains(t *testing.T, got []trustpolicy.Digest, want trustpolicy.Digest) {
	t.Helper()
	wantIdentity := mustDigestIdentity(want)
	for i := range got {
		if mustDigestIdentity(got[i]) == wantIdentity {
			return
		}
	}
	t.Fatalf("digest list missing %q", wantIdentity)
}

func assertDigestListExcludes(t *testing.T, got []trustpolicy.Digest, forbidden trustpolicy.Digest) {
	t.Helper()
	forbiddenIdentity := mustDigestIdentity(forbidden)
	for i := range got {
		if mustDigestIdentity(got[i]) == forbiddenIdentity {
			t.Fatalf("digest list unexpectedly contains %q", forbiddenIdentity)
		}
	}
}

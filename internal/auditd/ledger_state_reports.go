package auditd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (l *Ledger) preservedLastReportDigestLocked() string {
	prevState, err := l.loadState()
	if err != nil {
		return ""
	}
	return prevState.LastVerificationReportDigest
}

func (l *Ledger) recoverLatestVerificationReportDigestLocked() string {
	latestDigest, ok := l.latestVerificationReportDigestFromIndexLocked()
	if ok {
		return latestDigest
	}
	latestDigest = l.discoverLatestVerificationReportDigestLocked()
	if latestDigest != "" {
		return latestDigest
	}
	return l.preservedLastReportDigestLocked()
}

func (l *Ledger) latestVerificationReportDigestFromIndexLocked() (string, bool) {
	index, err := l.ensureDerivedIndexLocked()
	if err != nil {
		return "", false
	}
	digest := strings.TrimSpace(index.LatestVerificationReportDigest)
	if digest == "" {
		return "", false
	}
	if _, err := l.loadVerificationReportByDigestIdentityLocked(digest); err == nil {
		return digest, true
	}
	refreshed, refreshErr := l.refreshDerivedIndexLocked("latest verification report mismatch")
	if refreshErr != nil {
		return "", false
	}
	digest = strings.TrimSpace(refreshed.LatestVerificationReportDigest)
	if digest == "" {
		return "", false
	}
	if _, err := l.loadVerificationReportByDigestIdentityLocked(digest); err != nil {
		return "", false
	}
	return digest, true
}

func (l *Ledger) loadVerificationReportByDigestIdentityLocked(identity string) (trustpolicy.AuditVerificationReportPayload, error) {
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return trustpolicy.AuditVerificationReportPayload{}, err
	}
	path := filepath.Join(l.rootDir, sidecarDirName, verificationReportsDirName, digest.Hash+".json")
	report := trustpolicy.AuditVerificationReportPayload{}
	if err := readJSONFile(path, &report); err != nil {
		return trustpolicy.AuditVerificationReportPayload{}, err
	}
	computedDigest, err := canonicalDigest(report)
	if err != nil {
		return trustpolicy.AuditVerificationReportPayload{}, err
	}
	computedIdentity, _ := computedDigest.Identity()
	if computedIdentity != identity {
		return trustpolicy.AuditVerificationReportPayload{}, fmt.Errorf("verification report sidecar digest mismatch: expected %q computed %q", identity, computedIdentity)
	}
	if err := trustpolicy.ValidateAuditVerificationReportPayload(report); err != nil {
		return trustpolicy.AuditVerificationReportPayload{}, err
	}
	return report, nil
}

type verificationReportCandidate struct {
	digest     string
	verifiedAt time.Time
}

func (l *Ledger) discoverLatestVerificationReportDigestLocked() string {
	digest, err := l.discoverLatestVerificationReportDigestLockedWithError()
	if err != nil {
		return ""
	}
	return digest
}

func (l *Ledger) discoverLatestVerificationReportDigestLockedWithError() (string, error) {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, verificationReportsDirName))
	if err != nil {
		return "", err
	}
	best := verificationReportCandidate{}
	for _, entry := range entries {
		next, ok := l.verificationReportCandidateFromEntry(entry)
		if !ok {
			continue
		}
		if chooseVerificationReportCandidate(best, next) {
			best = next
		}
	}
	return best.digest, nil
}

func chooseVerificationReportCandidate(current, next verificationReportCandidate) bool {
	if current.digest == "" {
		return true
	}
	if next.verifiedAt.After(current.verifiedAt) {
		return true
	}
	return next.verifiedAt.Equal(current.verifiedAt) && next.digest > current.digest
}

func (l *Ledger) verificationReportCandidateFromEntry(entry os.DirEntry) (verificationReportCandidate, bool) {
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
		return verificationReportCandidate{}, false
	}
	report := trustpolicy.AuditVerificationReportPayload{}
	path := filepath.Join(l.rootDir, sidecarDirName, verificationReportsDirName, entry.Name())
	if err := readJSONFile(path, &report); err != nil {
		return verificationReportCandidate{}, false
	}
	verifiedAt, err := time.Parse(time.RFC3339, report.VerifiedAt)
	if err != nil {
		return verificationReportCandidate{}, false
	}
	digest := "sha256:" + strings.TrimSuffix(entry.Name(), ".json")
	return verificationReportCandidate{digest: digest, verifiedAt: verifiedAt}, true
}

package auditd

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

const stateSchemaVersion = 1

type Ledger struct {
	mu      sync.Mutex
	rootDir string
	nowFn   func() time.Time
}

func Open(rootDir string) (*Ledger, error) {
	if strings.TrimSpace(rootDir) == "" {
		return nil, fmt.Errorf("ledger root is required")
	}
	ledger := &Ledger{rootDir: rootDir, nowFn: time.Now}
	if err := ledger.ensureLayout(); err != nil {
		return nil, err
	}
	if _, err := ledger.recoverAndPersistStateLocked(); err != nil {
		return nil, err
	}
	return ledger, nil
}

func (l *Ledger) PersistReceiptEnvelope(envelope trustpolicy.SignedObjectEnvelope) (trustpolicy.Digest, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if envelope.PayloadSchemaID != trustpolicy.AuditReceiptSchemaID {
		return trustpolicy.Digest{}, fmt.Errorf("receipt envelope must use payload_schema_id %q", trustpolicy.AuditReceiptSchemaID)
	}
	return l.persistReceiptEnvelopeLocked(envelope)
}

func (l *Ledger) persistReceiptEnvelopeLocked(envelope trustpolicy.SignedObjectEnvelope) (trustpolicy.Digest, error) {
	receiptDigest, err := l.persistEnvelopeSidecar(receiptsDirName, envelope)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	if err := l.notePersistedReceiptInIncrementalFoundationLocked(receiptDigest, envelope); err != nil {
		return trustpolicy.Digest{}, err
	}
	return receiptDigest, nil
}

func (l *Ledger) persistEnvelopeSidecar(dirName string, envelope trustpolicy.SignedObjectEnvelope) (trustpolicy.Digest, error) {
	digest, err := trustpolicy.ComputeSignedEnvelopeAuditRecordDigest(envelope)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	identity, _ := digest.Identity()
	path := filepath.Join(l.rootDir, sidecarDirName, dirName, strings.TrimPrefix(identity, "sha256:")+".json")
	if err := writeCanonicalJSONFile(path, envelope); err != nil {
		return trustpolicy.Digest{}, err
	}
	return digest, nil
}

func (l *Ledger) PersistVerificationReport(report trustpolicy.AuditVerificationReportPayload) (trustpolicy.Digest, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.persistVerificationReportLocked(report)
}

func (l *Ledger) persistVerificationReportLocked(report trustpolicy.AuditVerificationReportPayload) (trustpolicy.Digest, error) {
	if err := trustpolicy.ValidateAuditVerificationReportPayload(report); err != nil {
		return trustpolicy.Digest{}, err
	}
	digest, err := canonicalDigest(report)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	identity, _ := digest.Identity()
	path := filepath.Join(l.rootDir, sidecarDirName, verificationReportsDirName, strings.TrimPrefix(identity, "sha256:")+".json")
	if err := writeCanonicalJSONFile(path, report); err != nil {
		return trustpolicy.Digest{}, err
	}
	state, err := l.recoverAndPersistStateLocked()
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	state.LastVerificationReportDigest = identity
	if err := l.saveState(state); err != nil {
		return trustpolicy.Digest{}, err
	}
	if err := l.notePersistedVerificationReportInDerivedIndexLocked(digest); err != nil {
		return trustpolicy.Digest{}, err
	}
	return digest, nil
}

func (l *Ledger) LatestVerificationReport() (trustpolicy.AuditVerificationReportPayload, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.latestVerificationReportLocked()
}

func (l *Ledger) latestVerificationReportLocked() (trustpolicy.AuditVerificationReportPayload, error) {
	state, err := l.recoverAndPersistStateLocked()
	if err != nil {
		return trustpolicy.AuditVerificationReportPayload{}, err
	}
	if state.LastVerificationReportDigest == "" {
		return trustpolicy.AuditVerificationReportPayload{}, fmt.Errorf("no verification report persisted")
	}
	return l.loadVerificationReportByDigestIdentityLocked(state.LastVerificationReportDigest)
}

func canonicalEnvelopeAndDigest(envelope trustpolicy.SignedObjectEnvelope) ([]byte, trustpolicy.Digest, error) {
	marshaled, err := json.Marshal(envelope)
	if err != nil {
		return nil, trustpolicy.Digest{}, err
	}
	canonical, err := jsoncanonicalizer.Transform(marshaled)
	if err != nil {
		return nil, trustpolicy.Digest{}, err
	}
	sum := sha256.Sum256(canonical)
	digest := trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}
	return canonical, digest, nil
}

func canonicalDigest(v any) (trustpolicy.Digest, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	sum := sha256.Sum256(canonical)
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, nil
}

func decodeFrameEnvelope(frame trustpolicy.AuditSegmentRecordFrame) (trustpolicy.SignedObjectEnvelope, error) {
	raw, err := base64.StdEncoding.DecodeString(frame.CanonicalSignedEnvelopeBytes)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	envelope := trustpolicy.SignedObjectEnvelope{}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	return envelope, nil
}

package auditd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	externalAnchorSidecarSchemaID      = "runecode.protocol.v0.ExternalAnchorSidecarEvidence"
	externalAnchorSidecarSchemaVersion = "0.1.0"
)

type ExternalAnchorSidecarPayload struct {
	SchemaID      string `json:"schema_id"`
	SchemaVersion string `json:"schema_version"`
	EvidenceKind  string `json:"evidence_kind"`
	Payload       any    `json:"payload"`
}

func (l *Ledger) PersistExternalAnchorSidecar(evidenceKind string, payload any) (trustpolicy.Digest, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := trustpolicy.ValidateExternalAnchorSidecarKind(evidenceKind); err != nil {
		return trustpolicy.Digest{}, err
	}
	wrapped := ExternalAnchorSidecarPayload{
		SchemaID:      externalAnchorSidecarSchemaID,
		SchemaVersion: externalAnchorSidecarSchemaVersion,
		EvidenceKind:  strings.TrimSpace(evidenceKind),
		Payload:       payload,
	}
	digest, err := canonicalDigest(wrapped)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	identity, _ := digest.Identity()
	path := filepath.Join(l.rootDir, sidecarDirName, externalAnchorSidecarsDir, strings.TrimPrefix(identity, "sha256:")+".json")
	if err := writeCanonicalJSONFile(path, wrapped); err != nil {
		return trustpolicy.Digest{}, err
	}
	return digest, nil
}

func (l *Ledger) loadExternalAnchorSidecarDigestByIdentityLocked(identity string) (trustpolicy.Digest, error) {
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return trustpolicy.Digest{}, fmt.Errorf("external anchor sidecar digest identity invalid: %w", err)
	}
	identity, _ = digest.Identity()
	wrapped, err := l.loadExternalAnchorSidecarPayloadByIdentityLocked(identity)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	computed, err := canonicalDigest(wrapped)
	if err != nil {
		return trustpolicy.Digest{}, fmt.Errorf("compute external anchor sidecar digest %s: %w", identity, err)
	}
	if mustDigestIdentity(computed) != identity {
		return trustpolicy.Digest{}, fmt.Errorf("external anchor sidecar digest mismatch for %s", identity)
	}
	return digest, nil
}

func (l *Ledger) loadExternalAnchorSidecarsLocked() ([]trustpolicy.Digest, error) {
	entries, err := os.ReadDir(filepath.Join(l.rootDir, sidecarDirName, externalAnchorSidecarsDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	result := make([]trustpolicy.Digest, 0, len(entries))
	for i := range entries {
		digest, ok, err := l.loadExternalAnchorSidecarDigestForEntryLocked(entries[i])
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		result = append(result, digest)
	}
	return result, nil
}

func (l *Ledger) loadExternalAnchorSidecarDigestForEntryLocked(entry os.DirEntry) (trustpolicy.Digest, bool, error) {
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
		return trustpolicy.Digest{}, false, nil
	}
	identity, ok, err := digestIdentityFromSidecarName(entry.Name())
	if err != nil {
		return trustpolicy.Digest{}, false, err
	}
	if !ok {
		return trustpolicy.Digest{}, false, nil
	}
	digest, err := l.loadExternalAnchorSidecarDigestByIdentityLocked(identity)
	if err != nil {
		return trustpolicy.Digest{}, false, err
	}
	return digest, true, nil
}

func (l *Ledger) loadExternalAnchorSidecarPayloadByIdentityLocked(identity string) (ExternalAnchorSidecarPayload, error) {
	path := filepath.Join(l.rootDir, sidecarDirName, externalAnchorSidecarsDir, strings.TrimPrefix(identity, "sha256:")+".json")
	wrapped := ExternalAnchorSidecarPayload{}
	if err := readJSONFile(path, &wrapped); err != nil {
		return ExternalAnchorSidecarPayload{}, fmt.Errorf("external anchor sidecar missing or unreadable for %s", identity)
	}
	if err := validateExternalAnchorSidecarPayload(wrapped); err != nil {
		return ExternalAnchorSidecarPayload{}, fmt.Errorf("external anchor sidecar invalid for %s: %w", identity, err)
	}
	return wrapped, nil
}

func validateExternalAnchorSidecarPayload(payload ExternalAnchorSidecarPayload) error {
	if strings.TrimSpace(payload.SchemaID) != externalAnchorSidecarSchemaID {
		return fmt.Errorf("schema_id must be %q", externalAnchorSidecarSchemaID)
	}
	if strings.TrimSpace(payload.SchemaVersion) != externalAnchorSidecarSchemaVersion {
		return fmt.Errorf("schema_version must be %q", externalAnchorSidecarSchemaVersion)
	}
	if err := trustpolicy.ValidateExternalAnchorSidecarKind(payload.EvidenceKind); err != nil {
		return err
	}
	if payload.Payload == nil {
		return fmt.Errorf("payload is required")
	}
	if _, err := json.Marshal(payload.Payload); err != nil {
		return fmt.Errorf("payload must be JSON-serializable: %w", err)
	}
	return nil
}

package auditd

import (
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
	path := filepath.Join(l.rootDir, sidecarDirName, externalAnchorEvidenceDir, strings.TrimPrefix(identity, "sha256:")+".json")
	if err := writeCanonicalJSONFile(path, wrapped); err != nil {
		return trustpolicy.Digest{}, err
	}
	return digest, nil
}

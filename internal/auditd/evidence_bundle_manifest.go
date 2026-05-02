package auditd

import (
	"strings"
	"time"
)

const (
	auditEvidenceBundleManifestSchemaID      = "runecode.protocol.v0.AuditEvidenceBundleManifest"
	auditEvidenceBundleManifestSchemaVersion = "0.1.0"
)

func (l *Ledger) BuildEvidenceBundleManifest(req AuditEvidenceBundleManifestRequest) (AuditEvidenceBundleManifest, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buildEvidenceBundleManifestLocked(req)
}

func (l *Ledger) buildEvidenceBundleManifestLocked(req AuditEvidenceBundleManifestRequest) (AuditEvidenceBundleManifest, error) {
	if err := validateEvidenceBundleManifestRequest(req); err != nil {
		return AuditEvidenceBundleManifest{}, err
	}
	profilePolicy := evidenceBundleExportProfilePolicy(strings.TrimSpace(req.ExportProfile))
	data, err := l.evidenceBundleManifestDataLocked(req.Scope, profilePolicy)
	if err != nil {
		return AuditEvidenceBundleManifest{}, err
	}
	requestedRedactions := normalizeEvidenceBundleRedactions(req.Redactions)
	declaredProfileRedactions := evidenceBundleProfileDeclaredRedactions(profilePolicy)
	allRedactions := normalizeEvidenceBundleRedactions(append(declaredProfileRedactions, requestedRedactions...))
	included, userRedactionApplied := applyEvidenceBundleRedactions(data.includedObjects, requestedRedactions)
	disclosurePosture := resolveEvidenceBundleDisclosurePosture(req.DisclosurePosture, profilePolicy, userRedactionApplied, len(allRedactions) > 0)
	manifest := AuditEvidenceBundleManifest{
		SchemaID:          auditEvidenceBundleManifestSchemaID,
		SchemaVersion:     auditEvidenceBundleManifestSchemaVersion,
		BundleID:          evidenceBundleID(l.nowFn()),
		CreatedAt:         l.nowFn().UTC().Format(time.RFC3339),
		CreatedByTool:     normalizeEvidenceBundleToolIdentity(req.CreatedByTool),
		ExportProfile:     strings.TrimSpace(req.ExportProfile),
		Scope:             normalizeEvidenceBundleScope(req.Scope),
		InstanceIdentity:  strings.TrimSpace(data.instanceIdentity),
		IncludedObjects:   included,
		RootDigests:       data.rootDigests,
		SealReferences:    data.sealRefs,
		VerifierIdentity:  l.evidenceBundleVerifierIdentityLocked(),
		TrustRootDigests:  l.evidenceBundleTrustRootDigestsLocked(),
		DisclosurePosture: disclosurePosture,
		Redactions:        allRedactions,
	}
	return manifest, nil
}

func (l *Ledger) evidenceBundleInstanceIdentityLocked() (string, error) {
	_, _, _, _, instanceIdentityDigests, err := l.externalAnchorDerivedEvidenceIdentitiesLocked()
	if err != nil {
		return "", err
	}
	normalized := normalizeIdentityList(instanceIdentityDigests)
	if len(normalized) == 0 {
		return "", nil
	}
	return normalized[0], nil
}

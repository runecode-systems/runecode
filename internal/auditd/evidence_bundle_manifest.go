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
	profilePolicy, err := evidenceBundleExportProfilePolicy(strings.TrimSpace(req.ExportProfile))
	if err != nil {
		return AuditEvidenceBundleManifest{}, err
	}
	data, err := l.evidenceBundleManifestDataLocked(req.Scope, profilePolicy)
	if err != nil {
		return AuditEvidenceBundleManifest{}, err
	}
	requestedRedactions, allRedactions := evidenceBundleRedactionSets(req.Redactions, profilePolicy)
	included, disclosurePosture := buildManifestDisclosureState(data.includedObjects, requestedRedactions, allRedactions, req.DisclosurePosture, profilePolicy)
	manifest := AuditEvidenceBundleManifest{
		SchemaID:                     auditEvidenceBundleManifestSchemaID,
		SchemaVersion:                auditEvidenceBundleManifestSchemaVersion,
		BundleID:                     evidenceBundleID(l.nowFn()),
		CreatedAt:                    l.nowFn().UTC().Format(time.RFC3339),
		CreatedByTool:                normalizeEvidenceBundleToolIdentity(req.CreatedByTool),
		ExportProfile:                strings.TrimSpace(req.ExportProfile),
		Scope:                        normalizeEvidenceBundleScope(req.Scope),
		RepositoryIdentityDigest:     strings.TrimSpace(req.IdentityContext.RepositoryIdentityDigest),
		ProductInstanceID:            strings.TrimSpace(req.IdentityContext.ProductInstanceID),
		LedgerIdentity:               strings.TrimSpace(data.identityContext.LedgerIdentity),
		ControlPlane:                 data.controlPlane,
		ProjectContextIdentityDigest: strings.TrimSpace(data.projectContextIdentity),
		IncludedObjects:              included,
		RootDigests:                  data.rootDigests,
		SealReferences:               data.sealRefs,
		VerifierIdentity:             l.evidenceBundleVerifierIdentityLocked(),
		TrustRootDigests:             l.evidenceBundleTrustRootDigestsLocked(),
		DisclosurePosture:            disclosurePosture,
		Redactions:                   allRedactions,
	}
	if err := validateManifestControlPlane(manifest.ControlPlane); err != nil {
		return AuditEvidenceBundleManifest{}, err
	}
	return manifest, nil
}

func evidenceBundleRedactionSets(requested []AuditEvidenceBundleRedaction, profilePolicy evidenceBundleProfilePolicy) ([]AuditEvidenceBundleRedaction, []AuditEvidenceBundleRedaction) {
	requestedRedactions := normalizeEvidenceBundleRedactions(requested)
	declaredProfileRedactions := evidenceBundleProfileDeclaredRedactions(profilePolicy)
	allRedactions := normalizeEvidenceBundleRedactions(append(declaredProfileRedactions, requestedRedactions...))
	return requestedRedactions, allRedactions
}

func buildManifestDisclosureState(includedObjects []AuditEvidenceBundleIncludedObject, requestedRedactions, allRedactions []AuditEvidenceBundleRedaction, requestedPosture AuditEvidenceBundleDisclosurePosture, profilePolicy evidenceBundleProfilePolicy) ([]AuditEvidenceBundleIncludedObject, AuditEvidenceBundleDisclosurePosture) {
	included, userRedactionApplied := applyEvidenceBundleRedactions(includedObjects, requestedRedactions)
	disclosurePosture := resolveEvidenceBundleDisclosurePosture(requestedPosture, profilePolicy, userRedactionApplied, len(allRedactions) > 0)
	return included, disclosurePosture
}

func validateManifestControlPlane(controlPlane *AuditEvidenceBundleControlProvenance) error {
	if controlPlane == nil {
		return nil
	}
	return validateEvidenceBundleControlProvenance(*controlPlane)
}

func (l *Ledger) evidenceBundleProjectContextIdentityLocked() (string, error) {
	_, _, _, _, _, _, _, projectContextDigests, _, _, err := l.externalAnchorDerivedEvidenceIdentitiesLocked()
	if err != nil {
		return "", err
	}
	normalized := normalizeIdentityList(projectContextDigests)
	if len(normalized) == 0 {
		return "", nil
	}
	return normalized[0], nil
}

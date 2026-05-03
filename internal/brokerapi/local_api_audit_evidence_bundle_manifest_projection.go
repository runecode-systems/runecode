package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type projectedAuditEvidenceBundleManifestParts struct {
	rootDigests      []trustpolicy.Digest
	trustRootDigests []trustpolicy.Digest
	includedObjects  []AuditEvidenceBundleIncludedObject
	sealReferences   []AuditEvidenceBundleSealReference
	scope            AuditEvidenceBundleScope
	instanceIdentity *trustpolicy.Digest
}

func projectAuditEvidenceBundleManifest(manifest auditd.AuditEvidenceBundleManifest) (AuditEvidenceBundleManifest, error) {
	parts, err := projectAuditEvidenceBundleManifestPartsFromTrusted(manifest)
	if err != nil {
		return AuditEvidenceBundleManifest{}, err
	}
	toolIdentity, err := projectAuditEvidenceBundleToolIdentity(manifest.CreatedByTool)
	if err != nil {
		return AuditEvidenceBundleManifest{}, err
	}
	controlPlane, err := projectAuditEvidenceBundleControlProvenance(manifest.ControlPlane)
	if err != nil {
		return AuditEvidenceBundleManifest{}, err
	}
	return AuditEvidenceBundleManifest{
		SchemaID:          "runecode.protocol.v0.AuditEvidenceBundleManifest",
		SchemaVersion:     "0.1.0",
		BundleID:          manifest.BundleID,
		CreatedAt:         manifest.CreatedAt,
		CreatedByTool:     toolIdentity,
		ExportProfile:     manifest.ExportProfile,
		Scope:             parts.scope,
		ControlPlane:      controlPlane,
		InstanceIdentity:  parts.instanceIdentity,
		IncludedObjects:   parts.includedObjects,
		RootDigests:       parts.rootDigests,
		SealReferences:    parts.sealReferences,
		VerifierIdentity:  projectAuditEvidenceBundleVerifierIdentity(manifest.VerifierIdentity),
		TrustRootDigests:  parts.trustRootDigests,
		DisclosurePosture: projectAuditEvidenceBundleDisclosurePosture(manifest.DisclosurePosture),
		Redactions:        projectAuditEvidenceBundleRedactions(manifest.Redactions),
	}, nil
}

func projectAuditEvidenceBundleManifestPartsFromTrusted(manifest auditd.AuditEvidenceBundleManifest) (projectedAuditEvidenceBundleManifestParts, error) {
	rootDigests, err := projectAuditSnapshotDigests(manifest.RootDigests)
	if err != nil {
		return projectedAuditEvidenceBundleManifestParts{}, err
	}
	trustRootDigests, err := projectAuditSnapshotDigests(manifest.TrustRootDigests)
	if err != nil {
		return projectedAuditEvidenceBundleManifestParts{}, err
	}
	includedObjects, err := projectAuditEvidenceBundleIncludedObjects(manifest.IncludedObjects)
	if err != nil {
		return projectedAuditEvidenceBundleManifestParts{}, err
	}
	sealReferences, err := projectAuditEvidenceBundleSealReferences(manifest.SealReferences)
	if err != nil {
		return projectedAuditEvidenceBundleManifestParts{}, err
	}
	scope, err := projectAuditEvidenceBundleScope(manifest.Scope)
	if err != nil {
		return projectedAuditEvidenceBundleManifestParts{}, err
	}
	instanceIdentity, err := optionalDigestFromAuditIdentity(manifest.InstanceIdentity)
	if err != nil {
		return projectedAuditEvidenceBundleManifestParts{}, err
	}
	return projectedAuditEvidenceBundleManifestParts{
		rootDigests:      rootDigests,
		trustRootDigests: trustRootDigests,
		includedObjects:  includedObjects,
		sealReferences:   sealReferences,
		scope:            scope,
		instanceIdentity: instanceIdentity,
	}, nil
}

func projectAuditEvidenceBundleIncludedObjects(values []auditd.AuditEvidenceBundleIncludedObject) ([]AuditEvidenceBundleIncludedObject, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]AuditEvidenceBundleIncludedObject, 0, len(values))
	for i := range values {
		d, err := digestFromIdentity(values[i].Digest)
		if err != nil {
			return nil, err
		}
		out = append(out, AuditEvidenceBundleIncludedObject{ObjectFamily: values[i].ObjectFamily, Digest: d, Path: values[i].Path, ByteLength: values[i].ByteLength})
	}
	return out, nil
}

func projectAuditEvidenceBundleSealReferences(values []auditd.AuditEvidenceBundleSealReference) ([]AuditEvidenceBundleSealReference, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]AuditEvidenceBundleSealReference, 0, len(values))
	for i := range values {
		sealDigest, previous, err := projectAuditEvidenceBundleSealDigests(values[i])
		if err != nil {
			return nil, err
		}
		out = append(out, AuditEvidenceBundleSealReference{SegmentID: values[i].SegmentID, SealDigest: sealDigest, SealChainIndex: values[i].SealChainIndex, PreviousSealDigest: previous})
	}
	return out, nil
}

func projectAuditEvidenceBundleSealDigests(value auditd.AuditEvidenceBundleSealReference) (trustpolicy.Digest, *trustpolicy.Digest, error) {
	sealDigest, err := digestFromIdentity(value.SealDigest)
	if err != nil {
		return trustpolicy.Digest{}, nil, err
	}
	var previous *trustpolicy.Digest
	if strings.TrimSpace(value.PreviousSealDigest) != "" {
		d, err := digestFromIdentity(value.PreviousSealDigest)
		if err != nil {
			return trustpolicy.Digest{}, nil, err
		}
		previous = &d
	}
	return sealDigest, previous, nil
}

func projectAuditEvidenceBundleScope(scope auditd.AuditEvidenceBundleScope) (AuditEvidenceBundleScope, error) {
	artifactDigests, err := projectAuditSnapshotDigests(scope.ArtifactDigests)
	if err != nil {
		return AuditEvidenceBundleScope{}, err
	}
	return AuditEvidenceBundleScope{ScopeKind: scope.ScopeKind, RunID: scope.RunID, IncidentID: scope.IncidentID, ArtifactDigests: artifactDigests}, nil
}

func projectAuditEvidenceBundleToolIdentity(identity auditd.AuditEvidenceBundleToolIdentity) (AuditEvidenceBundleToolIdentity, error) {
	var protocolBundleManifestHash *trustpolicy.Digest
	if hash := strings.TrimSpace(identity.ProtocolBundleManifestHash); hash != "" {
		d, err := digestFromIdentity(hash)
		if err != nil {
			return AuditEvidenceBundleToolIdentity{}, err
		}
		protocolBundleManifestHash = &d
	}
	return AuditEvidenceBundleToolIdentity{
		ToolName:                   identity.ToolName,
		ToolVersion:                identity.ToolVersion,
		BuildRevision:              identity.BuildRevision,
		ProtocolBundleManifestHash: protocolBundleManifestHash,
	}, nil
}

func projectAuditEvidenceBundleControlProvenance(control *auditd.AuditEvidenceBundleControlProvenance) (*AuditEvidenceBundleControlProvenance, error) {
	if control == nil {
		return nil, nil
	}
	workflowDefinitionHash, err := optionalDigestFromAuditIdentity(control.WorkflowDefinitionHash)
	if err != nil {
		return nil, err
	}
	toolManifestDigest, err := optionalDigestFromAuditIdentity(control.ToolManifestDigest)
	if err != nil {
		return nil, err
	}
	promptTemplateDigest, err := optionalDigestFromAuditIdentity(control.PromptTemplateDigest)
	if err != nil {
		return nil, err
	}
	protocolBundleHash, err := optionalDigestFromAuditIdentity(control.ProtocolBundleHash)
	if err != nil {
		return nil, err
	}
	verifierImplDigest, err := optionalDigestFromAuditIdentity(control.VerifierImplDigest)
	if err != nil {
		return nil, err
	}
	trustPolicyDigest, err := optionalDigestFromAuditIdentity(control.TrustPolicyDigest)
	if err != nil {
		return nil, err
	}
	if workflowDefinitionHash == nil && toolManifestDigest == nil && promptTemplateDigest == nil && protocolBundleHash == nil && verifierImplDigest == nil && trustPolicyDigest == nil {
		return nil, nil
	}
	return &AuditEvidenceBundleControlProvenance{
		WorkflowDefinitionHash: workflowDefinitionHash,
		ToolManifestDigest:     toolManifestDigest,
		PromptTemplateDigest:   promptTemplateDigest,
		ProtocolBundleHash:     protocolBundleHash,
		VerifierImplDigest:     verifierImplDigest,
		TrustPolicyDigest:      trustPolicyDigest,
	}, nil
}

func projectAuditEvidenceBundleVerifierIdentity(identity auditd.AuditEvidenceBundleVerifierIdentity) AuditEvidenceBundleVerifierIdentity {
	return AuditEvidenceBundleVerifierIdentity{
		KeyID:          identity.KeyID,
		KeyIDValue:     identity.KeyIDValue,
		LogicalPurpose: identity.LogicalPurpose,
		LogicalScope:   identity.LogicalScope,
	}
}

func projectAuditEvidenceBundleDisclosurePosture(posture auditd.AuditEvidenceBundleDisclosurePosture) AuditEvidenceBundleDisclosurePosture {
	return AuditEvidenceBundleDisclosurePosture{Posture: posture.Posture, SelectiveDisclosureApplied: posture.SelectiveDisclosureApplied}
}

func projectAuditEvidenceBundleRedactions(redactions []auditd.AuditEvidenceBundleRedaction) []AuditEvidenceBundleRedaction {
	if len(redactions) == 0 {
		return nil
	}
	out := make([]AuditEvidenceBundleRedaction, 0, len(redactions))
	for i := range redactions {
		out = append(out, AuditEvidenceBundleRedaction{Path: redactions[i].Path, ReasonCode: redactions[i].ReasonCode})
	}
	return out
}

func projectAuditEvidenceBundleToolIdentityToTrusted(identity AuditEvidenceBundleToolIdentity) (auditd.AuditEvidenceBundleToolIdentity, error) {
	protocolBundleManifestHash := ""
	if identity.ProtocolBundleManifestHash != nil {
		id, err := identity.ProtocolBundleManifestHash.Identity()
		if err != nil {
			return auditd.AuditEvidenceBundleToolIdentity{}, err
		}
		protocolBundleManifestHash = id
	}
	return auditd.AuditEvidenceBundleToolIdentity{
		ToolName:                   identity.ToolName,
		ToolVersion:                identity.ToolVersion,
		BuildRevision:              identity.BuildRevision,
		ProtocolBundleManifestHash: protocolBundleManifestHash,
	}, nil
}

func projectAuditEvidenceBundleScopeToTrusted(scope AuditEvidenceBundleScope) (auditd.AuditEvidenceBundleScope, error) {
	artifacts := make([]string, 0, len(scope.ArtifactDigests))
	for i := range scope.ArtifactDigests {
		identity, err := scope.ArtifactDigests[i].Identity()
		if err != nil {
			return auditd.AuditEvidenceBundleScope{}, err
		}
		artifacts = append(artifacts, identity)
	}
	return auditd.AuditEvidenceBundleScope{ScopeKind: scope.ScopeKind, RunID: scope.RunID, IncidentID: scope.IncidentID, ArtifactDigests: artifacts}, nil
}

func projectAuditEvidenceBundleDisclosurePostureToTrusted(posture AuditEvidenceBundleDisclosurePosture) auditd.AuditEvidenceBundleDisclosurePosture {
	return auditd.AuditEvidenceBundleDisclosurePosture{Posture: posture.Posture, SelectiveDisclosureApplied: posture.SelectiveDisclosureApplied}
}

func projectAuditEvidenceBundleRedactionsToTrusted(redactions []AuditEvidenceBundleRedaction) []auditd.AuditEvidenceBundleRedaction {
	if len(redactions) == 0 {
		return nil
	}
	out := make([]auditd.AuditEvidenceBundleRedaction, 0, len(redactions))
	for i := range redactions {
		out = append(out, auditd.AuditEvidenceBundleRedaction{Path: redactions[i].Path, ReasonCode: redactions[i].ReasonCode})
	}
	return out
}

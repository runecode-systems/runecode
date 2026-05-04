package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type projectedAuditEvidenceBundleManifestParts struct {
	rootDigests                  []trustpolicy.Digest
	trustRootDigests             []trustpolicy.Digest
	includedObjects              []AuditEvidenceBundleIncludedObject
	sealReferences               []AuditEvidenceBundleSealReference
	scope                        AuditEvidenceBundleScope
	repositoryIdentityDigest     *trustpolicy.Digest
	projectContextIdentityDigest *trustpolicy.Digest
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
		SchemaID:                     "runecode.protocol.v0.AuditEvidenceBundleManifest",
		SchemaVersion:                "0.1.0",
		BundleID:                     manifest.BundleID,
		CreatedAt:                    manifest.CreatedAt,
		CreatedByTool:                toolIdentity,
		ExportProfile:                manifest.ExportProfile,
		Scope:                        parts.scope,
		RepositoryIdentityDigest:     parts.repositoryIdentityDigest,
		ProductInstanceID:            manifest.ProductInstanceID,
		LedgerIdentity:               manifest.LedgerIdentity,
		ControlPlane:                 controlPlane,
		ProjectContextIdentityDigest: parts.projectContextIdentityDigest,
		IncludedObjects:              parts.includedObjects,
		RootDigests:                  parts.rootDigests,
		SealReferences:               parts.sealReferences,
		VerifierIdentity:             projectAuditEvidenceBundleVerifierIdentity(manifest.VerifierIdentity),
		TrustRootDigests:             parts.trustRootDigests,
		DisclosurePosture:            projectAuditEvidenceBundleDisclosurePosture(manifest.DisclosurePosture),
		Redactions:                   projectAuditEvidenceBundleRedactions(manifest.Redactions),
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
	repositoryIdentityDigest, err := optionalDigestFromAuditIdentity(manifest.RepositoryIdentityDigest)
	if err != nil {
		return projectedAuditEvidenceBundleManifestParts{}, err
	}
	projectContextIdentityDigest, err := optionalDigestFromAuditIdentity(manifest.ProjectContextIdentityDigest)
	if err != nil {
		return projectedAuditEvidenceBundleManifestParts{}, err
	}
	return projectedAuditEvidenceBundleManifestParts{
		rootDigests:                  rootDigests,
		trustRootDigests:             trustRootDigests,
		includedObjects:              includedObjects,
		sealReferences:               sealReferences,
		scope:                        scope,
		repositoryIdentityDigest:     repositoryIdentityDigest,
		projectContextIdentityDigest: projectContextIdentityDigest,
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

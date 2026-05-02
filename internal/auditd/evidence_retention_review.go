package auditd

import "strings"

func (l *Ledger) BuildEvidenceRetentionReview(scope AuditEvidenceBundleScope) (AuditEvidenceSnapshot, AuditEvidenceBundleManifest, AuditEvidenceSnapshotCompletenessReview, error) {
	snapshot, err := l.EvidenceSnapshot()
	if err != nil {
		return AuditEvidenceSnapshot{}, AuditEvidenceBundleManifest{}, AuditEvidenceSnapshotCompletenessReview{}, err
	}
	manifest, err := l.BuildEvidenceBundleManifest(AuditEvidenceBundleManifestRequest{
		Scope:             scope,
		ExportProfile:     "operator_private_full",
		CreatedByTool:     AuditEvidenceBundleToolIdentity{ToolName: "runecode-auditd", ToolVersion: "0.0.0-dev"},
		DisclosurePosture: AuditEvidenceBundleDisclosurePosture{Posture: "operator_private", SelectiveDisclosureApplied: false},
	})
	if err != nil {
		return AuditEvidenceSnapshot{}, AuditEvidenceBundleManifest{}, AuditEvidenceSnapshotCompletenessReview{}, err
	}
	return snapshot, manifest, EvaluateEvidenceRetentionCompleteness(snapshot, manifest), nil
}

func EvaluateEvidenceRetentionCompleteness(snapshot AuditEvidenceSnapshot, manifest AuditEvidenceBundleManifest) AuditEvidenceSnapshotCompletenessReview {
	review := AuditEvidenceSnapshotCompletenessReview{}
	included := includedIdentitySet(manifest.IncludedObjects)
	track := evidenceCompletenessTracker{review: &review, included: included, redactions: manifest.Redactions}
	for _, family := range evidenceRetentionDigestFamilies(snapshot) {
		track.digestFamily(family.name, family.identities, family.pathFor)
	}
	track.instanceIdentityDigests(snapshot.InstanceIdentityDigests, manifest.InstanceIdentity)

	review.FullySatisfied = len(review.Missing) == 0 && len(review.DeclaredRedactions) == 0
	return review
}

type evidenceCompletenessTracker struct {
	review     *AuditEvidenceSnapshotCompletenessReview
	included   map[string]struct{}
	redactions []AuditEvidenceBundleRedaction
}

type evidenceRetentionDigestFamily struct {
	name       string
	identities []string
	pathFor    func(string) string
}

func includedIdentitySet(objects []AuditEvidenceBundleIncludedObject) map[string]struct{} {
	included := map[string]struct{}{}
	for i := range objects {
		identity := strings.TrimSpace(objects[i].Digest)
		if identity != "" {
			included[identity] = struct{}{}
		}
	}
	return included
}

func evidenceRetentionDigestFamilies(snapshot AuditEvidenceSnapshot) []evidenceRetentionDigestFamily {
	return []evidenceRetentionDigestFamily{
		{name: "segment_seal_digest", identities: snapshot.SegmentSealDigests, pathFor: func(identity string) string { return evidenceBundleSidecarObjectPath(sealsDirName, identity) }},
		{name: "audit_receipt_digest", identities: snapshot.AuditReceiptDigests, pathFor: func(identity string) string { return evidenceBundleSidecarObjectPath(receiptsDirName, identity) }},
		{name: "verification_report_digest", identities: snapshot.VerificationReportDigests, pathFor: func(identity string) string {
			return evidenceBundleSidecarObjectPath(verificationReportsDirName, identity)
		}},
		{name: "runtime_evidence_digest", identities: snapshot.RuntimeEvidenceDigests, pathFor: func(string) string { return "contracts/signer-evidence.json" }},
		{name: "attestation_evidence_digest", identities: snapshot.AttestationEvidenceDigests, pathFor: func(identity string) string {
			return evidenceBundleSidecarObjectPath(externalAnchorSidecarsDir, identity)
		}},
		{name: "policy_evidence_digest", identities: snapshot.PolicyEvidenceDigests, pathFor: func(identity string) string {
			return evidenceBundleSidecarObjectPath(externalAnchorEvidenceDir, identity)
		}},
		{name: "approval_evidence_digest", identities: snapshot.ApprovalEvidenceDigests, pathFor: func(identity string) string {
			return evidenceBundleSidecarObjectPath(externalAnchorEvidenceDir, identity)
		}},
		{name: "anchor_evidence_digest", identities: snapshot.AnchorEvidenceDigests, pathFor: anchorEvidenceRetentionPath},
	}
}

func anchorEvidenceRetentionPath(identity string) string {
	if strings.HasPrefix(identity, "sha256:") {
		return evidenceBundleSidecarObjectPath(externalAnchorEvidenceDir, identity)
	}
	return ""
}

func (t evidenceCompletenessTracker) digestFamily(family string, identities []string, pathFor func(string) string) {
	for i := range identities {
		identity := strings.TrimSpace(identities[i])
		if identity == "" {
			continue
		}
		_, ok := t.included[identity]
		t.track(family, identity, pathFor(identity), ok)
	}
}

func (t evidenceCompletenessTracker) instanceIdentityDigests(identities []string, manifestIdentity string) {
	for i := range identities {
		identity := strings.TrimSpace(identities[i])
		if identity == "" {
			continue
		}
		t.review.RequiredIdentityCount++
		if strings.TrimSpace(manifestIdentity) != identity {
			t.review.Missing = append(t.review.Missing, AuditEvidenceSnapshotCompleteness{Family: "instance_identity_digest", Identity: identity})
		}
	}
}

func (t evidenceCompletenessTracker) track(family string, identity string, redactionPath string, exists bool) {
	t.review.RequiredIdentityCount++
	if exists {
		return
	}
	entry := AuditEvidenceSnapshotCompleteness{Family: family, Identity: identity}
	if redactionPath != "" && isIdentityPathRedacted(t.redactions, redactionPath) {
		t.review.DeclaredRedactions = append(t.review.DeclaredRedactions, entry)
		return
	}
	t.review.Missing = append(t.review.Missing, entry)
}

func isIdentityPathRedacted(redactions []AuditEvidenceBundleRedaction, objectPath string) bool {
	cleanPath := filepathToBundlePath(objectPath)
	if cleanPath == "" {
		return false
	}
	for i := range redactions {
		if evidenceBundlePathRedactionMatches(cleanPath, redactions[i]) {
			return true
		}
	}
	return false
}

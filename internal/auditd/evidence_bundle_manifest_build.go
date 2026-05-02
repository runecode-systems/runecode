package auditd

type evidenceBundleManifestData struct {
	includedObjects  []AuditEvidenceBundleIncludedObject
	rootDigests      []string
	sealRefs         []AuditEvidenceBundleSealReference
	instanceIdentity string
}

func (l *Ledger) evidenceBundleManifestDataLocked(scope AuditEvidenceBundleScope, profilePolicy evidenceBundleProfilePolicy) (evidenceBundleManifestData, error) {
	selectedSegmentIDs, err := l.selectEvidenceBundleSegmentIDsLocked(scope)
	if err != nil {
		return evidenceBundleManifestData{}, err
	}
	selectedSegmentSet := make(map[string]struct{}, len(selectedSegmentIDs))
	for i := range selectedSegmentIDs {
		selectedSegmentSet[selectedSegmentIDs[i]] = struct{}{}
	}
	included, err := l.collectEvidenceBundleIncludedObjectsLocked(profilePolicy, selectedSegmentIDs, selectedSegmentSet)
	if err != nil {
		return evidenceBundleManifestData{}, err
	}
	rootDigests, sealRefs, err := l.evidenceBundleRootsAndSealsLocked(selectedSegmentIDs)
	if err != nil {
		return evidenceBundleManifestData{}, err
	}
	instanceIdentity, err := l.evidenceBundleInstanceIdentityLocked()
	if err != nil {
		return evidenceBundleManifestData{}, err
	}
	return evidenceBundleManifestData{
		includedObjects:  included,
		rootDigests:      rootDigests,
		sealRefs:         sealRefs,
		instanceIdentity: instanceIdentity,
	}, nil
}

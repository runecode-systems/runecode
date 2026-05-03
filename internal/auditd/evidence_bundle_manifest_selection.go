package auditd

import (
	"os"
	"sort"
	"strings"
)

func (l *Ledger) collectEvidenceBundleIncludedObjectsLocked(profilePolicy evidenceBundleProfilePolicy, segmentIDs []string, segmentSet map[string]struct{}) ([]AuditEvidenceBundleIncludedObject, error) {
	objects := []AuditEvidenceBundleIncludedObject{}
	if profilePolicy.IncludeSegments {
		var err error
		objects, err = l.appendEvidenceBundleSegmentObjectsLocked(objects, segmentIDs)
		if err != nil {
			return nil, err
		}
	}
	if profilePolicy.IncludeVerificationInputs {
		var err error
		objects, err = l.appendEvidenceBundleVerificationInputObjectsLocked(objects)
		if err != nil {
			return nil, err
		}
	}
	selectedSeals, err := l.selectedSealDigestSetLocked(segmentSet)
	if err != nil {
		return nil, err
	}
	for _, family := range evidenceBundleSidecarFamilies(profilePolicy) {
		objects, err = l.appendEvidenceBundleSidecarObjectsLocked(objects, family.dirName, family.objectFamily, segmentSet, selectedSeals)
		if err != nil {
			return nil, err
		}
	}
	sortEvidenceBundleIncludedObjects(objects)
	return objects, nil
}

func (l *Ledger) appendEvidenceBundleVerificationInputObjectsLocked(objects []AuditEvidenceBundleIncludedObject) ([]AuditEvidenceBundleIncludedObject, error) {
	for _, contract := range []struct {
		path   string
		family string
	}{
		{path: "contracts/event-contract-catalog.json", family: "event_contract_catalog"},
		{path: "contracts/verifier-records.json", family: "verifier_record_set"},
		{path: "contracts/signer-evidence.json", family: "signer_evidence"},
		{path: "contracts/storage-posture.json", family: "storage_posture"},
	} {
		object, ok, err := l.evidenceBundleContractObjectLocked(contract.path, contract.family)
		if err != nil {
			return nil, err
		}
		if ok {
			objects = append(objects, object)
		}
	}
	return objects, nil
}

func (l *Ledger) evidenceBundleContractObjectLocked(objectPath string, family string) (AuditEvidenceBundleIncludedObject, bool, error) {
	abs, err := l.bundleObjectAbsolutePathLocked(objectPath)
	if err != nil {
		return AuditEvidenceBundleIncludedObject{}, false, err
	}
	raw, err := readFileBytes(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return AuditEvidenceBundleIncludedObject{}, false, nil
		}
		return AuditEvidenceBundleIncludedObject{}, false, err
	}
	identity, err := digestIdentityFromCanonicalPayload(raw)
	if err != nil {
		return AuditEvidenceBundleIncludedObject{}, false, err
	}
	return AuditEvidenceBundleIncludedObject{ObjectFamily: family, Digest: identity, Path: objectPath, ByteLength: int64(len(raw))}, true, nil
}

func readFileBytes(path string) ([]byte, error) {
	return os.ReadFile(path)
}

type evidenceBundleSidecarFamily struct {
	dirName      string
	objectFamily string
}

func evidenceBundleSidecarFamilies(profilePolicy evidenceBundleProfilePolicy) []evidenceBundleSidecarFamily {
	families := []evidenceBundleSidecarFamily{{dirName: sealsDirName, objectFamily: "audit_segment_seal"}}
	if profilePolicy.IncludeReceipts {
		families = append(families, evidenceBundleSidecarFamily{dirName: receiptsDirName, objectFamily: "audit_receipt"})
	}
	if profilePolicy.IncludeVerificationReports {
		families = append(families, evidenceBundleSidecarFamily{dirName: verificationReportsDirName, objectFamily: "audit_verification_report"})
	}
	if profilePolicy.IncludeExternalAnchor {
		families = append(families,
			evidenceBundleSidecarFamily{dirName: externalAnchorEvidenceDir, objectFamily: "external_anchor_evidence"},
			evidenceBundleSidecarFamily{dirName: externalAnchorSidecarsDir, objectFamily: "external_anchor_sidecar"},
		)
	}
	return families
}

func (l *Ledger) appendEvidenceBundleSegmentObjectsLocked(objects []AuditEvidenceBundleIncludedObject, segmentIDs []string) ([]AuditEvidenceBundleIncludedObject, error) {
	for i := range segmentIDs {
		segmentID := strings.TrimSpace(segmentIDs[i])
		if segmentID == "" {
			continue
		}
		segmentDigest, err := l.segmentObjectDigestIdentityLocked(segmentID)
		if err != nil {
			return nil, err
		}
		segmentPath := evidenceBundleSegmentObjectPath(segmentID)
		byteLength, err := l.evidenceBundleObjectByteLengthLocked(segmentPath)
		if err != nil {
			return nil, err
		}
		objects = append(objects, AuditEvidenceBundleIncludedObject{ObjectFamily: "audit_segment", Digest: segmentDigest, Path: segmentPath, ByteLength: byteLength})
	}
	return objects, nil
}

func (l *Ledger) selectedSealDigestSetLocked(segmentSet map[string]struct{}) (map[string]struct{}, error) {
	selected := map[string]struct{}{}
	for segmentID := range segmentSet {
		lookup, ok, err := l.lookupSegmentSealLocked(segmentID, false)
		if err != nil {
			return nil, err
		}
		if !ok || strings.TrimSpace(lookup.SealDigest) == "" {
			continue
		}
		selected[strings.TrimSpace(lookup.SealDigest)] = struct{}{}
	}
	return selected, nil
}

func (l *Ledger) appendEvidenceBundleSidecarObjectsLocked(objects []AuditEvidenceBundleIncludedObject, dirName string, family string, segmentSet map[string]struct{}, selectedSeals map[string]struct{}) ([]AuditEvidenceBundleIncludedObject, error) {
	identities, err := l.sidecarDigestIdentitiesLocked(dirName)
	if err != nil {
		return nil, err
	}
	for i := range identities {
		identity := strings.TrimSpace(identities[i])
		if identity == "" {
			continue
		}
		include, err := l.shouldIncludeEvidenceBundleObjectLocked(identity, family, segmentSet, selectedSeals)
		if err != nil {
			return nil, err
		}
		if !include {
			continue
		}
		object, err := l.evidenceBundleIncludedSidecarObjectLocked(dirName, family, identity)
		if err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}
	return objects, nil
}

func (l *Ledger) shouldIncludeEvidenceBundleObjectLocked(identity string, family string, segmentSet map[string]struct{}, selectedSeals map[string]struct{}) (bool, error) {
	switch family {
	case "audit_segment_seal":
		_, ok := selectedSeals[strings.TrimSpace(identity)]
		return ok, nil
	case "audit_receipt":
		return l.shouldIncludeReceiptForSegmentSetLocked(identity, segmentSet)
	case "audit_verification_report":
		return l.shouldIncludeVerificationReportForSegmentSetLocked(identity, segmentSet)
	case "external_anchor_evidence", "external_anchor_sidecar":
		return l.shouldIncludeExternalAnchorObjectForSegmentSetLocked(identity, family, segmentSet)
	default:
		return true, nil
	}
}

func (l *Ledger) evidenceBundleIncludedSidecarObjectLocked(dirName string, family string, identity string) (AuditEvidenceBundleIncludedObject, error) {
	objectPath := evidenceBundleSidecarObjectPath(dirName, identity)
	byteLength, err := l.evidenceBundleObjectByteLengthLocked(objectPath)
	if err != nil {
		return AuditEvidenceBundleIncludedObject{}, err
	}
	return AuditEvidenceBundleIncludedObject{ObjectFamily: family, Digest: identity, Path: objectPath, ByteLength: byteLength}, nil
}

func sortEvidenceBundleIncludedObjects(objects []AuditEvidenceBundleIncludedObject) {
	sort.Slice(objects, func(i, j int) bool {
		if objects[i].ObjectFamily == objects[j].ObjectFamily {
			return objects[i].Digest < objects[j].Digest
		}
		return objects[i].ObjectFamily < objects[j].ObjectFamily
	})
}

func (l *Ledger) segmentObjectDigestIdentityLocked(segmentID string) (string, error) {
	segment, err := l.loadSegment(segmentID)
	if err != nil {
		return "", err
	}
	digest, err := canonicalDigest(segment)
	if err != nil {
		return "", err
	}
	return digest.Identity()
}

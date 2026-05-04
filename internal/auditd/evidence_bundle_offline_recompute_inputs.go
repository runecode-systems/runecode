package auditd

import (
	"encoding/json"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type offlineRequiredRecomputeInputs struct {
	segment         trustpolicy.AuditSegmentFilePayload
	segmentRaw      []byte
	sealEnvelope    trustpolicy.SignedObjectEnvelope
	sealDigest      trustpolicy.Digest
	verifierRecords []trustpolicy.VerifierRecord
	eventCatalog    trustpolicy.AuditEventContractCatalog
}

func loadOfflineRequiredRecomputeInputs(bundle offlineBundleSnapshot, segmentID string) (offlineRequiredRecomputeInputs, []string, error) {
	missing := []string{}
	segment, segmentRaw, hasSegment, err := loadOfflineSegmentObject(bundle, segmentID)
	if err != nil {
		return offlineRequiredRecomputeInputs{}, nil, err
	}
	if !hasSegment {
		missing = append(missing, "segment:"+segmentID)
	}
	sealEnvelope, sealDigest, hasSeal, err := loadOfflineSealEnvelopeForSegment(bundle, segmentID)
	if err != nil {
		return offlineRequiredRecomputeInputs{}, nil, err
	}
	if !hasSeal {
		missing = append(missing, "segment_seal:"+segmentID)
	}
	verifierRecords, hasVerifierRecords, err := loadOfflineVerifierRecords(bundle)
	if err != nil {
		return offlineRequiredRecomputeInputs{}, nil, err
	}
	if !hasVerifierRecords {
		missing = append(missing, "contracts/verifier-records.json")
	}
	eventCatalog, hasEventCatalog, err := loadOfflineEventCatalog(bundle)
	if err != nil {
		return offlineRequiredRecomputeInputs{}, nil, err
	}
	if !hasEventCatalog {
		missing = append(missing, "contracts/event-contract-catalog.json")
	}
	return offlineRequiredRecomputeInputs{
		segment:         segment,
		segmentRaw:      segmentRaw,
		sealEnvelope:    sealEnvelope,
		sealDigest:      sealDigest,
		verifierRecords: verifierRecords,
		eventCatalog:    eventCatalog,
	}, missing, nil
}

func loadOfflineSegmentObject(bundle offlineBundleSnapshot, segmentID string) (trustpolicy.AuditSegmentFilePayload, []byte, bool, error) {
	path := evidenceBundleSegmentObjectPath(segmentID)
	raw, ok := bundle.objects[path]
	if !ok {
		return trustpolicy.AuditSegmentFilePayload{}, nil, false, nil
	}
	segment := trustpolicy.AuditSegmentFilePayload{}
	if err := json.Unmarshal(raw.content, &segment); err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, nil, false, err
	}
	framed, err := offlineRawFramedSegmentBytes(segment)
	if err != nil {
		return trustpolicy.AuditSegmentFilePayload{}, nil, false, err
	}
	return segment, framed, true, nil
}

func offlineRawFramedSegmentBytes(segment trustpolicy.AuditSegmentFilePayload) ([]byte, error) {
	raw := make([]byte, 0, len(segment.Frames)*256)
	for i := range segment.Frames {
		envelope, err := decodeFrameEnvelope(segment.Frames[i])
		if err != nil {
			return nil, err
		}
		canonical, _, err := canonicalEnvelopeAndDigest(envelope)
		if err != nil {
			return nil, err
		}
		raw = append(raw, canonical...)
		raw = append(raw, '\n')
	}
	return raw, nil
}

func loadOfflineSealEnvelopeForSegment(bundle offlineBundleSnapshot, segmentID string) (trustpolicy.SignedObjectEnvelope, trustpolicy.Digest, bool, error) {
	for i := range bundle.manifest.SealReferences {
		ref := bundle.manifest.SealReferences[i]
		if strings.TrimSpace(ref.SegmentID) != strings.TrimSpace(segmentID) {
			continue
		}
		obj, ok := offlineObjectByFamilyAndDigest(bundle, "audit_segment_seal", strings.TrimSpace(ref.SealDigest))
		if !ok {
			return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, false, nil
		}
		envelope := trustpolicy.SignedObjectEnvelope{}
		if err := json.Unmarshal(obj.content, &envelope); err != nil {
			return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, false, err
		}
		digest, err := digestFromIdentity(strings.TrimSpace(ref.SealDigest))
		if err != nil {
			return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, false, err
		}
		return envelope, digest, true, nil
	}
	return trustpolicy.SignedObjectEnvelope{}, trustpolicy.Digest{}, false, nil
}

func loadOfflineVerifierRecords(bundle offlineBundleSnapshot) ([]trustpolicy.VerifierRecord, bool, error) {
	obj, ok := offlineObjectByFamily(bundle, "verifier_record_set")
	if !ok {
		return nil, false, nil
	}
	records := []trustpolicy.VerifierRecord{}
	if err := json.Unmarshal(obj.content, &records); err != nil {
		return nil, true, err
	}
	return records, true, nil
}

func loadOfflineEventCatalog(bundle offlineBundleSnapshot) (trustpolicy.AuditEventContractCatalog, bool, error) {
	obj, ok := offlineObjectByFamily(bundle, "event_contract_catalog")
	if !ok {
		return trustpolicy.AuditEventContractCatalog{}, false, nil
	}
	catalog := trustpolicy.AuditEventContractCatalog{}
	if err := json.Unmarshal(obj.content, &catalog); err != nil {
		return trustpolicy.AuditEventContractCatalog{}, true, err
	}
	return catalog, true, nil
}

func loadOfflineReceiptsForSeal(bundle offlineBundleSnapshot, sealDigest trustpolicy.Digest) ([]trustpolicy.SignedObjectEnvelope, error) {
	sealIdentity, _ := sealDigest.Identity()
	receipts := []trustpolicy.SignedObjectEnvelope{}
	receiptObjects := offlineBundleObjectsByFamily(bundle.manifest.IncludedObjects, "audit_receipt")
	for i := range receiptObjects {
		objRaw, ok := bundle.objects[receiptObjects[i].Path]
		if !ok {
			continue
		}
		envelope := trustpolicy.SignedObjectEnvelope{}
		if err := json.Unmarshal(objRaw.content, &envelope); err != nil {
			continue
		}
		payload := bundleManifestReceiptPayload{}
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			continue
		}
		subjectIdentity, err := payload.SubjectDigest.Identity()
		if err == nil && strings.TrimSpace(subjectIdentity) == strings.TrimSpace(sealIdentity) {
			receipts = append(receipts, envelope)
		}
	}
	return receipts, nil
}

func offlineKnownSealDigests(refs []AuditEvidenceBundleSealReference) ([]trustpolicy.Digest, error) {
	out := make([]trustpolicy.Digest, 0, len(refs))
	for i := range refs {
		d, err := digestFromIdentity(refs[i].SealDigest)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

package artifacts

import (
	"fmt"
	"os"
	"strings"
)

func (s *Store) reconcileArtifactIndexFromAuditLocked() (bool, error) {
	events, err := s.storeIO.readAuditEvents()
	if err != nil {
		return false, err
	}
	changed := false
	for _, event := range events {
		record, ok, err := s.recoveredArtifactRecordFromAuditEvent(event)
		if err != nil {
			return false, err
		}
		if !ok {
			continue
		}
		s.state.Artifacts[record.Reference.Digest] = record
		changed = true
	}
	return changed, nil
}

func (s *Store) recoveredArtifactRecordFromAuditEvent(event AuditEvent) (ArtifactRecord, bool, error) {
	details, ok := artifactPutDetailsFromAuditEvent(event)
	if !ok || s.artifactIndexed(details.digest) {
		return ArtifactRecord{}, false, nil
	}
	blobPath, size, ok, err := s.recoverableArtifactBlob(details.digest)
	if err != nil || !ok {
		return ArtifactRecord{}, false, err
	}
	return ArtifactRecord{
		Reference:         ArtifactReference{Digest: details.digest, SizeBytes: size, ContentType: "application/octet-stream", DataClass: details.dataClass, ProvenanceReceiptHash: details.provenance},
		BlobPath:          blobPath,
		CreatedAt:         event.OccurredAt.UTC(),
		CreatedByRole:     recoveredArtifactActor(event.Actor),
		StorageProtection: s.state.StorageProtectionPosture,
	}, true, nil
}

func (s *Store) artifactIndexed(digest string) bool {
	_, exists := s.state.Artifacts[digest]
	return exists
}

func (s *Store) recoverableArtifactBlob(digest string) (string, int64, bool, error) {
	blobPath, err := s.storeIO.validatedBlobPath(digest)
	if err != nil {
		return "", 0, false, nil
	}
	blobInfo, err := os.Stat(blobPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0, false, nil
		}
		return "", 0, false, err
	}
	blobBytes, err := s.storeIO.readBlob(blobPath)
	if err != nil {
		return "", 0, false, err
	}
	if computed := digestBytes(blobBytes); computed != digest {
		return "", 0, false, fmt.Errorf("artifact blob digest mismatch for %q", digest)
	}
	return blobPath, blobInfo.Size(), true, nil
}

type recoveredArtifactAuditDetails struct {
	digest     string
	dataClass  DataClass
	provenance string
}

func artifactPutDetailsFromAuditEvent(event AuditEvent) (recoveredArtifactAuditDetails, bool) {
	if event.Type != "artifact_put" {
		return recoveredArtifactAuditDetails{}, false
	}
	digest, _ := event.Details["digest"].(string)
	if !isValidDigest(digest) {
		return recoveredArtifactAuditDetails{}, false
	}
	dataClass := DataClass(strings.TrimSpace(stringValue(event.Details, "data_class")))
	if _, ok := allDataClasses[dataClass]; !ok {
		return recoveredArtifactAuditDetails{}, false
	}
	provenance := stringValue(event.Details, "provenance_receipt_hash")
	if !isValidDigest(provenance) {
		provenance = ""
	}
	return recoveredArtifactAuditDetails{digest: digest, dataClass: dataClass, provenance: provenance}, true
}

func recoveredArtifactActor(actor string) string {
	return createdByRole(PutRequest{CreatedByRole: actor})
}

func stringValue(details map[string]interface{}, key string) string {
	value, _ := details[key].(string)
	return value
}

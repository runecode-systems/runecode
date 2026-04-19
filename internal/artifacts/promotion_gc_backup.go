package artifacts

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

type gcCandidate struct {
	digest string
	rec    ArtifactRecord
}

func (s *Store) buildApprovedRecord(source ArtifactRecord, req PromotionRequest) (ArtifactReference, ArtifactRecord, string, string, error) {
	payload, err := s.storeIO.readBlob(source.BlobPath)
	if err != nil {
		return ArtifactReference{}, ArtifactRecord{}, "", "", err
	}
	approvedPayload := append([]byte("approved:\n"), payload...)
	newDigest := digestBytes(approvedPayload)
	if _, err := s.storeIO.writeBlobIfMissing(newDigest, approvedPayload); err != nil {
		return ArtifactReference{}, ArtifactRecord{}, "", "", err
	}
	now := s.nowFn().UTC()
	decisionHash, requestHash, err := promotionHashes(req)
	if err != nil {
		return ArtifactReference{}, ArtifactRecord{}, "", "", err
	}
	ref := ArtifactReference{
		Digest:                newDigest,
		SizeBytes:             int64(len(approvedPayload)),
		ContentType:           source.Reference.ContentType,
		DataClass:             DataClassApprovedFileExcerpts,
		ProvenanceReceiptHash: source.Reference.ProvenanceReceiptHash,
	}
	record := ArtifactRecord{
		Reference:            ref,
		BlobPath:             s.storeIO.blobPath(newDigest),
		CreatedAt:            now,
		CreatedByRole:        source.CreatedByRole,
		RunID:                source.RunID,
		StepID:               source.StepID,
		StorageProtection:    s.state.StorageProtectionPosture,
		ApprovalOfDigest:     req.UnapprovedDigest,
		ApprovalDecisionHash: decisionHash,
		PromotionRequestHash: requestHash,
		PromotionApprovedBy:  req.Approver,
		PromotionApprovedAt:  &now,
	}
	return ref, record, decisionHash, requestHash, nil
}

func promotionHashes(req PromotionRequest) (string, string, error) {
	decisionHash, err := promotionDecisionHash(req)
	if err != nil {
		return "", "", err
	}
	requestHash, err := promotionActionRequestHash(req)
	if err != nil {
		return "", "", err
	}
	return decisionHash, requestHash, nil
}

func promotionDecisionHash(req PromotionRequest) (string, error) {
	if req.ApprovalDecision == nil {
		return "", ErrApprovalArtifactRequired
	}
	b, err := json.Marshal(req.ApprovalDecision)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalizeJSONBytes(b)
	if err != nil {
		return "", err
	}
	return digestBytes(canonical), nil
}

func promotionAuditDetails(req PromotionRequest, approvedDigest, requestHash, decisionHash string) map[string]interface{} {
	return map[string]interface{}{
		"action":                  "promoted",
		"source_digest":           req.UnapprovedDigest,
		"approved_digest":         approvedDigest,
		"promotion_request_hash":  requestHash,
		"promotion_decision_hash": decisionHash,
		"repo_path":               req.RepoPath,
		"commit":                  req.Commit,
		"extractor_tool_version":  req.ExtractorToolVersion,
	}
}

func recentTimes(entries []time.Time, windowStart time.Time) []time.Time {
	out := make([]time.Time, 0, len(entries))
	for _, ts := range entries {
		if ts.After(windowStart) {
			out = append(out, ts)
		}
	}
	return out
}

func ensureTTL(seconds int64) time.Duration {
	ttl := time.Duration(seconds) * time.Second
	if ttl <= 0 {
		return 7 * 24 * time.Hour
	}
	return ttl
}

func gcCandidates(artifactsMap map[string]ArtifactRecord, runs map[string]string, now time.Time, ttl time.Duration) []gcCandidate {
	result := []gcCandidate{}
	for digest, rec := range artifactsMap {
		if rec.RunID != "" {
			if status := runs[rec.RunID]; status == "active" || status == "retained" {
				continue
			}
		}
		if now.Sub(rec.CreatedAt) < ttl {
			continue
		}
		result = append(result, gcCandidate{digest: digest, rec: rec})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].digest < result[j].digest })
	return result
}

func buildBackupManifest(state StoreState, exportedAt time.Time) BackupManifest {
	manifest := newBackupManifest(state, exportedAt)
	populateBackupManifestCollections(&manifest, state)
	sortBackupManifestCollections(&manifest)
	return manifest
}

func newBackupManifest(state StoreState, exportedAt time.Time) BackupManifest {
	return BackupManifest{
		Schema:                "runecode.backup.artifacts.v1",
		ExportedAt:            exportedAt,
		StorageProtection:     state.StorageProtectionPosture,
		Policy:                state.Policy,
		Runs:                  map[string]string{},
		Artifacts:             make([]ArtifactRecord, 0, len(state.Artifacts)),
		Sessions:              make([]SessionDurableState, 0, len(state.Sessions)),
		PolicyDecisions:       make([]PolicyDecisionRecord, 0, len(state.PolicyDecisions)),
		Approvals:             make([]ApprovalRecord, 0, len(state.Approvals)),
		ProviderProfiles:      make([]ProviderProfileDurableState, 0, len(state.ProviderProfiles)),
		ProviderSetupSessions: make([]ProviderSetupSessionDurableState, 0, len(state.ProviderSetupSessions)),
	}
}

func populateBackupManifestCollections(manifest *BackupManifest, state StoreState) {
	for runID, status := range state.Runs {
		manifest.Runs[runID] = status
	}
	for _, rec := range state.Artifacts {
		manifest.Artifacts = append(manifest.Artifacts, rec)
	}
	for _, rec := range state.Sessions {
		manifest.Sessions = append(manifest.Sessions, copySessionDurableState(rec))
	}
	for _, rec := range state.PolicyDecisions {
		manifest.PolicyDecisions = append(manifest.PolicyDecisions, rec)
	}
	for _, rec := range state.Approvals {
		manifest.Approvals = append(manifest.Approvals, rec)
	}
	manifest.ProviderProfiles = append(manifest.ProviderProfiles, sortedProviderProfiles(state.ProviderProfiles)...)
	manifest.ProviderSetupSessions = append(manifest.ProviderSetupSessions, sortedProviderSetupSessions(state.ProviderSetupSessions)...)
}

func sortBackupManifestCollections(manifest *BackupManifest) {
	sort.Slice(manifest.Artifacts, func(i, j int) bool {
		return manifest.Artifacts[i].Reference.Digest < manifest.Artifacts[j].Reference.Digest
	})
	sort.Slice(manifest.Sessions, func(i, j int) bool {
		return manifest.Sessions[i].SessionID < manifest.Sessions[j].SessionID
	})
	sort.Slice(manifest.PolicyDecisions, func(i, j int) bool {
		return manifest.PolicyDecisions[i].Digest < manifest.PolicyDecisions[j].Digest
	})
	sort.Slice(manifest.Approvals, func(i, j int) bool {
		return manifest.Approvals[i].ApprovalID < manifest.Approvals[j].ApprovalID
	})
	sort.Slice(manifest.ProviderProfiles, func(i, j int) bool {
		return manifest.ProviderProfiles[i].ProviderProfileID < manifest.ProviderProfiles[j].ProviderProfileID
	})
	sort.Slice(manifest.ProviderSetupSessions, func(i, j int) bool {
		return manifest.ProviderSetupSessions[i].SetupSessionID < manifest.ProviderSetupSessions[j].SetupSessionID
	})
}

func validateRestoredRecord(record ArtifactRecord, ioStore *storeIO) (ArtifactRecord, error) {
	if _, ok := allDataClasses[record.Reference.DataClass]; !ok {
		return ArtifactRecord{}, ErrInvalidDataClass
	}
	blobPath, err := ioStore.validatedBlobPath(record.Reference.Digest)
	if err != nil {
		return ArtifactRecord{}, err
	}
	blob, err := ioStore.readBlob(blobPath)
	if err != nil {
		return ArtifactRecord{}, err
	}
	actualDigest := digestBytes(blob)
	if actualDigest != record.Reference.Digest {
		return ArtifactRecord{}, fmt.Errorf("backup digest mismatch for %s", record.Reference.Digest)
	}
	if int64(len(blob)) != record.Reference.SizeBytes {
		return ArtifactRecord{}, fmt.Errorf("backup size mismatch for %s", record.Reference.Digest)
	}
	record.BlobPath = blobPath
	return record, nil
}

func validateApprovedRestores(allArtifacts map[string]ArtifactRecord, unapprovedByDigest map[string]ArtifactRecord) error {
	for _, rec := range allArtifacts {
		if rec.Reference.DataClass != DataClassApprovedFileExcerpts {
			continue
		}
		if rec.ApprovalOfDigest == "" || rec.ApprovalDecisionHash == "" || rec.PromotionRequestHash == "" || rec.PromotionApprovedBy == "" || rec.PromotionApprovedAt == nil {
			return ErrApprovedClassRequiresPromotion
		}
		source, ok := unapprovedByDigest[rec.ApprovalOfDigest]
		if !ok {
			return fmt.Errorf("approved excerpt %s missing unapproved source %s", rec.Reference.Digest, rec.ApprovalOfDigest)
		}
		if source.Reference.DataClass != DataClassUnapprovedFileExcerpts {
			return fmt.Errorf("approved excerpt %s source class mismatch", rec.Reference.Digest)
		}
	}
	return nil
}

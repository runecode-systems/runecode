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

func gcCandidates(artifactsMap map[string]ArtifactRecord, runs map[string]string, dependencyBatches map[string]DependencyCacheBatchRecord, dependencyUnits map[string]DependencyCacheResolvedUnitRecord, now time.Time, ttl time.Duration) []gcCandidate {
	result := []gcCandidate{}
	canonicalDependencyDigests := dependencyCanonicalDigests(dependencyBatches, dependencyUnits)
	for digest, rec := range artifactsMap {
		if shouldSkipGCCandidate(digest, rec, canonicalDependencyDigests, runs, now, ttl) {
			continue
		}
		result = append(result, gcCandidate{digest: digest, rec: rec})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].digest < result[j].digest })
	return result
}

func shouldSkipGCCandidate(digest string, rec ArtifactRecord, canonicalDependencyDigests map[string]struct{}, runs map[string]string, now time.Time, ttl time.Duration) bool {
	if _, protected := canonicalDependencyDigests[digest]; protected {
		return true
	}
	if rec.RunID != "" {
		status := runs[rec.RunID]
		if status == "active" || status == "retained" {
			return true
		}
	}
	return now.Sub(rec.CreatedAt) < ttl
}

func dependencyCanonicalDigests(dependencyBatches map[string]DependencyCacheBatchRecord, dependencyUnits map[string]DependencyCacheResolvedUnitRecord) map[string]struct{} {
	protected := map[string]struct{}{}
	for _, batch := range dependencyBatches {
		if batch.ResolutionState != "complete" {
			continue
		}
		protected[batch.BatchManifestDigest] = struct{}{}
		for _, digest := range batch.ResolvedUnitDigests {
			protected[digest] = struct{}{}
		}
	}
	for _, unit := range dependencyUnits {
		if unit.IntegrityState != "verified" {
			continue
		}
		protected[unit.ManifestDigest] = struct{}{}
		for _, digest := range unit.PayloadDigest {
			protected[digest] = struct{}{}
		}
	}
	return protected
}

func validateRestoredRecord(record ArtifactRecord, ioStore *storeIO) (ArtifactRecord, error) {
	if _, ok := allDataClasses[record.Reference.DataClass]; !ok {
		return ArtifactRecord{}, ErrInvalidDataClass
	}
	blobPath, err := ioStore.validatedBlobPath(record.Reference.Digest)
	if err != nil {
		return ArtifactRecord{}, err
	}
	if err := ioStore.verifyBlobDigestAndSize(blobPath, record.Reference.Digest, record.Reference.SizeBytes); err != nil {
		return ArtifactRecord{}, err
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

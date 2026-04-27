package artifacts

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

type streamedArtifactBlob struct {
	tmpPath string
	digest  string
	size    int64
}

func (s *Store) SetPolicy(policy Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validatePolicy(policy); err != nil {
		return err
	}
	s.state.Policy = policy
	return s.saveStateLocked()
}

func (s *Store) Policy() Policy {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.Policy
}

func (s *Store) Put(req PutRequest) (ArtifactReference, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.putLocked(req)
}

func (s *Store) PutStream(req PutStreamRequest) (ArtifactReference, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.putStreamLocked(req)
}

func (s *Store) putStreamLocked(req PutStreamRequest) (ArtifactReference, error) {
	putReq, err := prepareStreamPutRequest(req, s.state.Policy)
	if err != nil {
		return ArtifactReference{}, err
	}
	actorRole := createdByRole(putReq)
	blob, err := s.streamPutBlobLocked(req.Reader, putReq, actorRole)
	if err != nil {
		return ArtifactReference{}, err
	}
	cleanupTmp := func() {
		if blob.tmpPath != "" {
			_ = s.storeIO.removeBlob(blob.tmpPath)
		}
	}
	if ref, ok, err := s.tryReturnExistingReference(putReq, blob.digest); ok || err != nil {
		cleanupTmp()
		return ref, err
	}
	createdBlob, err := s.storeIO.persistBlobFromTempFile(blob.tmpPath, blob.digest)
	if err != nil {
		cleanupTmp()
		return ArtifactReference{}, err
	}
	blob.tmpPath = ""
	ref := buildArtifactReference(blob.digest, blob.size, putReq)
	blobPath := s.storeIO.blobPath(blob.digest)
	rollback := s.captureArtifactPutRollback(putReq.RunID, blob.digest)
	s.upsertArtifactRecord(ref, putReq, blob.digest, actorRole)
	if putReq.RunID != "" {
		s.state.Runs[putReq.RunID] = "active"
	}
	if err := s.appendAuditLocked("artifact_put", actorRole, map[string]interface{}{"digest": blob.digest, "data_class": putReq.DataClass, "provenance_receipt_hash": putReq.ProvenanceReceiptHash}); err != nil {
		return ArtifactReference{}, s.rollbackStagedArtifactPut(rollback, blobPath, createdBlob, err)
	}
	if err := s.saveStateLocked(); err != nil {
		return ArtifactReference{}, s.rollbackStagedArtifactPut(rollback, blobPath, createdBlob, err)
	}
	return ref, nil
}

func (s *Store) streamPutBlobLocked(reader io.Reader, putReq PutRequest, actorRole string) (streamedArtifactBlob, error) {
	tmpPath, digest, size, err := s.storeIO.streamToTempBlob(reader)
	if err != nil {
		return streamedArtifactBlob{}, err
	}
	if err := s.checkQuotasLocked(actorRole, putReq.StepID, size); err != nil {
		_ = s.storeIO.removeBlob(tmpPath)
		if auditErr := s.appendAuditLocked("artifact_quota_violation", actorRole, map[string]interface{}{"role": actorRole, "step_id": putReq.StepID}); auditErr != nil {
			return streamedArtifactBlob{}, errors.Join(err, auditErr)
		}
		return streamedArtifactBlob{}, err
	}
	return streamedArtifactBlob{tmpPath: tmpPath, digest: digest, size: size}, nil
}

func prepareStreamPutRequest(req PutStreamRequest, policy Policy) (PutRequest, error) {
	if req.Reader == nil {
		return PutRequest{}, fmt.Errorf("reader is required")
	}
	putReq := PutRequest{
		ContentType:           req.ContentType,
		DataClass:             req.DataClass,
		ProvenanceReceiptHash: req.ProvenanceReceiptHash,
		CreatedByRole:         req.CreatedByRole,
		TrustedSource:         req.TrustedSource,
		RunID:                 req.RunID,
		StepID:                req.StepID,
	}
	if err := validateStreamPutRequest(putReq, policy); err != nil {
		return PutRequest{}, err
	}
	return putReq, nil
}

func validateStreamPutRequest(req PutRequest, policy Policy) error {
	if req.TrustedSource && strings.TrimSpace(req.CreatedByRole) == "" {
		return ErrTrustedCreatedByRoleRequired
	}
	if isDependencyDataClass(req.DataClass) && !req.TrustedSource {
		return ErrDependencyCacheTrustedSourceRequired
	}
	if err := validatePutRequest(req, policy); err != nil {
		return err
	}
	if isJSONContentType(req.ContentType) {
		return fmt.Errorf("stream put does not support json canonicalization")
	}
	return nil
}

func (s *Store) putLocked(req PutRequest) (ArtifactReference, error) {
	if req.TrustedSource && strings.TrimSpace(req.CreatedByRole) == "" {
		return ArtifactReference{}, ErrTrustedCreatedByRoleRequired
	}
	if isDependencyDataClass(req.DataClass) && !req.TrustedSource {
		return ArtifactReference{}, ErrDependencyCacheTrustedSourceRequired
	}
	actorRole := createdByRole(req)
	payload, digest, err := s.preparePutPayload(req)
	if err != nil {
		return ArtifactReference{}, err
	}
	if ref, ok, err := s.tryReturnExistingReference(req, digest); ok || err != nil {
		return ref, err
	}
	ref := buildArtifactReference(digest, int64(len(payload)), req)
	blobPath, createdBlob, rollback, err := s.stageArtifactPut(ref, req, digest, payload, actorRole)
	if err != nil {
		return ArtifactReference{}, err
	}
	if err := s.appendAuditLocked("artifact_put", actorRole, map[string]interface{}{"digest": digest, "data_class": req.DataClass, "provenance_receipt_hash": req.ProvenanceReceiptHash}); err != nil {
		return ArtifactReference{}, s.rollbackStagedArtifactPut(rollback, blobPath, createdBlob, err)
	}
	if err := s.saveStateLocked(); err != nil {
		return ArtifactReference{}, s.rollbackStagedArtifactPut(rollback, blobPath, createdBlob, err)
	}
	return ref, nil
}

func (s *Store) rollbackStagedArtifactPut(rollback func(), blobPath string, createdBlob bool, cause error) error {
	rollback()
	if !createdBlob {
		return cause
	}
	if removeErr := s.storeIO.removeBlob(blobPath); removeErr != nil {
		return errors.Join(cause, removeErr)
	}
	return cause
}

func (s *Store) stageArtifactPut(ref ArtifactReference, req PutRequest, digest string, payload []byte, actorRole string) (string, bool, func(), error) {
	createdBlob, err := s.storeIO.writeBlobIfMissing(digest, payload)
	if err != nil {
		return "", false, nil, err
	}
	blobPath := s.storeIO.blobPath(digest)
	rollback := s.captureArtifactPutRollback(req.RunID, digest)
	s.upsertArtifactRecord(ref, req, digest, actorRole)
	if req.RunID != "" {
		s.state.Runs[req.RunID] = "active"
	}
	return blobPath, createdBlob, rollback, nil
}

func (s *Store) captureArtifactPutRollback(runID, digest string) func() {
	priorRunStatus, hadRunStatus := "", false
	if runID != "" {
		priorRunStatus, hadRunStatus = s.state.Runs[runID]
	}
	return func() {
		delete(s.state.Artifacts, digest)
		if runID == "" {
			return
		}
		if hadRunStatus {
			s.state.Runs[runID] = priorRunStatus
			return
		}
		delete(s.state.Runs, runID)
	}
}

func (s *Store) preparePutPayload(req PutRequest) ([]byte, string, error) {
	if err := validatePutRequest(req, s.state.Policy); err != nil {
		return nil, "", err
	}
	canonical, err := canonicalPayload(req.ContentType, req.Payload)
	if err != nil {
		return nil, "", err
	}
	actorRole := createdByRole(req)
	if err := s.checkQuotasLocked(actorRole, req.StepID, int64(len(canonical))); err != nil {
		if auditErr := s.appendAuditLocked("artifact_quota_violation", actorRole, map[string]interface{}{"role": actorRole, "step_id": req.StepID}); auditErr != nil {
			return nil, "", errors.Join(err, auditErr)
		}
		return nil, "", err
	}
	return canonical, digestBytes(canonical), nil
}

func (s *Store) tryReturnExistingReference(req PutRequest, digest string) (ArtifactReference, bool, error) {
	existing, ok := s.state.Artifacts[digest]
	if !ok {
		return ArtifactReference{}, false, nil
	}
	if existing.Reference.DataClass != req.DataClass {
		return ArtifactReference{}, true, ErrDataClassMutationDenied
	}
	return existing.Reference, true, nil
}

func (s *Store) upsertArtifactRecord(ref ArtifactReference, req PutRequest, digest string, actorRole string) {
	now := s.nowFn().UTC()
	s.state.Artifacts[digest] = ArtifactRecord{
		Reference:         ref,
		BlobPath:          s.storeIO.blobPath(digest),
		CreatedAt:         now,
		CreatedByRole:     actorRole,
		RunID:             req.RunID,
		StepID:            req.StepID,
		StorageProtection: s.state.StorageProtectionPosture,
	}
}

func createdByRole(req PutRequest) string {
	role := strings.TrimSpace(req.CreatedByRole)
	if req.TrustedSource {
		return role
	}
	if role == "workspace" || role == "model_gateway" || role == "untrusted_client" {
		return role
	}
	if role == "" {
		return "untrusted_client"
	}
	return "untrusted_client"
}

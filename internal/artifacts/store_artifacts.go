package artifacts

import (
	"errors"
	"io"
	"sort"
	"strings"
)

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

func (s *Store) putLocked(req PutRequest) (ArtifactReference, error) {
	if req.TrustedSource && strings.TrimSpace(req.CreatedByRole) == "" {
		return ArtifactReference{}, ErrTrustedCreatedByRoleRequired
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
	blobPath, rollback, err := s.stageArtifactPut(ref, req, digest, payload, actorRole)
	if err != nil {
		return ArtifactReference{}, err
	}
	if err := s.appendAuditLocked("artifact_put", actorRole, map[string]interface{}{"digest": digest, "data_class": req.DataClass, "provenance_receipt_hash": req.ProvenanceReceiptHash}); err != nil {
		return ArtifactReference{}, s.rollbackStagedArtifactPut(rollback, blobPath, err)
	}
	if err := s.saveStateLocked(); err != nil {
		return ArtifactReference{}, s.rollbackStagedArtifactPut(rollback, blobPath, err)
	}
	return ref, nil
}

func (s *Store) rollbackStagedArtifactPut(rollback func(), blobPath string, cause error) error {
	rollback()
	if removeErr := s.storeIO.removeBlob(blobPath); removeErr != nil {
		return errors.Join(cause, removeErr)
	}
	return cause
}

func (s *Store) stageArtifactPut(ref ArtifactReference, req PutRequest, digest string, payload []byte, actorRole string) (string, func(), error) {
	if err := s.storeIO.writeBlobIfMissing(digest, payload); err != nil {
		return "", nil, err
	}
	blobPath := s.storeIO.blobPath(digest)
	rollback := s.captureArtifactPutRollback(req.RunID, digest)
	s.upsertArtifactRecord(ref, req, digest, actorRole)
	if req.RunID != "" {
		s.state.Runs[req.RunID] = "active"
	}
	return blobPath, rollback, nil
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

func (s *Store) Get(digest string) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.lookupRecord(digest)
	if err != nil {
		return nil, err
	}
	return s.storeIO.openBlob(record.BlobPath)
}

func (s *Store) GetForFlow(req ArtifactReadRequest) (io.ReadCloser, ArtifactRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.lookupRecord(req.Digest)
	if err != nil {
		return nil, ArtifactRecord{}, err
	}
	checkReq := flowCheckRequestFromRead(req, record.Reference.DataClass)
	if err := validateFlowInputs(s.state.Policy, checkReq); err != nil {
		return nil, ArtifactRecord{}, err
	}
	if err := s.enforceFlowRecordConsistencyLocked(record, checkReq); err != nil {
		return nil, ArtifactRecord{}, err
	}
	if err := s.enforceFlowPolicyLocked(checkReq); err != nil {
		return nil, ArtifactRecord{}, err
	}
	r, err := s.storeIO.openBlob(record.BlobPath)
	if err != nil {
		return nil, ArtifactRecord{}, err
	}
	return r, record, nil
}

func flowCheckRequestFromRead(req ArtifactReadRequest, fallbackClass DataClass) FlowCheckRequest {
	class := req.DataClass
	if class == "" {
		class = fallbackClass
	}
	return FlowCheckRequest{
		ProducerRole:  req.ProducerRole,
		ConsumerRole:  req.ConsumerRole,
		DataClass:     class,
		Digest:        req.Digest,
		IsEgress:      req.IsEgress,
		ManifestOptIn: req.ManifestOptIn,
	}
}

func (s *Store) Head(digest string) (ArtifactRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lookupRecord(digest)
}

func (s *Store) lookupRecord(digest string) (ArtifactRecord, error) {
	if !isValidDigest(digest) {
		return ArtifactRecord{}, ErrInvalidDigest
	}
	record, ok := s.state.Artifacts[digest]
	if !ok {
		return ArtifactRecord{}, ErrArtifactNotFound
	}
	return record, nil
}

func (s *Store) List() []ArtifactRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ArtifactRecord, 0, len(s.state.Artifacts))
	for _, rec := range s.state.Artifacts {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (s *Store) SetRunStatus(runID, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validateRunStatusInput(runID, status); err != nil {
		return err
	}
	s.state.Runs[runID] = status
	return s.saveStateLocked()
}

func (s *Store) RunStatuses() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.state.Runs))
	for runID, status := range s.state.Runs {
		out[runID] = status
	}
	return out
}

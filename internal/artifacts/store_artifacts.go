package artifacts

import (
	"errors"
	"io"
	"sort"
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
	payload, digest, err := s.preparePutPayload(req)
	if err != nil {
		return ArtifactReference{}, err
	}
	if ref, ok, err := s.tryReturnExistingReference(req, digest); ok || err != nil {
		return ref, err
	}
	if err := s.storeIO.writeBlobIfMissing(digest, payload); err != nil {
		return ArtifactReference{}, err
	}
	ref := buildArtifactReference(digest, int64(len(payload)), req)
	s.upsertArtifactRecord(ref, req, digest)
	if req.RunID != "" {
		s.state.Runs[req.RunID] = "active"
	}
	if err := s.appendAuditLocked("artifact_put", req.CreatedByRole, map[string]interface{}{"digest": digest, "data_class": req.DataClass}); err != nil {
		return ArtifactReference{}, err
	}
	if err := s.saveStateLocked(); err != nil {
		return ArtifactReference{}, err
	}
	return ref, nil
}

func (s *Store) preparePutPayload(req PutRequest) ([]byte, string, error) {
	if err := validatePutRequest(req, s.state.Policy); err != nil {
		return nil, "", err
	}
	canonical, err := canonicalPayload(req.ContentType, req.Payload)
	if err != nil {
		return nil, "", err
	}
	if err := s.checkQuotasLocked(req.CreatedByRole, req.StepID, int64(len(canonical))); err != nil {
		if auditErr := s.appendAuditLocked("artifact_quota_violation", req.CreatedByRole, map[string]interface{}{"role": req.CreatedByRole, "step_id": req.StepID}); auditErr != nil {
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

func (s *Store) upsertArtifactRecord(ref ArtifactReference, req PutRequest, digest string) {
	now := s.nowFn().UTC()
	s.state.Artifacts[digest] = ArtifactRecord{
		Reference:         ref,
		BlobPath:          s.storeIO.blobPath(digest),
		CreatedAt:         now,
		CreatedByRole:     createdByRole(req),
		RunID:             req.RunID,
		StepID:            req.StepID,
		StorageProtection: s.state.StorageProtectionPosture,
	}
}

func createdByRole(req PutRequest) string {
	if req.TrustedSource {
		return req.CreatedByRole
	}
	if req.CreatedByRole == "auditd" || req.CreatedByRole == "secretsd" || req.CreatedByRole == "launcher" {
		return "untrusted_client"
	}
	return req.CreatedByRole
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

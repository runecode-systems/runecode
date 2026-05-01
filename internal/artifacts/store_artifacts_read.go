package artifacts

import (
	"io"
	"sort"
)

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

func (s *Store) HasAuditEvents() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.LastAuditSequence > 0
}

package artifacts

import (
	"errors"
	"path/filepath"
)

func (s *Store) GarbageCollect() (GCResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.nowFn().UTC()
	ttl := ensureTTL(s.state.Policy.UnreferencedTTLSeconds)
	candidates := gcCandidates(s.state.Artifacts, s.state.Runs, s.state.DependencyCacheBatches, s.state.DependencyCacheUnits, now, ttl)
	result, err := s.deleteCandidatesLocked(candidates)
	if err != nil {
		return GCResult{}, err
	}
	if err := s.auditGCResultLocked(result); err != nil {
		return GCResult{}, err
	}
	return result, nil
}

func (s *Store) deleteCandidatesLocked(candidates []gcCandidate) (GCResult, error) {
	result := GCResult{}
	for _, c := range candidates {
		if err := s.storeIO.removeBlob(c.rec.BlobPath); err != nil {
			return result, err
		}
		delete(s.state.Artifacts, c.digest)
		purgeRunPlanAuthoritiesByDigestLocked(&s.state, c.digest)
		result.DeletedDigests = append(result.DeletedDigests, c.digest)
		result.FreedBytes += c.rec.Reference.SizeBytes
	}
	return result, nil
}

func (s *Store) DeleteDigest(digest string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.state.Artifacts[digest]
	if !ok {
		return nil
	}
	priorRunPlanAuthorities, priorRunPlanCompilations := runPlanRecordsForDigest(s.state, digest)
	delete(s.state.Artifacts, digest)
	purgeRunPlanAuthoritiesByDigestLocked(&s.state, digest)
	if err := s.saveStateLocked(); err != nil {
		restoreDeletedDigestState(&s.state, digest, record, priorRunPlanAuthorities, priorRunPlanCompilations)
		return err
	}
	if err := s.storeIO.removeBlob(record.BlobPath); err != nil {
		restoreDeletedDigestState(&s.state, digest, record, priorRunPlanAuthorities, priorRunPlanCompilations)
		rollbackErr := s.saveStateLocked()
		if rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return err
	}
	return nil
}

func runPlanRecordsForDigest(state StoreState, digest string) (map[string]RunPlanAuthorityRecord, map[string]RunPlanCompilationRecord) {
	priorRunPlanAuthorities := map[string]RunPlanAuthorityRecord{}
	priorRunPlanCompilations := map[string]RunPlanCompilationRecord{}
	for key, rec := range state.RunPlanAuthorities {
		if rec.RunPlanDigest != digest {
			continue
		}
		priorRunPlanAuthorities[key] = rec
	}
	for key, rec := range state.RunPlanCompilations {
		if _, ok := priorRunPlanAuthorities[key]; !ok {
			continue
		}
		priorRunPlanCompilations[key] = rec
	}
	return priorRunPlanAuthorities, priorRunPlanCompilations
}

func restoreDeletedDigestState(state *StoreState, digest string, record ArtifactRecord, priorRunPlanAuthorities map[string]RunPlanAuthorityRecord, priorRunPlanCompilations map[string]RunPlanCompilationRecord) {
	state.Artifacts[digest] = record
	for key, rec := range priorRunPlanAuthorities {
		state.RunPlanAuthorities[key] = rec
	}
	for key, rec := range priorRunPlanCompilations {
		state.RunPlanCompilations[key] = rec
	}
	rebuildRunPlanRefsByRunLocked(state)
}

func (s *Store) auditGCResultLocked(result GCResult) error {
	if len(result.DeletedDigests) == 0 {
		return nil
	}
	if err := s.appendAuditLocked("artifact_retention_action", "system", map[string]interface{}{"action": "gc", "deleted_digests": result.DeletedDigests, "freed_bytes": result.FreedBytes}); err != nil {
		return err
	}
	return s.saveStateLocked()
}

func (s *Store) ExportBackup(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path = sanitizeBackupPath(path)
	manifest := buildBackupManifest(s.state, s.nowFn().UTC())
	if err := s.storeIO.writeBackup(path, manifest); err != nil {
		return err
	}
	if err := s.storeIO.writeBackupBlobs(path, manifest.Artifacts); err != nil {
		return err
	}
	signature, err := computeBackupSignature(manifest, s.state.BackupHMACKey)
	if err != nil {
		return err
	}
	if err := s.storeIO.writeBackupSignature(backupSignaturePath(path), signature); err != nil {
		return err
	}
	if err := s.appendAuditLocked("artifact_retention_action", "system", map[string]interface{}{"action": "export_backup", "path": filepath.Base(path)}); err != nil {
		return err
	}
	return s.saveStateLocked()
}

func (s *Store) RestoreBackup(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path = sanitizeBackupPath(path)
	manifest, err := s.storeIO.readBackup(path)
	if err != nil {
		return err
	}
	signature, err := s.storeIO.readBackupSignature(backupSignaturePath(path))
	if err != nil {
		return err
	}
	if err := verifyBackupSignature(manifest, signature, s.state.BackupHMACKey); err != nil {
		return err
	}
	restoredDigests, err := s.storeIO.restoreBackupBlobsStaged(path, manifest.Artifacts)
	if err != nil {
		return err
	}
	next, err := stateFromBackup(manifest, s.state.LastAuditSequence, s.storeIO)
	if err != nil {
		rollbackRestoredBlobDigests(s.storeIO, restoredDigests)
		return err
	}
	next.BackupHMACKey = s.state.BackupHMACKey
	next = normalizeState(next)
	priorState := s.state
	s.state = next
	if err := s.appendAuditLocked("artifact_retention_action", "system", map[string]interface{}{"action": "restore_backup", "path": filepath.Base(path)}); err != nil {
		s.state = priorState
		rollbackRestoredBlobDigests(s.storeIO, restoredDigests)
		return err
	}
	return nil
}

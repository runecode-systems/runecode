package artifacts

import (
	"fmt"
	"sync"
	"time"
)

type Store struct {
	mu         sync.Mutex
	rootDir    string
	blobDir    string
	storeIO    *storeIO
	state      StoreState
	stateStamp stateFileStamp
	nowFn      func() time.Time
}

func NewStore(rootDir string) (*Store, error) {
	if rootDir == "" {
		return nil, fmt.Errorf("root dir is required")
	}
	blobDir := defaultBlobDir(rootDir)
	sio, err := newStoreIO(rootDir, blobDir)
	if err != nil {
		return nil, err
	}
	store := &Store{rootDir: rootDir, blobDir: blobDir, storeIO: sio, nowFn: time.Now}
	if err := store.loadState(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) loadState() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, err := s.storeIO.loadStateFile()
	if err != nil {
		return err
	}
	stamp, err := s.storeIO.stateFileStamp()
	if err != nil {
		return err
	}
	initialized, changed, err := s.initializeLoadedState(state)
	if err != nil {
		return err
	}
	s.state = initialized
	s.stateStamp = stamp

	changed, err = s.reconcileLoadedState(changed)
	if err != nil {
		return err
	}
	if changed {
		if err := s.saveStateLocked(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) initializeLoadedState(state StoreState) (StoreState, bool, error) {
	normalized := normalizeState(state)
	withKey, err := ensureBackupKey(normalized)
	if err != nil {
		return StoreState{}, false, err
	}
	return withKey, state.BackupHMACKey == "", nil
}

func (s *Store) reconcileLoadedState(changed bool) (bool, error) {
	sequenceChanged, err := s.reconcileAuditSequenceLocked()
	if err != nil {
		return false, err
	}
	changed = changed || sequenceChanged

	artifactIndexChanged, err := s.reconcileArtifactIndexFromAuditLocked()
	if err != nil {
		return false, err
	}
	changed = changed || artifactIndexChanged

	approvalLinkChanged, err := s.reconcileApprovalPolicyDecisionLinksLocked()
	if err != nil {
		return false, err
	}
	changed = changed || approvalLinkChanged

	preparedRefsChanged := reconcileRunGitRemotePreparedRefsLocked(&s.state)
	changed = changed || preparedRefsChanged

	runnerChanged, err := s.reconcileRunnerAdvisoryDurableStateLocked()
	if err != nil {
		return false, err
	}
	changed = changed || runnerChanged

	runPlanChanged := reconcileRunPlanIndexesLocked(&s.state)
	changed = changed || runPlanChanged
	return changed, nil
}

func (s *Store) reconcileAuditSequenceLocked() (bool, error) {
	events, err := s.storeIO.readAuditEvents()
	if err != nil {
		return false, err
	}
	var maxSeq int64
	for _, event := range events {
		if event.Seq > maxSeq {
			maxSeq = event.Seq
		}
	}
	if maxSeq <= s.state.LastAuditSequence {
		return false, nil
	}
	s.state.LastAuditSequence = maxSeq
	return true, nil
}

func (s *Store) saveStateLocked() error {
	stamp, err := s.storeIO.saveStateFile(s.state)
	if err != nil {
		return err
	}
	s.stateStamp = stamp
	return nil
}

func (s *Store) SyncExternalState() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.syncExternalStateLocked()
}

func (s *Store) syncExternalStateLocked() error {
	stamp, err := s.storeIO.stateFileStamp()
	if err != nil {
		return err
	}
	if !stamp.differsFrom(s.stateStamp) {
		return nil
	}
	state, err := s.storeIO.loadStateFile()
	if err != nil {
		return err
	}
	initialized, _, err := s.initializeLoadedState(state)
	if err != nil {
		return err
	}
	s.state = initialized
	s.stateStamp = stamp
	_, err = s.reconcileLoadedState(false)
	return err
}

func (s *Store) appendAuditLocked(eventType, actor string, details map[string]interface{}) error {
	s.state.LastAuditSequence++
	event := newAuditEvent(s.state.LastAuditSequence, eventType, actor, details, s.nowFn)
	if err := s.storeIO.appendAuditEvent(event); err != nil {
		s.state.LastAuditSequence--
		return err
	}
	if err := s.saveStateLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Store) ReadAuditEvents() ([]AuditEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.storeIO.readAuditEvents()
}

func (s *Store) AppendTrustedAuditEvent(eventType, actor string, details map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if eventType == "" {
		return fmt.Errorf("event type is required")
	}
	if actor == "" {
		return fmt.Errorf("actor is required")
	}
	return s.appendAuditLocked(eventType, actor, details)
}

func (s *Store) SetNowFuncForTests(nowFn func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if nowFn == nil {
		s.nowFn = time.Now
		return
	}
	s.nowFn = nowFn
}

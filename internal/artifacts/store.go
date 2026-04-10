package artifacts

import (
	"fmt"
	"sync"
	"time"
)

type Store struct {
	mu      sync.Mutex
	rootDir string
	blobDir string
	storeIO *storeIO
	state   StoreState
	nowFn   func() time.Time
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
	normalized := normalizeState(state)
	withKey, err := ensureBackupKey(normalized)
	if err != nil {
		return err
	}
	s.state = withKey
	changed := state.BackupHMACKey == ""
	sequenceChanged, err := s.reconcileAuditSequenceLocked()
	if err != nil {
		return err
	}
	changed = changed || sequenceChanged
	approvalLinkChanged, err := s.reconcileApprovalPolicyDecisionLinksLocked()
	if err != nil {
		return err
	}
	changed = changed || approvalLinkChanged
	runnerChanged, err := s.reconcileRunnerAdvisoryDurableStateLocked()
	if err != nil {
		return err
	}
	changed = changed || runnerChanged
	if changed {
		if err := s.saveStateLocked(); err != nil {
			return err
		}
	}
	return nil
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
	return s.storeIO.saveStateFile(s.state)
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
		actor = "trusted_component"
	}
	return s.appendAuditLocked(eventType, actor, details)
}

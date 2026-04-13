package secretsd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func (s *Service) loadState() error {
	path := filepath.Join(s.root, stateFileName)
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		s.state = state{Version: stateVersion, Secrets: map[string]secretRecord{}, Leases: map[string]leaseRecord{}}
		return s.persistState()
	}
	if err != nil {
		return err
	}
	loaded := state{}
	if err := json.Unmarshal(b, &loaded); err != nil {
		return fmt.Errorf("%w", ErrStateRecoveryFailed)
	}
	if err := s.validateState(&loaded); err != nil {
		return err
	}
	if loaded.Secrets == nil {
		loaded.Secrets = map[string]secretRecord{}
	}
	if loaded.Leases == nil {
		loaded.Leases = map[string]leaseRecord{}
	}
	s.state = loaded
	return nil
}

func (s *Service) validateState(st *state) error {
	if st.Version != stateVersion {
		return fmt.Errorf("%w", ErrStateRecoveryFailed)
	}
	if err := s.validateSecretRecords(st.Secrets); err != nil {
		return err
	}
	if err := validateLeaseRecords(st.Leases, st.Secrets); err != nil {
		return err
	}
	return nil
}

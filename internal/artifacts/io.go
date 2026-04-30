package artifacts

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type storeIO struct {
	statePath string
	auditPath string
	blobDir   string
}

func newStoreIO(rootDir, blobDir string) (*storeIO, error) {
	if err := os.MkdirAll(blobDir, 0o700); err != nil {
		return nil, err
	}
	return &storeIO{
		statePath: filepath.Join(rootDir, "state.json"),
		auditPath: filepath.Join(rootDir, "audit.log"),
		blobDir:   blobDir,
	}, nil
}

func (s *storeIO) loadStateFile() (StoreState, error) {
	state := StoreState{}
	b, err := os.ReadFile(s.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	if len(b) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(b, &state); err != nil {
		return state, err
	}
	return state, nil
}

func (s *storeIO) saveStateFile(state StoreState) error {
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.statePath, b, 0o600)
}

func (s *storeIO) appendAuditEvent(event AuditEvent) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.auditPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func (s *storeIO) readAuditEvents() ([]AuditEvent, error) {
	b, err := os.ReadFile(s.auditPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	out := make([]AuditEvent, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event AuditEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, nil
}

func (s *storeIO) blobPath(digest string) string {
	return filepath.Join(s.blobDir, strings.TrimPrefix(digest, "sha256:"))
}

func (s *storeIO) validatedBlobPath(digest string) (string, error) {
	if !isValidDigest(digest) {
		return "", ErrInvalidDigest
	}
	path := s.blobPath(digest)
	rel, err := filepath.Rel(s.blobDir, path)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", ErrInvalidDigest
	}
	return path, nil
}

func (s *storeIO) writeBlobIfMissing(digest string, payload []byte) (bool, error) {
	path := s.blobPath(digest)
	if _, err := os.Stat(path); err == nil {
		existing, readErr := s.readBlob(path)
		if readErr != nil {
			return false, readErr
		}
		if digestBytes(existing) != digest {
			return false, ErrInvalidDigest
		}
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return false, err
	}
	return true, nil
}

func (s *storeIO) createBlobTempFile() (*os.File, error) {
	return os.CreateTemp(s.blobDir, ".blob-tmp-*")
}

func (s *storeIO) persistBlobFromTempFile(tmpPath string, digest string) (bool, error) {
	path := s.blobPath(digest)
	if _, err := os.Stat(path); err == nil {
		existing, readErr := s.readBlob(path)
		if readErr != nil {
			return false, readErr
		}
		if digestBytes(existing) != digest {
			return false, ErrInvalidDigest
		}
		if removeErr := os.Remove(tmpPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return false, removeErr
		}
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return false, err
	}
	return true, nil
}

func (s *storeIO) streamToTempBlob(r io.Reader) (tmpPath string, digest string, size int64, err error) {
	tmpFile, err := s.createBlobTempFile()
	if err != nil {
		return "", "", 0, err
	}
	tmpPath = tmpFile.Name()
	h := newDigestWriter()
	written, copyErr := io.Copy(io.MultiWriter(tmpFile, h), r)
	closeErr := tmpFile.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return "", "", 0, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return "", "", 0, closeErr
	}
	return tmpPath, h.identity(), written, nil
}

func (s *storeIO) openBlob(path string) (*os.File, error) {
	return os.Open(path)
}

func (s *storeIO) readBlob(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (s *storeIO) verifyBlobDigestAndSize(path string, expectedDigest string, expectedSize int64) error {
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	h := newDigestWriter()
	written, err := io.Copy(h, in)
	if err != nil {
		return err
	}
	if h.identity() != expectedDigest {
		return fmt.Errorf("backup digest mismatch for %s", expectedDigest)
	}
	if written != expectedSize {
		return fmt.Errorf("backup size mismatch for %s", expectedDigest)
	}
	return nil
}

func (s *storeIO) removeBlob(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func newAuditEvent(seq int64, eventType, actor string, details map[string]interface{}, nowFn func() time.Time) AuditEvent {
	return AuditEvent{Seq: seq, Type: eventType, OccurredAt: nowFn().UTC(), Actor: actor, Details: details}
}

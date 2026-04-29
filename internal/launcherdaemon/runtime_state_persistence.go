package launcherdaemon

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// writeRuntimeStateFile atomically stages data in a same-directory temp file,
// then replaces destination content in a Windows-safe way.
func writeRuntimeStateFile(path string, tempPattern string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), tempPattern)
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := writeRuntimeStateData(tmp, data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := replaceRuntimeStateFile(tmpPath, path); err != nil {
		if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
			return fmt.Errorf("destination exists as directory")
		}
		return err
	}
	return nil
}

func writeRuntimeStateData(w io.Writer, data []byte) error {
	written, err := io.Copy(w, bytes.NewReader(data))
	if err != nil {
		return err
	}
	if written != int64(len(data)) {
		return io.ErrShortWrite
	}
	return nil
}

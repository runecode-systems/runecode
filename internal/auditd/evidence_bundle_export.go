package auditd

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

const (
	auditEvidenceBundleArchiveFormatTar = "tar"
)

func (l *Ledger) ExportEvidenceBundle(req AuditEvidenceBundleExportRequest) (AuditEvidenceBundleExport, error) {
	if l == nil {
		return AuditEvidenceBundleExport{}, fmt.Errorf("ledger is required")
	}
	archiveFormat := normalizeEvidenceBundleArchiveFormat(req.ArchiveFormat)
	if archiveFormat != auditEvidenceBundleArchiveFormatTar {
		return AuditEvidenceBundleExport{}, fmt.Errorf("unsupported archive format %q", archiveFormat)
	}
	manifest, err := l.BuildEvidenceBundleManifest(req.ManifestRequest)
	if err != nil {
		return AuditEvidenceBundleExport{}, err
	}
	reader, writer := io.Pipe()
	go func() {
		err := l.streamEvidenceBundleTar(manifest, writer)
		_ = writer.CloseWithError(err)
	}()
	return AuditEvidenceBundleExport{Manifest: manifest, Reader: reader}, nil
}

func normalizeEvidenceBundleArchiveFormat(format string) string {
	trimmed := strings.TrimSpace(format)
	if trimmed == "" {
		return auditEvidenceBundleArchiveFormatTar
	}
	return trimmed
}

func (l *Ledger) streamEvidenceBundleTar(manifest AuditEvidenceBundleManifest, out io.Writer) error {
	tarWriter := tar.NewWriter(out)
	defer tarWriter.Close()
	manifestBytes, err := evidenceBundleCanonicalBytes(manifest)
	if err != nil {
		return err
	}
	if err := writeTarRegularFile(tarWriter, "manifest.json", manifestBytes); err != nil {
		return err
	}
	for i := range manifest.IncludedObjects {
		if err := l.writeEvidenceBundleObjectToTar(manifest.IncludedObjects[i], tarWriter); err != nil {
			return err
		}
	}
	return nil
}

func (l *Ledger) writeEvidenceBundleObjectToTar(object AuditEvidenceBundleIncludedObject, tarWriter *tar.Writer) error {
	cleanPath := strings.TrimSpace(object.Path)
	if cleanPath == "" {
		return fmt.Errorf("bundle object path is required")
	}
	if object.ObjectFamily == "audit_segment" {
		return l.writeEvidenceBundleSegmentObjectToTar(object, cleanPath, tarWriter)
	}
	absolute, err := l.bundleObjectAbsolutePathLocked(cleanPath)
	if err != nil {
		return err
	}
	return writeTarFileFromDisk(tarWriter, cleanPath, absolute, object.ByteLength)
}

func (l *Ledger) writeEvidenceBundleSegmentObjectToTar(object AuditEvidenceBundleIncludedObject, cleanPath string, tarWriter *tar.Writer) error {
	segmentID, ok := segmentIDFromPath(cleanPath)
	if !ok {
		return fmt.Errorf("invalid segment bundle path %q", cleanPath)
	}
	raw, err := l.segmentBundleObjectBytesLocked(segmentID)
	if err != nil {
		return err
	}
	if err := validateSegmentBundleObject(object, cleanPath, raw); err != nil {
		return err
	}
	return writeTarRegularFile(tarWriter, cleanPath, raw)
}

func (l *Ledger) segmentBundleObjectBytesLocked(segmentID string) ([]byte, error) {
	segment, err := l.loadSegment(segmentID)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(segment)
	if err != nil {
		return nil, err
	}
	return jsoncanonicalizer.Transform(b)
}

func validateSegmentBundleObject(object AuditEvidenceBundleIncludedObject, cleanPath string, raw []byte) error {
	segmentIdentity, err := digestIdentityFromSegmentPayload(raw)
	if err != nil {
		return err
	}
	if strings.TrimSpace(object.Digest) != segmentIdentity {
		return fmt.Errorf("bundle object digest mismatch for %q", cleanPath)
	}
	if object.ByteLength != int64(len(raw)) {
		return fmt.Errorf("bundle object byte_length mismatch for %q", cleanPath)
	}
	return nil
}

func writeTarFileFromDisk(tarWriter *tar.Writer, tarPath string, diskPath string, expectedSize int64) error {
	file, err := os.Open(diskPath)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	if expectedSize >= 0 && stat.Size() != expectedSize {
		return fmt.Errorf("bundle object byte_length mismatch for %q", tarPath)
	}
	header := &tar.Header{Name: filepath.ToSlash(tarPath), Mode: 0o600, Size: stat.Size(), ModTime: time.Unix(0, 0).UTC()}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(tarWriter, file)
	return err
}

func writeTarRegularFile(tarWriter *tar.Writer, tarPath string, content []byte) error {
	header := &tar.Header{Name: filepath.ToSlash(strings.TrimSpace(tarPath)), Mode: 0o600, Size: int64(len(content)), ModTime: time.Unix(0, 0).UTC()}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	_, err := tarWriter.Write(content)
	return err
}

func evidenceBundleCanonicalBytes(manifest AuditEvidenceBundleManifest) ([]byte, error) {
	b, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	return jsoncanonicalizer.Transform(b)
}

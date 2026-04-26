package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"path/filepath"
	"sort"
)

func defaultBlobDir(rootDir string) string {
	return filepath.Join(rootDir, "blobs")
}

func isValidDigest(digest string) bool {
	return digestPattern.MatchString(digest)
}

func digestBytes(payload []byte) string {
	h := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(h[:])
}

type digestWriter struct {
	h hash.Hash
}

func newDigestWriter() *digestWriter {
	return &digestWriter{h: sha256.New()}
}

func (w *digestWriter) Write(p []byte) (int, error) {
	return w.h.Write(p)
}

func (w *digestWriter) identity() string {
	return "sha256:" + hex.EncodeToString(w.h.Sum(nil))
}

func DigestBytes(payload []byte) string {
	return digestBytes(payload)
}

func canonicalPayload(contentType string, payload []byte) ([]byte, error) {
	if isJSONContentType(contentType) {
		canonical, err := canonicalizeJSONBytes(payload)
		if err != nil {
			return nil, fmt.Errorf("invalid json payload: %w", err)
		}
		return canonical, nil
	}
	return append([]byte(nil), payload...), nil
}

func isReservedDataClass(dataClass DataClass) bool {
	return dataClass == DataClassWebQuery || dataClass == DataClassWebCitations
}

func isDependencyDataClass(dataClass DataClass) bool {
	switch dataClass {
	case DataClassDependencyBatchManifest, DataClassDependencyResolvedUnit, DataClassDependencyPayloadUnit, DataClassDependencyMaterialized:
		return true
	default:
		return false
	}
}

func buildArtifactReference(digest string, size int64, req PutRequest) ArtifactReference {
	return ArtifactReference{
		Digest:                digest,
		SizeBytes:             size,
		ContentType:           req.ContentType,
		DataClass:             req.DataClass,
		ProvenanceReceiptHash: req.ProvenanceReceiptHash,
	}
}

func validateRunStatusInput(runID, status string) error {
	if runID == "" {
		return fmt.Errorf("run id is required")
	}
	if !isSupportedRunStatus(status) {
		return fmt.Errorf("unsupported run status")
	}
	return nil
}

func isSupportedRunStatus(status string) bool {
	switch status {
	case "pending", "starting", "active", "blocked", "recovering", "completed", "failed", "cancelled", "retained", "closed":
		return true
	default:
		return false
	}
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	clone := append([]string{}, values...)
	sort.Strings(clone)
	out := make([]string, 0, len(clone))
	for _, v := range clone {
		if len(out) == 0 || out[len(out)-1] != v {
			out = append(out, v)
		}
	}
	return out
}

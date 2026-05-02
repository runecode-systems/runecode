package auditd

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
)

type offlineBundleObject struct {
	path    string
	content []byte
}

type offlineBundleSnapshot struct {
	manifest               AuditEvidenceBundleManifest
	manifestCanonicalJSON  []byte
	manifestDigestIdentity string
	objects                map[string]offlineBundleObject
}

func loadAuditEvidenceBundleFromTar(reader io.Reader) (offlineBundleSnapshot, error) {
	objects, err := loadOfflineBundleTarObjects(reader)
	if err != nil {
		return offlineBundleSnapshot{}, err
	}
	return offlineBundleSnapshotFromObjects(objects)
}

func loadOfflineBundleTarObjects(reader io.Reader) (map[string]offlineBundleObject, error) {
	objects := map[string]offlineBundleObject{}
	tarReader := tar.NewReader(reader)
	for {
		done, err := loadNextOfflineBundleTarObject(tarReader, objects)
		if err != nil {
			return nil, err
		}
		if done {
			return objects, nil
		}
	}
}

func loadNextOfflineBundleTarObject(tarReader *tar.Reader, objects map[string]offlineBundleObject) (bool, error) {
	header, err := tarReader.Next()
	if err == io.EOF {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	cleanPath, ok := offlineBundleTarRegularPath(header)
	if !ok {
		return false, nil
	}
	if err := ensureOfflineBundlePathAbsent(objects, cleanPath); err != nil {
		return false, err
	}
	payload, err := io.ReadAll(tarReader)
	if err != nil {
		return false, err
	}
	objects[cleanPath] = offlineBundleObject{path: cleanPath, content: payload}
	return false, nil
}

func offlineBundleTarRegularPath(header *tar.Header) (string, bool) {
	if header == nil || header.Typeflag != tar.TypeReg {
		return "", false
	}
	cleanPath := filepathToBundlePath(header.Name)
	if cleanPath == "" {
		return "", false
	}
	return cleanPath, true
}

func ensureOfflineBundlePathAbsent(objects map[string]offlineBundleObject, cleanPath string) error {
	if _, exists := objects[cleanPath]; exists {
		return fmt.Errorf("bundle contains duplicate path %q", cleanPath)
	}
	return nil
}

func offlineBundleSnapshotFromObjects(objects map[string]offlineBundleObject) (offlineBundleSnapshot, error) {
	manifestObject, ok := objects["manifest.json"]
	if !ok {
		return offlineBundleSnapshot{}, fmt.Errorf("bundle manifest.json missing")
	}
	manifest := AuditEvidenceBundleManifest{}
	if err := json.Unmarshal(manifestObject.content, &manifest); err != nil {
		return offlineBundleSnapshot{}, fmt.Errorf("bundle manifest decode failed: %w", err)
	}
	if strings.TrimSpace(manifest.SchemaID) != auditEvidenceBundleManifestSchemaID || strings.TrimSpace(manifest.SchemaVersion) != auditEvidenceBundleManifestSchemaVersion {
		return offlineBundleSnapshot{}, fmt.Errorf("bundle manifest schema mismatch")
	}
	manifestCanonicalJSON, err := evidenceBundleCanonicalBytes(manifest)
	if err != nil {
		return offlineBundleSnapshot{}, err
	}
	manifestDigest, err := canonicalDigest(manifest)
	if err != nil {
		return offlineBundleSnapshot{}, err
	}
	manifestDigestIdentity, _ := manifestDigest.Identity()
	return offlineBundleSnapshot{manifest: manifest, manifestCanonicalJSON: manifestCanonicalJSON, manifestDigestIdentity: manifestDigestIdentity, objects: objects}, nil
}

func filepathToBundlePath(name string) string {
	clean := path.Clean(strings.TrimSpace(strings.ReplaceAll(name, "\\", "/")))
	if clean == "." || clean == "" || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return ""
	}
	return clean
}

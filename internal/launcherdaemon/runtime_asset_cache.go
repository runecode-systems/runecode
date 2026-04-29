package launcherdaemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

const verifiedRuntimeCacheDir = "verified-runtime-cache"

const verifiedRuntimeAdmissionDir = "admissions"

type admittedRuntimeImage struct {
	componentPaths  map[string]string
	cacheEvidence   *launcherbackend.BackendCacheEvidence
	admissionRecord launcherbackend.RuntimeAdmissionRecord
	toolchain       *verifiedRuntimeToolchainDescriptor
}

func admitRuntimeImage(workRoot string, image launcherbackend.RuntimeImageDescriptor) (admittedRuntimeImage, error) {
	if err := image.Validate(); err != nil {
		return admittedRuntimeImage{}, backendError(launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch, err.Error())
	}
	cacheRoot := verifiedRuntimeCacheRoot(workRoot)
	admissionRecord, toolchainDescriptor, err := buildRuntimeAdmissionRecord(cacheRoot, image)
	if err != nil {
		return admittedRuntimeImage{}, backendError(launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch, err.Error())
	}
	cacheResult, err := ensureRuntimeAdmissionRecord(cacheRoot, image, admissionRecord)
	if err != nil {
		return admittedRuntimeImage{}, backendError(launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch, err.Error())
	}
	componentPaths, resolvedDigests, err := resolveRuntimeImageComponents(cacheRoot, image.ComponentDigests)
	if err != nil {
		return admittedRuntimeImage{}, backendError(launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch, err.Error())
	}
	return admittedRuntimeImage{
		componentPaths:  componentPaths,
		admissionRecord: admissionRecord,
		toolchain:       toolchainDescriptor,
		cacheEvidence: &launcherbackend.BackendCacheEvidence{
			ImageCacheResult:              cacheResult,
			BootArtifactCacheResult:       launcherbackend.CacheResultHit,
			ResolvedImageDescriptorDigest: image.DescriptorDigest,
			ResolvedBootComponentDigests:  resolvedDigests,
		},
	}, nil
}

func buildRuntimeAdmissionRecord(cacheRoot string, image launcherbackend.RuntimeImageDescriptor) (launcherbackend.RuntimeAdmissionRecord, *verifiedRuntimeToolchainDescriptor, error) {
	admissionRecord, err := launcherbackend.NewRuntimeAdmissionRecord(image)
	if err != nil {
		return launcherbackend.RuntimeAdmissionRecord{}, nil, err
	}
	toolchainDescriptor, err := verifyRuntimeToolchainSignature(cacheRoot, image)
	if err != nil {
		return launcherbackend.RuntimeAdmissionRecord{}, nil, err
	}
	if toolchainDescriptor != nil {
		admissionRecord.RuntimeToolchainBundleDigest = toolchainDescriptor.PublicationBundleDigest
	}
	return admissionRecord, toolchainDescriptor, nil
}

func ensureRuntimeAdmissionRecord(cacheRoot string, image launcherbackend.RuntimeImageDescriptor, admissionRecord launcherbackend.RuntimeAdmissionRecord) (string, error) {
	persistedRecord, found, err := loadRuntimeAdmissionRecord(cacheRoot, image.DescriptorDigest)
	if err != nil {
		return "", err
	}
	if found {
		if !reflect.DeepEqual(persistedRecord, admissionRecord) {
			return "", fmt.Errorf("persisted runtime admission record does not match image identity")
		}
	} else if err := persistVerifiedRuntimeAdmission(cacheRoot, image, admissionRecord); err != nil {
		return "", err
	}
	if err := verifyRuntimeImageSignature(cacheRoot, image); err != nil {
		return "", err
	}
	return cacheResultForAdmission(found), nil
}

func persistVerifiedRuntimeAdmission(cacheRoot string, image launcherbackend.RuntimeImageDescriptor, admissionRecord launcherbackend.RuntimeAdmissionRecord) error {
	if err := verifyRuntimeImageSignature(cacheRoot, image); err != nil {
		return err
	}
	return persistRuntimeAdmissionRecord(cacheRoot, admissionRecord)
}

func resolveRuntimeImageComponents(cacheRoot string, digests map[string]string) (map[string]string, []string, error) {
	componentPaths := make(map[string]string, len(digests))
	resolvedDigests := make([]string, 0, len(digests))
	for name, digest := range digests {
		assetPath, err := resolveVerifiedRuntimeAsset(cacheRoot, digest)
		if err != nil {
			return nil, nil, fmt.Errorf("verified runtime asset %s unavailable", name)
		}
		componentPaths[name] = assetPath
		resolvedDigests = append(resolvedDigests, digest)
	}
	return componentPaths, resolvedDigests, nil
}

func verifyRuntimeImageSignature(cacheRoot string, image launcherbackend.RuntimeImageDescriptor) error {
	signaturePath, err := resolveVerifiedRuntimeAsset(cacheRoot, image.Signing.SignatureDigest)
	if err != nil {
		return fmt.Errorf("signature material unavailable")
	}
	envelope, err := readSignedEnvelope(signaturePath)
	if err != nil {
		return err
	}
	registry, err := loadAuthorizedRuntimeVerifierRegistry(cacheRoot, image.Signing.VerifierSetRef, runtimeVerifierKindImage)
	if err != nil {
		return err
	}
	if err := trustpolicy.VerifySignedEnvelope(envelope, registry, trustpolicy.EnvelopeVerificationOptions{
		RequirePayloadSchemaMatch: true,
		ExpectedPayloadSchemaID:   launcherbackend.RuntimeImageSignedPayloadSchemaID,
		ExpectedPayloadVersion:    launcherbackend.RuntimeImageSignedPayloadSchemaVersion,
	}); err != nil {
		return fmt.Errorf("runtime image signature verification failed: %w", err)
	}
	expectedPayload, err := image.SignedPayloadCanonicalBytes()
	if err != nil {
		return err
	}
	canonicalPayload, err := jsoncanonicalizer.Transform(envelope.Payload)
	if err != nil {
		return fmt.Errorf("runtime image payload canonicalization failed: %w", err)
	}
	if !bytes.Equal(expectedPayload, canonicalPayload) {
		return fmt.Errorf("runtime image signed payload does not match descriptor identity")
	}
	return nil
}

func isDigestFormat(value string) bool {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 || parts[0] != "sha256" || len(parts[1]) != 64 {
		return false
	}
	for _, ch := range parts[1] {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

func readSignedEnvelope(path string) (trustpolicy.SignedObjectEnvelope, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	envelope := trustpolicy.SignedObjectEnvelope{}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, fmt.Errorf("decode signed envelope: %w", err)
	}
	return envelope, nil
}

func verifiedRuntimeCacheRoot(workRoot string) string {
	root := strings.TrimSpace(workRoot)
	if root == "" {
		root = filepath.Join(os.TempDir(), "runecode-launcher")
	}
	return filepath.Join(root, verifiedRuntimeCacheDir)
}

func resolveVerifiedRuntimeAsset(cacheRoot string, digest string) (string, error) {
	parts := strings.SplitN(strings.TrimSpace(digest), ":", 2)
	if len(parts) != 2 || parts[0] != "sha256" || parts[1] == "" {
		return "", fmt.Errorf("invalid digest")
	}
	path := filepath.Join(cacheRoot, parts[0], parts[1])
	computed, err := digestFileSHA256(path)
	if err != nil {
		return "", err
	}
	if computed != digest {
		return "", fmt.Errorf("runtime asset digest mismatch")
	}
	return path, nil
}

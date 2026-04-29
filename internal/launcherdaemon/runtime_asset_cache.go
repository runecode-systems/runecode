package launcherdaemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

const verifiedRuntimeCacheDir = "verified-runtime-cache"

type admittedRuntimeImage struct {
	componentPaths map[string]string
	cacheEvidence  *launcherbackend.BackendCacheEvidence
}

func admitRuntimeImage(workRoot string, image launcherbackend.RuntimeImageDescriptor) (admittedRuntimeImage, error) {
	if err := image.Validate(); err != nil {
		return admittedRuntimeImage{}, backendError(launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch, err.Error())
	}
	cacheRoot := verifiedRuntimeCacheRoot(workRoot)
	if err := verifyRuntimeImageSignature(cacheRoot, image); err != nil {
		return admittedRuntimeImage{}, backendError(launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch, err.Error())
	}
	componentPaths := make(map[string]string, len(image.ComponentDigests))
	resolvedDigests := make([]string, 0, len(image.ComponentDigests))
	for name, digest := range image.ComponentDigests {
		assetPath, err := resolveVerifiedRuntimeAsset(cacheRoot, digest)
		if err != nil {
			return admittedRuntimeImage{}, backendError(launcherbackend.BackendErrorCodeImageDescriptorSignatureMismatch, fmt.Sprintf("verified runtime asset %s unavailable", name))
		}
		componentPaths[name] = assetPath
		resolvedDigests = append(resolvedDigests, digest)
	}
	return admittedRuntimeImage{
		componentPaths: componentPaths,
		cacheEvidence: &launcherbackend.BackendCacheEvidence{
			ImageCacheResult:              launcherbackend.CacheResultHit,
			BootArtifactCacheResult:       launcherbackend.CacheResultHit,
			ResolvedImageDescriptorDigest: image.DescriptorDigest,
			ResolvedBootComponentDigests:  resolvedDigests,
		},
	}, nil
}

func verifyRuntimeImageSignature(cacheRoot string, image launcherbackend.RuntimeImageDescriptor) error {
	signaturePath, err := resolveVerifiedRuntimeAsset(cacheRoot, image.Signing.SignatureDigest)
	if err != nil {
		return fmt.Errorf("signature material unavailable")
	}
	verifierSetPath, err := resolveVerifiedRuntimeAsset(cacheRoot, image.Signing.VerifierSetRef)
	if err != nil {
		return fmt.Errorf("verifier set unavailable")
	}
	envelope, err := readSignedEnvelope(signaturePath)
	if err != nil {
		return err
	}
	registry, err := readVerifierRegistry(verifierSetPath)
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

func readVerifierRegistry(path string) (*trustpolicy.VerifierRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var records []trustpolicy.VerifierRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("decode verifier set: %w", err)
	}
	registry, err := trustpolicy.NewVerifierRegistry(records)
	if err != nil {
		return nil, fmt.Errorf("load verifier set: %w", err)
	}
	return registry, nil
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

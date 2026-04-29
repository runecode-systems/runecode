package launcherdaemon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

type verifiedRuntimeToolchainDescriptor struct {
	SchemaID                string            `json:"schema_id"`
	SchemaVersion           string            `json:"schema_version"`
	DescriptorDigest        string            `json:"descriptor_digest"`
	ToolchainFamily         string            `json:"toolchain_family"`
	ToolchainVersion        string            `json:"toolchain_version"`
	ArtifactDigests         map[string]string `json:"artifact_digests"`
	PublicationBundleDigest string            `json:"publication_bundle_digest,omitempty"`
}

func verifyRuntimeToolchainSignature(cacheRoot string, image launcherbackend.RuntimeImageDescriptor) (*verifiedRuntimeToolchainDescriptor, error) {
	if image.Signing == nil || image.Signing.Toolchain == nil {
		return nil, nil
	}
	toolchain := image.Signing.Toolchain
	envelope, err := loadSignedToolchainEnvelope(cacheRoot, toolchain)
	if err != nil {
		return nil, err
	}
	descriptor, err := decodeVerifiedToolchainDescriptor(envelope)
	if err != nil {
		return nil, err
	}
	if err := validateSignedToolchainBinding(toolchain, *descriptor, envelope.Payload); err != nil {
		return nil, err
	}
	return descriptor, nil
}

func loadSignedToolchainEnvelope(cacheRoot string, toolchain *launcherbackend.RuntimeToolchainSigningHooks) (trustpolicy.SignedObjectEnvelope, error) {
	signaturePath, err := resolveVerifiedRuntimeAsset(cacheRoot, toolchain.SignatureDigest)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, fmt.Errorf("toolchain signature material unavailable")
	}
	envelope, err := readSignedEnvelope(signaturePath)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	registry, err := loadAuthorizedRuntimeVerifierRegistry(cacheRoot, toolchain.VerifierSetRef, runtimeVerifierKindToolchain)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	if err := trustpolicy.VerifySignedEnvelope(envelope, registry, trustpolicy.EnvelopeVerificationOptions{
		RequirePayloadSchemaMatch: true,
		ExpectedPayloadSchemaID:   launcherbackend.RuntimeToolchainDescriptorSchemaID,
		ExpectedPayloadVersion:    launcherbackend.RuntimeToolchainDescriptorSchemaVersion,
	}); err != nil {
		return trustpolicy.SignedObjectEnvelope{}, fmt.Errorf("runtime toolchain signature verification failed: %w", err)
	}
	return envelope, nil
}

func decodeVerifiedToolchainDescriptor(envelope trustpolicy.SignedObjectEnvelope) (*verifiedRuntimeToolchainDescriptor, error) {
	descriptor := verifiedRuntimeToolchainDescriptor{}
	if err := json.Unmarshal(envelope.Payload, &descriptor); err != nil {
		return nil, fmt.Errorf("runtime toolchain payload decode failed: %w", err)
	}
	if err := validateRuntimeToolchainDescriptor(descriptor); err != nil {
		return nil, err
	}
	return &descriptor, nil
}

func validateSignedToolchainBinding(toolchain *launcherbackend.RuntimeToolchainSigningHooks, descriptor verifiedRuntimeToolchainDescriptor, payload []byte) error {
	if strings.TrimSpace(descriptor.DescriptorDigest) != strings.TrimSpace(toolchain.DescriptorDigest) {
		return fmt.Errorf("runtime toolchain signed payload does not match descriptor identity")
	}
	if digest, err := canonicalToolchainDescriptorDigest(payload); err != nil {
		return err
	} else if digest != descriptor.DescriptorDigest {
		return fmt.Errorf("runtime toolchain descriptor digest does not match canonical payload")
	}
	if strings.TrimSpace(toolchain.BundleDigest) != "" && strings.TrimSpace(toolchain.BundleDigest) != strings.TrimSpace(descriptor.PublicationBundleDigest) {
		return fmt.Errorf("runtime toolchain publication bundle digest does not match signed payload")
	}
	return nil
}

func validateRuntimeToolchainDescriptor(descriptor verifiedRuntimeToolchainDescriptor) error {
	if err := validateRuntimeToolchainDescriptorIdentity(descriptor); err != nil {
		return err
	}
	if err := validateRuntimeToolchainDescriptorArtifacts(descriptor.ArtifactDigests); err != nil {
		return err
	}
	if strings.TrimSpace(descriptor.PublicationBundleDigest) != "" && !isDigestFormat(descriptor.PublicationBundleDigest) {
		return fmt.Errorf("runtime toolchain payload publication bundle digest is invalid")
	}
	return nil
}

func validateRuntimeToolchainDescriptorIdentity(descriptor verifiedRuntimeToolchainDescriptor) error {
	if !isDigestFormat(descriptor.DescriptorDigest) {
		return fmt.Errorf("runtime toolchain payload descriptor digest is invalid")
	}
	if strings.TrimSpace(descriptor.SchemaID) != launcherbackend.RuntimeToolchainDescriptorSchemaID || strings.TrimSpace(descriptor.SchemaVersion) != launcherbackend.RuntimeToolchainDescriptorSchemaVersion {
		return fmt.Errorf("runtime toolchain payload schema identity is invalid")
	}
	if strings.TrimSpace(descriptor.ToolchainFamily) == "" || strings.TrimSpace(descriptor.ToolchainVersion) == "" {
		return fmt.Errorf("runtime toolchain payload requires toolchain family and version")
	}
	return nil
}

func validateRuntimeToolchainDescriptorArtifacts(artifactDigests map[string]string) error {
	if len(artifactDigests) == 0 {
		return fmt.Errorf("runtime toolchain payload requires artifact digests")
	}
	for name, digest := range artifactDigests {
		if strings.TrimSpace(name) == "" || !isDigestFormat(digest) {
			return fmt.Errorf("runtime toolchain payload artifact digests are invalid")
		}
	}
	return nil
}

func canonicalToolchainDescriptorDigest(payload []byte) (string, error) {
	fields := map[string]any{}
	if err := json.Unmarshal(payload, &fields); err != nil {
		return "", fmt.Errorf("runtime toolchain payload decode failed: %w", err)
	}
	delete(fields, "descriptor_digest")
	raw, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("runtime toolchain payload re-encode failed: %w", err)
	}
	canonicalPayload, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		return "", fmt.Errorf("runtime toolchain payload canonicalization failed: %w", err)
	}
	sum := sha256.Sum256(canonicalPayload)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func verifyRuntimeToolchainArtifact(path string, descriptor *verifiedRuntimeToolchainDescriptor) error {
	if descriptor == nil {
		return nil
	}
	artifactName := filepath.Base(strings.TrimSpace(path))
	expectedDigest := strings.TrimSpace(descriptor.ArtifactDigests[artifactName])
	if expectedDigest == "" {
		return fmt.Errorf("runtime toolchain signed payload does not cover launched artifact %q", artifactName)
	}
	actualDigest, err := digestFileSHA256(path)
	if err != nil {
		return fmt.Errorf("runtime toolchain artifact digest failed: %w", err)
	}
	if actualDigest != expectedDigest {
		return fmt.Errorf("runtime toolchain artifact digest mismatch")
	}
	return nil
}

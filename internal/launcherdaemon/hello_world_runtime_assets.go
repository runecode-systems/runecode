package launcherdaemon

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func seedHelloWorldToolchainVerificationAssets(cacheRoot string, toolchain *launcherbackend.RuntimeToolchainSigningHooks, qemuBinary string, signer helloWorldSignerMaterial) error {
	if toolchain == nil {
		return nil
	}
	toolchain.VerifierSetRef = signer.verifierSetRef
	qemuDigest, err := helloWorldQEMUDigest(qemuBinary)
	if err != nil {
		return err
	}
	canonicalPayload, descriptorDigest, err := helloWorldToolchainPayload(qemuDigest)
	if err != nil {
		return err
	}
	toolchain.DescriptorDigest = descriptorDigest
	envelopeBlob, err := marshalSignedEnvelope(launcherbackend.RuntimeToolchainDescriptorSchemaID, launcherbackend.RuntimeToolchainDescriptorSchemaVersion, canonicalPayload, signer.privateKey, signer.record.KeyIDValue)
	if err != nil {
		return err
	}
	signatureDigest, err := seedHelloWorldRuntimeAsset(cacheRoot, envelopeBlob)
	if err != nil {
		return err
	}
	toolchain.SignatureDigest = signatureDigest
	return nil
}

func helloWorldQEMUDigest(qemuBinary string) (string, error) {
	qemuDigest, err := digestFileSHA256(qemuBinary)
	if err == nil {
		return qemuDigest, nil
	}
	if os.IsNotExist(err) {
		fallbackSum := sha256.Sum256([]byte(helloWorldQEMUFallbackFixture))
		return "sha256:" + hex.EncodeToString(fallbackSum[:]), nil
	}
	return "", fmt.Errorf("resolve hello-world qemu toolchain digest: %w", err)
}

func helloWorldToolchainPayload(qemuDigest string) ([]byte, string, error) {
	payload := map[string]any{
		"schema_id":         launcherbackend.RuntimeToolchainDescriptorSchemaID,
		"schema_version":    launcherbackend.RuntimeToolchainDescriptorSchemaVersion,
		"toolchain_family":  "qemu",
		"toolchain_version": "host-local",
		"artifact_digests":  map[string]string{"qemu-system-x86_64": qemuDigest},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	canonicalPayload, err := jsoncanonicalizer.Transform(payloadJSON)
	if err != nil {
		return nil, "", err
	}
	descriptorSum := sha256.Sum256(canonicalPayload)
	descriptorDigest := "sha256:" + hex.EncodeToString(descriptorSum[:])
	payload["descriptor_digest"] = descriptorDigest
	payloadJSON, err = json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	canonicalPayload, err = jsoncanonicalizer.Transform(payloadJSON)
	if err != nil {
		return nil, "", err
	}
	return canonicalPayload, descriptorDigest, nil
}

func marshalSignedEnvelope(schemaID string, schemaVersion string, payload []byte, privateKey ed25519.PrivateKey, keyIDValue string) ([]byte, error) {
	envelope := trustpolicy.SignedObjectEnvelope{
		SchemaID:             trustpolicy.EnvelopeSchemaID,
		SchemaVersion:        trustpolicy.EnvelopeSchemaVersion,
		PayloadSchemaID:      schemaID,
		PayloadSchemaVersion: schemaVersion,
		Payload:              json.RawMessage(payload),
		SignatureInput:       trustpolicy.SignatureInputProfile,
		Signature: trustpolicy.SignatureBlock{
			Alg:        "ed25519",
			KeyID:      trustpolicy.KeyIDProfile,
			KeyIDValue: keyIDValue,
			Signature:  base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payload)),
		},
	}
	return json.Marshal(envelope)
}

func seedHelloWorldRuntimeAsset(cacheRoot string, data []byte) (string, error) {
	sum := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	path := filepath.Join(cacheRoot, "sha256", hex.EncodeToString(sum[:]))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if existing, err := os.ReadFile(path); err == nil {
		existingSum := sha256.Sum256(existing)
		if "sha256:"+hex.EncodeToString(existingSum[:]) != digest {
			return "", fmt.Errorf("hello-world runtime cache asset digest mismatch")
		}
		return digest, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return digest, nil
}

func seedHelloWorldRuntimeAssetFile(cacheRoot string, sourcePath string) (string, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}
	return seedHelloWorldRuntimeAsset(cacheRoot, data)
}

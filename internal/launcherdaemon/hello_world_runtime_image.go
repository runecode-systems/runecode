package launcherdaemon

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	helloWorldKernelFixture       = "runecode-hello-world-kernel-linux-amd64-v1"
	helloWorldInitrdFixture       = "runecode-hello-world-initrd-linux-amd64-v1"
	helloWorldDefaultQEMUBinary   = "/usr/bin/qemu-system-x86_64"
	helloWorldQEMUFallbackFixture = "runecode-hello-world-qemu-system-x86_64-fallback-v1"
	helloWorldAuthorityChangedAt  = "2026-04-29T00:00:00Z"
	helloWorldAuthorityReason     = "launcher hello-world local authority"
)

// PrepareHelloWorldRuntimeImageForLaunch seeds deterministic runtime assets in
// the verified runtime cache and returns an image descriptor that can pass
// admitRuntimeImage without bypassing signed admission checks.
func PrepareHelloWorldRuntimeImageForLaunch(workRoot string) (launcherbackend.RuntimeImageDescriptor, error) {
	if strings.TrimSpace(workRoot) == "" {
		return launcherbackend.RuntimeImageDescriptor{}, fmt.Errorf("hello-world launch requires a non-empty work root")
	}
	cacheRoot := verifiedRuntimeCacheRoot(workRoot)
	image, imageSigner, toolchainSigner, err := prepareHelloWorldImageForSigning(workRoot, cacheRoot)
	if err != nil {
		return launcherbackend.RuntimeImageDescriptor{}, err
	}
	payloadBytes, err := image.SignedPayloadCanonicalBytes()
	if err != nil {
		return launcherbackend.RuntimeImageDescriptor{}, err
	}
	envelopeBlob, err := marshalSignedEnvelope(launcherbackend.RuntimeImageSignedPayloadSchemaID, launcherbackend.RuntimeImageSignedPayloadSchemaVersion, payloadBytes, imageSigner.privateKey, imageSigner.record.KeyIDValue)
	if err != nil {
		return launcherbackend.RuntimeImageDescriptor{}, err
	}
	signatureDigest, err := seedHelloWorldRuntimeAsset(cacheRoot, envelopeBlob)
	if err != nil {
		return launcherbackend.RuntimeImageDescriptor{}, err
	}
	image.Signing.SignatureDigest = signatureDigest
	if err := seedHelloWorldToolchainVerificationAssets(cacheRoot, image.Signing.Toolchain, strings.TrimSpace(helloWorldDefaultQEMUBinary), toolchainSigner); err != nil {
		return launcherbackend.RuntimeImageDescriptor{}, err
	}
	if err := image.Validate(); err != nil {
		return launcherbackend.RuntimeImageDescriptor{}, err
	}
	return image, nil
}

func prepareHelloWorldImageForSigning(workRoot string, cacheRoot string) (launcherbackend.RuntimeImageDescriptor, helloWorldSignerMaterial, helloWorldSignerMaterial, error) {
	imageSigner, toolchainSigner, err := ensureHelloWorldAuthorityState(workRoot)
	if err != nil {
		return launcherbackend.RuntimeImageDescriptor{}, helloWorldSignerMaterial{}, helloWorldSignerMaterial{}, err
	}
	kernelDigest, err := seedHelloWorldRuntimeAsset(cacheRoot, []byte(helloWorldKernelFixture))
	if err != nil {
		return launcherbackend.RuntimeImageDescriptor{}, helloWorldSignerMaterial{}, helloWorldSignerMaterial{}, err
	}
	initrdDigest, err := seedHelloWorldRuntimeAsset(cacheRoot, []byte(helloWorldInitrdFixture))
	if err != nil {
		return launcherbackend.RuntimeImageDescriptor{}, helloWorldSignerMaterial{}, helloWorldSignerMaterial{}, err
	}
	image, err := buildHelloWorldRuntimeImage(kernelDigest, initrdDigest, imageSigner, toolchainSigner)
	if err != nil {
		return launcherbackend.RuntimeImageDescriptor{}, helloWorldSignerMaterial{}, helloWorldSignerMaterial{}, err
	}
	return image, imageSigner, toolchainSigner, nil
}

func buildHelloWorldRuntimeImage(kernelDigest string, initrdDigest string, imageSigner helloWorldSignerMaterial, toolchainSigner helloWorldSignerMaterial) (launcherbackend.RuntimeImageDescriptor, error) {
	image := launcherbackend.RuntimeImageDescriptor{
		BackendKind:         launcherbackend.BackendKindMicroVM,
		BootContractVersion: launcherbackend.BootProfileMicroVMLinuxKernelInitrdV1,
		PlatformCompatibility: launcherbackend.RuntimeImagePlatformCompat{
			OS: "linux", Architecture: "amd64", AccelerationKind: launcherbackend.AccelerationKindKVM,
		},
		ComponentDigests: map[string]string{"kernel": kernelDigest, "initrd": initrdDigest},
	}
	descriptorDigest, err := image.ExpectedDescriptorDigest()
	if err != nil {
		return launcherbackend.RuntimeImageDescriptor{}, err
	}
	image.Signing = &launcherbackend.RuntimeImageSigningHooks{
		PayloadSchemaID:      launcherbackend.RuntimeImageSignedPayloadSchemaID,
		PayloadSchemaVersion: launcherbackend.RuntimeImageSignedPayloadSchemaVersion,
		PayloadDigest:        descriptorDigest,
		SignerRef:            imageSigner.record.OwnerPrincipal.PrincipalID,
		VerifierSetRef:       imageSigner.verifierSetRef,
		Toolchain: &launcherbackend.RuntimeToolchainSigningHooks{
			DescriptorSchemaID:      launcherbackend.RuntimeToolchainDescriptorSchemaID,
			DescriptorSchemaVersion: launcherbackend.RuntimeToolchainDescriptorSchemaVersion,
			SignerRef:               toolchainSigner.record.OwnerPrincipal.PrincipalID,
			VerifierSetRef:          toolchainSigner.verifierSetRef,
		},
	}
	image.DescriptorDigest = descriptorDigest
	return image, nil
}

type helloWorldSignerMaterial struct {
	record         trustpolicy.VerifierRecord
	privateKey     ed25519.PrivateKey
	verifierSetRef string
}

func verifierRecordForHelloWorldSigner(publicKey ed25519.PublicKey, purpose string, principalID string) trustpolicy.VerifierRecord {
	keyIDValue := sha256HexStringHelloWorld(publicKey)
	return trustpolicy.VerifierRecord{
		SchemaID:      trustpolicy.VerifierSchemaID,
		SchemaVersion: trustpolicy.VerifierSchemaVersion,
		KeyID:         trustpolicy.KeyIDProfile,
		KeyIDValue:    keyIDValue,
		Alg:           "ed25519",
		PublicKey: trustpolicy.PublicKey{
			Encoding: "base64",
			Value:    base64.StdEncoding.EncodeToString(publicKey),
		},
		LogicalPurpose:         purpose,
		LogicalScope:           "publisher",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.1.0", ActorKind: "service", PrincipalID: principalID, InstanceID: "local"},
		KeyProtectionPosture:   "ephemeral_memory",
		IdentityBindingPosture: "tofu",
		PresenceMode:           "none",
		CreatedAt:              helloWorldAuthorityChangedAt,
		Status:                 "active",
	}
}

func authorityEntryForHelloWorldSigner(material helloWorldSignerMaterial) runtimeVerifierAuthorityEntry {
	return runtimeVerifierAuthorityEntry{
		VerifierSetRef: material.verifierSetRef,
		Records:        []trustpolicy.VerifierRecord{material.record},
		Status:         runtimeVerifierAuthorityStatusActive,
		Source:         runtimeVerifierAuthoritySourceImported,
		ChangedAt:      helloWorldAuthorityChangedAt,
		Reason:         helloWorldAuthorityReason,
	}
}

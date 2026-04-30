package launcherbackend

import (
	"fmt"
	"sort"
	"strings"
)

func (r BackendLaunchReceipt) Normalized() BackendLaunchReceipt {
	out := r
	normalizeReceiptCoreFields(&out)
	normalizeReceiptImageFields(&out)
	normalizeReceiptQEMUAndAttachmentFields(&out)
	normalizeReceiptProvisioningAndSessionSecurity(&out)
	normalizeReceiptRuntimePolicyFields(&out)
	return out
}

func normalizeReceiptCoreFields(receipt *BackendLaunchReceipt) {
	receipt.BackendKind = normalizeBackendKind(receipt.BackendKind)
	receipt.IsolationAssuranceLevel = normalizeIsolationAssuranceLevel(receipt.IsolationAssuranceLevel)
	receipt.ProvisioningPosture = normalizeProvisioningPostureForBackend(receipt.ProvisioningPosture, receipt.BackendKind)
	receipt.HypervisorImplementation = normalizeHypervisorImplementationForBackend(receipt.HypervisorImplementation, receipt.BackendKind)
	receipt.AccelerationKind = normalizeAccelerationKindForBackend(receipt.AccelerationKind, receipt.BackendKind)
	receipt.TransportKind = normalizeTransportKindForBackend(receipt.TransportKind, receipt.BackendKind)
	receipt.LaunchFailureReasonCode = normalizeBackendErrorCode(receipt.LaunchFailureReasonCode)
	receipt.SessionNonce = strings.TrimSpace(receipt.SessionNonce)
	receipt.LaunchContextDigest = strings.TrimSpace(receipt.LaunchContextDigest)
	receipt.HandshakeTranscriptHash = strings.TrimSpace(receipt.HandshakeTranscriptHash)
	receipt.IsolateSessionKeyIDValue = strings.TrimSpace(receipt.IsolateSessionKeyIDValue)
	receipt.HostingNodeID = strings.TrimSpace(receipt.HostingNodeID)
	normalizeReceiptAttestationFields(receipt)
}

func normalizeReceiptImageFields(receipt *BackendLaunchReceipt) {
	if strings.TrimSpace(receipt.RuntimeImageDescriptorDigest) == "" {
		receipt.RuntimeImageDescriptorDigest = strings.TrimSpace(receipt.RuntimeImageDigest)
	}
	if strings.TrimSpace(receipt.RuntimeImageDigest) == "" {
		receipt.RuntimeImageDigest = receipt.RuntimeImageDescriptorDigest
	}
	receipt.RuntimeImageBootProfile = normalizeBootProfile(receipt.RuntimeImageBootProfile)
	receipt.RuntimeImageSignerRef = strings.TrimSpace(receipt.RuntimeImageSignerRef)
	receipt.RuntimeImageVerifierRef = strings.TrimSpace(receipt.RuntimeImageVerifierRef)
	receipt.RuntimeImageSignatureDigest = strings.TrimSpace(receipt.RuntimeImageSignatureDigest)
	receipt.RuntimeToolchainDescriptorDigest = strings.TrimSpace(receipt.RuntimeToolchainDescriptorDigest)
	receipt.RuntimeToolchainSignerRef = strings.TrimSpace(receipt.RuntimeToolchainSignerRef)
	receipt.RuntimeToolchainVerifierRef = strings.TrimSpace(receipt.RuntimeToolchainVerifierRef)
	receipt.RuntimeToolchainSignatureDigest = strings.TrimSpace(receipt.RuntimeToolchainSignatureDigest)
	if receipt.RuntimeImageSignatureDigest != "" && !looksLikeDigest(receipt.RuntimeImageSignatureDigest) {
		receipt.RuntimeImageSignatureDigest = ""
	}
	if receipt.RuntimeToolchainDescriptorDigest != "" && !looksLikeDigest(receipt.RuntimeToolchainDescriptorDigest) {
		receipt.RuntimeToolchainDescriptorDigest = ""
	}
	if receipt.RuntimeToolchainSignatureDigest != "" && !looksLikeDigest(receipt.RuntimeToolchainSignatureDigest) {
		receipt.RuntimeToolchainSignatureDigest = ""
	}
	normalizeBootComponentDigestFields(receipt)
	if strings.TrimSpace(receipt.RuntimeImageDescriptorDigest) == "" {
		receipt.RuntimeImageBootProfile = ""
		receipt.RuntimeImageSignerRef = ""
		receipt.RuntimeImageVerifierRef = ""
		receipt.RuntimeImageSignatureDigest = ""
		receipt.RuntimeToolchainDescriptorDigest = ""
		receipt.RuntimeToolchainSignerRef = ""
		receipt.RuntimeToolchainVerifierRef = ""
		receipt.RuntimeToolchainSignatureDigest = ""
	}
}

func normalizeBootComponentDigestFields(receipt *BackendLaunchReceipt) {
	normalizeBootComponentDigestByName(receipt)
	populateBootComponentDigestSlices(receipt)
	if len(receipt.BootComponentDigests) > 1 {
		sort.Strings(receipt.BootComponentDigests)
	}
}

func normalizeBootComponentDigestByName(receipt *BackendLaunchReceipt) {
	if len(receipt.BootComponentDigestByName) == 0 {
		return
	}
	for name, digest := range receipt.BootComponentDigestByName {
		if !roleTokenPattern.MatchString(strings.TrimSpace(name)) || !looksLikeDigest(digest) {
			delete(receipt.BootComponentDigestByName, name)
		}
	}
	if len(receipt.BootComponentDigestByName) == 0 {
		receipt.BootComponentDigestByName = nil
	}
}

func populateBootComponentDigestSlices(receipt *BackendLaunchReceipt) {
	if len(receipt.BootComponentDigests) == 0 && len(receipt.BootComponentDigestByName) > 0 {
		receipt.BootComponentDigests = make([]string, 0, len(receipt.BootComponentDigestByName))
		for _, digest := range receipt.BootComponentDigestByName {
			receipt.BootComponentDigests = append(receipt.BootComponentDigests, digest)
		}
		return
	}
	if len(receipt.BootComponentDigestByName) != 0 || len(receipt.BootComponentDigests) == 0 {
		return
	}
	receipt.BootComponentDigestByName = map[string]string{}
	for idx, digest := range receipt.BootComponentDigests {
		if looksLikeDigest(digest) {
			receipt.BootComponentDigestByName[fmt.Sprintf("component_%02d", idx)] = digest
		}
	}
	if len(receipt.BootComponentDigestByName) == 0 {
		receipt.BootComponentDigestByName = nil
	}
}

func normalizeReceiptQEMUAndAttachmentFields(receipt *BackendLaunchReceipt) {
	receipt.QEMUProvenance = normalizeQEMUProvenance(receipt.QEMUProvenance)
	receipt.AttachmentPlanSummary = normalizeAttachmentPlanSummary(receipt.AttachmentPlanSummary)
	receipt.WorkspaceEncryptionPosture = normalizeWorkspaceEncryptionPosture(receipt.WorkspaceEncryptionPosture)
}

func normalizeQEMUProvenance(provenance *QEMUProvenance) *QEMUProvenance {
	if provenance == nil {
		return nil
	}
	trimmed := provenance.Trimmed()
	if trimmed.IsZero() || trimmed.Validate() != nil {
		return nil
	}
	return &trimmed
}

func normalizeAttachmentPlanSummary(summary *AttachmentPlanSummary) *AttachmentPlanSummary {
	if summary == nil {
		return nil
	}
	normalized := summary.Normalized()
	if normalized.Validate() != nil {
		return nil
	}
	return &normalized
}

func normalizeWorkspaceEncryptionPosture(posture *WorkspaceEncryptionPosture) *WorkspaceEncryptionPosture {
	if posture == nil {
		return nil
	}
	normalized := posture.Normalized()
	if normalized.Validate() != nil {
		return nil
	}
	return &normalized
}

func normalizeReceiptProvisioningAndSessionSecurity(receipt *BackendLaunchReceipt) {
	if receipt.ProvisioningPosture == ProvisioningPostureTOFU {
		if len(receipt.ProvisioningDegradedReasons) == 0 {
			receipt.ProvisioningDegradedReasons = []string{"tofu_unattested"}
		}
		receipt.ProvisioningPostureDegraded = true
	}
	if receipt.SessionSecurity == nil {
		return
	}
	if receipt.SessionSecurity.FrameFormat == "" {
		receipt.SessionSecurity.FrameFormat = SessionFramingLengthPrefixedV1
	}
	if len(receipt.SessionSecurity.DegradedReasons) > 0 {
		receipt.SessionSecurity.Degraded = true
	}
}

func normalizeReceiptRuntimePolicyFields(receipt *BackendLaunchReceipt) {
	if receipt.ResourceLimits != nil && receipt.ResourceLimits.Validate() != nil {
		receipt.ResourceLimits = nil
	}
	receipt.WatchdogPolicy = normalizeWatchdogPolicy(receipt.WatchdogPolicy)
	receipt.Lifecycle = normalizeLifecycleSnapshot(receipt.Lifecycle)
	if receipt.CachePosture != nil && receipt.CachePosture.Validate() != nil {
		receipt.CachePosture = nil
	}
	receipt.CacheEvidence = normalizeCacheEvidence(receipt.CacheEvidence)
}

func normalizeWatchdogPolicy(policy *BackendWatchdogPolicy) *BackendWatchdogPolicy {
	if policy == nil {
		return nil
	}
	normalized := policy.Normalized()
	if normalized.Validate() != nil {
		return nil
	}
	return &normalized
}

func normalizeLifecycleSnapshot(snapshot *BackendLifecycleSnapshot) *BackendLifecycleSnapshot {
	if snapshot == nil {
		return nil
	}
	normalized := snapshot.Normalized()
	if normalized.Validate() != nil {
		return nil
	}
	return &normalized
}

func normalizeCacheEvidence(evidence *BackendCacheEvidence) *BackendCacheEvidence {
	if evidence == nil {
		return nil
	}
	normalized := evidence.Normalized()
	if normalized.Validate() != nil {
		return nil
	}
	return &normalized
}

package brokerapi

import "github.com/runecode-ai/runecode/internal/launcherbackend"

func projectReceiptImageState(state map[string]any, receipt launcherbackend.BackendLaunchReceipt) {
	projectOptionalImageDigestState(state, receipt)
	projectOptionalImageSigningState(state, receipt)
	projectOptionalToolchainState(state, receipt)
	projectOptionalBootComponentState(state, receipt)
	if receipt.LaunchFailureReasonCode != "" {
		state["launch_failure_reason_code"] = receipt.LaunchFailureReasonCode
	}
}

func projectOptionalImageDigestState(state map[string]any, receipt launcherbackend.BackendLaunchReceipt) {
	if receipt.RuntimeImageDescriptorDigest != "" {
		state["runtime_image_descriptor_digest"] = receipt.RuntimeImageDescriptorDigest
		state["runtime_image_digest"] = receipt.RuntimeImageDescriptorDigest
	}
	if receipt.RuntimeImageBootProfile != "" {
		state["runtime_image_boot_profile"] = receipt.RuntimeImageBootProfile
	}
}

func projectOptionalImageSigningState(state map[string]any, receipt launcherbackend.BackendLaunchReceipt) {
	projectOptionalStringState(state, "runtime_image_signer_ref", receipt.RuntimeImageSignerRef)
	projectOptionalStringState(state, "runtime_image_verifier_ref", receipt.RuntimeImageVerifierRef)
	projectOptionalStringState(state, "runtime_image_signature_digest", receipt.RuntimeImageSignatureDigest)
}

func projectOptionalToolchainState(state map[string]any, receipt launcherbackend.BackendLaunchReceipt) {
	projectOptionalStringState(state, "runtime_toolchain_descriptor_digest", receipt.RuntimeToolchainDescriptorDigest)
	projectOptionalStringState(state, "runtime_toolchain_signer_ref", receipt.RuntimeToolchainSignerRef)
	projectOptionalStringState(state, "runtime_toolchain_verifier_ref", receipt.RuntimeToolchainVerifierRef)
	projectOptionalStringState(state, "runtime_toolchain_signature_digest", receipt.RuntimeToolchainSignatureDigest)
	projectOptionalStringState(state, "authority_state_digest", receipt.AuthorityStateDigest)
	if receipt.AuthorityStateRevision > 0 {
		state["authority_state_revision"] = receipt.AuthorityStateRevision
	}
}

func projectOptionalBootComponentState(state map[string]any, receipt launcherbackend.BackendLaunchReceipt) {
	if len(receipt.BootComponentDigestByName) > 0 {
		state["boot_component_digest_by_name"] = receipt.BootComponentDigestByName
	}
	if len(receipt.BootComponentDigests) > 0 {
		state["boot_component_digests"] = receipt.BootComponentDigests
	}
}

func projectOptionalStringState(state map[string]any, key string, value string) {
	if value != "" {
		state[key] = value
	}
}

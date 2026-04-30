package launcherdaemon

import (
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

const trustedRuntimeAttestationVerifierPolicyID = "runtime_asset_admission_identity"

func populateRuntimeSessionBinding(receipt *launcherbackend.BackendLaunchReceipt, spec launcherbackend.BackendLaunchSpec, runtimeImageDescriptorDigest string, isolateID string, sessionID string, nonce string) {
	if receipt == nil {
		return
	}
	receipt.ProvisioningPosture = launcherbackend.ProvisioningPostureAttested
	receipt.IsolateID = isolateID
	receipt.SessionID = sessionID
	receipt.SessionNonce = nonce
	receipt.LaunchContextDigest = syntheticLaunchContextDigest(spec, nonce)
	receipt.HandshakeTranscriptHash = syntheticHandshakeTranscriptHash(spec, nonce, runtimeImageDescriptorDigest)
	receipt.IsolateSessionKeyIDValue = syntheticSessionKeyIDValue(spec, nonce, runtimeImageDescriptorDigest)
}

func applyTrustedRuntimeAttestation(receipt *launcherbackend.BackendLaunchReceipt, admission launcherbackend.RuntimeAdmissionRecord, now time.Time) error {
	expectedMeasurementDigests, err := canonicalTrustedRuntimeMeasurementDigests(receipt, admission)
	if err != nil {
		return err
	}
	receipt.BootComponentDigests = componentDigestValues(receipt.BootComponentDigestByName)
	receipt.AttestationEvidenceSourceKind = launcherbackend.AttestationSourceKindTrustedRuntime
	receipt.AttestationMeasurementProfile = admission.AttestationMeasurementProfile
	receipt.AttestationFreshnessMaterial = []string{"session_nonce"}
	receipt.AttestationFreshnessBindingClaims = []string{"session_nonce", "handshake_transcript_hash", "launch_context_digest"}
	receipt.AttestationEvidenceClaimsDigest = expectedMeasurementDigests[0]
	receipt.AttestationVerifierPolicyID = trustedRuntimeAttestationVerifierPolicyID
	if receipt.AuthorityStateDigest != "" {
		receipt.AttestationVerifierPolicyDigest = receipt.AuthorityStateDigest
	} else {
		receipt.AttestationVerifierPolicyDigest = admission.RuntimeImageVerifierSetRef
	}
	receipt.AttestationVerificationRulesVersion = "trusted-runtime-v1"
	receipt.AttestationVerificationTimestamp = now.UTC().Format(time.RFC3339)
	receipt.AttestationVerificationResult = launcherbackend.AttestationVerificationResultValid
	receipt.AttestationReplayVerdict = launcherbackend.AttestationReplayVerdictOriginal
	receipt.AttestationVerificationReasonCodes = nil
	return nil
}

func canonicalTrustedRuntimeMeasurementDigests(receipt *launcherbackend.BackendLaunchReceipt, admission launcherbackend.RuntimeAdmissionRecord) ([]string, error) {
	if receipt == nil {
		return nil, nil
	}
	if receipt.RuntimeImageDescriptorDigest == "" || receipt.RuntimeImageBootProfile == "" {
		return nil, fmt.Errorf("runtime identity is required before attestation")
	}
	if receipt.IsolateID == "" || receipt.SessionID == "" || receipt.SessionNonce == "" || receipt.HandshakeTranscriptHash == "" || receipt.IsolateSessionKeyIDValue == "" {
		return nil, fmt.Errorf("session binding is required before attestation")
	}
	if admission.AttestationMeasurementProfile == "" || len(admission.AttestationExpectedMeasurementDigests) == 0 {
		return nil, fmt.Errorf("admitted attestation expectations are required")
	}
	expectedMeasurementDigests, err := launcherbackend.DeriveExpectedMeasurementDigests(admission.AttestationMeasurementProfile, admission.BootContractVersion, admission.ComponentDigests)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(expectedMeasurementDigests, admission.AttestationExpectedMeasurementDigests) {
		return nil, fmt.Errorf("admitted attestation expectations do not match canonical runtime identity")
	}
	return expectedMeasurementDigests, nil
}

func componentDigestValues(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	unique := out[:0]
	for _, value := range out {
		if value == "" {
			continue
		}
		if len(unique) == 0 || unique[len(unique)-1] != value {
			unique = append(unique, value)
		}
	}
	return append([]string{}, unique...)
}

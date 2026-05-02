package zkproof

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func normalizePrivateRemainderV0(payload trustpolicy.IsolateSessionBoundPayload) (IsolateSessionBoundPrivateRemainder, error) {
	backendKindCode, isolationAssuranceLevelCode, provisioningPostureCode, err := normalizePrivateRemainderEnumCodes(payload)
	if err != nil {
		return IsolateSessionBoundPrivateRemainder{}, err
	}
	launchContextDigest, handshakeTranscriptHashDigest, err := normalizePrivateRemainderTraceDigests(payload)
	if err != nil {
		return IsolateSessionBoundPrivateRemainder{}, err
	}
	runIDDigest, isolateIDDigest, sessionIDDigest, err := normalizePrivateRemainderIdentifierDigests(payload)
	if err != nil {
		return IsolateSessionBoundPrivateRemainder{}, err
	}
	return IsolateSessionBoundPrivateRemainder{RunIDDigest: runIDDigest, IsolateIDDigest: isolateIDDigest, SessionIDDigest: sessionIDDigest, BackendKindCode: backendKindCode, IsolationAssuranceLevelCode: isolationAssuranceLevelCode, ProvisioningPostureCode: provisioningPostureCode, LaunchContextDigest: launchContextDigest, HandshakeTranscriptHashDigest: handshakeTranscriptHashDigest}, nil
}

func normalizePrivateRemainderEnumCodes(payload trustpolicy.IsolateSessionBoundPayload) (uint16, uint16, uint16, error) {
	backendKindCode, err := normalizeEnumCode(payload.BackendKind, map[string]uint16{"microvm": 1, "container": 2, "unknown": 255}, "backend_kind")
	if err != nil {
		return 0, 0, 0, err
	}
	isolationAssuranceLevelCode, err := normalizeEnumCode(payload.IsolationAssuranceLevel, map[string]uint16{"isolated": 1, "degraded": 2, "unknown": 255, "not_applicable": 254}, "isolation_assurance_level")
	if err != nil {
		return 0, 0, 0, err
	}
	provisioningPostureCode, err := normalizeEnumCode(payload.ProvisioningPosture, map[string]uint16{"tofu": 1, "attested": 2, "unknown": 255, "not_applicable": 254}, "provisioning_posture")
	if err != nil {
		return 0, 0, 0, err
	}
	return backendKindCode, isolationAssuranceLevelCode, provisioningPostureCode, nil
}

func normalizePrivateRemainderTraceDigests(payload trustpolicy.IsolateSessionBoundPayload) (trustpolicy.Digest, trustpolicy.Digest, error) {
	launchContextDigest, err := parseDigestIdentity(payload.LaunchContextDigest, "launch_context_digest")
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	handshakeTranscriptHashDigest, err := parseDigestIdentity(payload.HandshakeTranscriptHash, "handshake_transcript_hash")
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	return launchContextDigest, handshakeTranscriptHashDigest, nil
}

func normalizePrivateRemainderIdentifierDigests(payload trustpolicy.IsolateSessionBoundPayload) (trustpolicy.Digest, trustpolicy.Digest, trustpolicy.Digest, error) {
	runIDDigest, err := stableIdentifierDigest(payload.RunID, "run_id")
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	isolateIDDigest, err := stableIdentifierDigest(payload.IsolateID, "isolate_id")
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	sessionIDDigest, err := stableIdentifierDigest(payload.SessionID, "session_id")
	if err != nil {
		return trustpolicy.Digest{}, trustpolicy.Digest{}, trustpolicy.Digest{}, err
	}
	return runIDDigest, isolateIDDigest, sessionIDDigest, nil
}

func normalizeEnumCode(value string, allowed map[string]uint16, fieldName string) (uint16, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	code, ok := allowed[normalized]
	if !ok {
		return 0, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("%s has unsupported value %q", fieldName, value)}
	}
	return code, nil
}

func stableIdentifierDigest(value string, fieldName string) (trustpolicy.Digest, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return trustpolicy.Digest{}, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("%s is required", fieldName)}
	}
	sum := sha256.Sum256([]byte(trimmed))
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, nil
}

func parseDigestIdentity(value string, fieldName string) (trustpolicy.Digest, error) {
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 {
		return trustpolicy.Digest{}, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("%s must be digest identity sha256:<64 lowercase hex>", fieldName)}
	}
	digest := trustpolicy.Digest{HashAlg: parts[0], Hash: parts[1]}
	if _, err := digest.Identity(); err != nil {
		return trustpolicy.Digest{}, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("%s: %v", fieldName, err)}
	}
	return digest, nil
}

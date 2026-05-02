package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/internal/zkproof"
)

type deterministicBindingCommitmentDeriver struct{}

func (deterministicBindingCommitmentDeriver) DeriveBindingCommitment(adapterProfileID string, normalized zkproof.IsolateSessionBoundPrivateRemainder) (string, error) {
	type commitmentInput struct {
		AdapterProfileID string                                      `json:"adapter_profile_id"`
		PrivateRemainder zkproof.IsolateSessionBoundPrivateRemainder `json:"private_remainder"`
	}
	b, err := json.Marshal(commitmentInput{AdapterProfileID: strings.TrimSpace(adapterProfileID), PrivateRemainder: normalized})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(append([]byte("runecode.zkproof.binding_commitment.v0:"), b...))
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

type deterministicSessionBindingRelationshipVerifier struct{}

func (deterministicSessionBindingRelationshipVerifier) VerifyNormalizedPrivateRemainderSessionBinding(normalized zkproof.IsolateSessionBoundPrivateRemainder, sourceSessionBindingDigest string) error {
	if strings.TrimSpace(sourceSessionBindingDigest) == "" {
		return fmt.Errorf("session_binding_digest is required")
	}
	want, err := deterministicSessionBindingDigest(normalized)
	if err != nil {
		return err
	}
	if strings.TrimSpace(want) != strings.TrimSpace(sourceSessionBindingDigest) {
		return fmt.Errorf("session_binding_digest mismatch")
	}
	return nil
}

func deterministicSessionBindingDigest(normalized zkproof.IsolateSessionBoundPrivateRemainder) (string, error) {
	input, err := deterministicSessionBindingInput(normalized)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(append([]byte("runecode.zkproof.fixture.session_binding.v0:"), b...))
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func deterministicSessionBindingInput(normalized zkproof.IsolateSessionBoundPrivateRemainder) (fixtureSessionBindingInput, error) {
	runID, isolateID, sessionID, err := deterministicSessionBindingCoreDigests(normalized)
	if err != nil {
		return fixtureSessionBindingInput{}, err
	}
	launchContext, err := normalized.LaunchContextDigest.Identity()
	if err != nil {
		return fixtureSessionBindingInput{}, err
	}
	handshake, err := normalized.HandshakeTranscriptHashDigest.Identity()
	if err != nil {
		return fixtureSessionBindingInput{}, err
	}
	return fixtureSessionBindingInput{
		RunIDDigest:                   runID,
		IsolateIDDigest:               isolateID,
		SessionIDDigest:               sessionID,
		BackendKindCode:               normalized.BackendKindCode,
		IsolationAssuranceLevelCode:   normalized.IsolationAssuranceLevelCode,
		ProvisioningPostureCode:       normalized.ProvisioningPostureCode,
		LaunchContextDigest:           launchContext,
		HandshakeTranscriptHashDigest: handshake,
	}, nil
}

func deterministicSessionBindingCoreDigests(normalized zkproof.IsolateSessionBoundPrivateRemainder) (string, string, string, error) {
	runID, err := normalized.RunIDDigest.Identity()
	if err != nil {
		return "", "", "", err
	}
	isolateID, err := normalized.IsolateIDDigest.Identity()
	if err != nil {
		return "", "", "", err
	}
	sessionID, err := normalized.SessionIDDigest.Identity()
	if err != nil {
		return "", "", "", err
	}
	return runID, isolateID, sessionID, nil
}

type fixtureSessionBindingInput struct {
	RunIDDigest                   string `json:"run_id_digest"`
	IsolateIDDigest               string `json:"isolate_id_digest"`
	SessionIDDigest               string `json:"session_id_digest"`
	BackendKindCode               uint16 `json:"backend_kind_code"`
	IsolationAssuranceLevelCode   uint16 `json:"isolation_assurance_level_code"`
	ProvisioningPostureCode       uint16 `json:"provisioning_posture_code"`
	LaunchContextDigest           string `json:"launch_context_digest"`
	HandshakeTranscriptHashDigest string `json:"handshake_transcript_hash_digest"`
}

type deterministicProofBackend struct{}

func (deterministicProofBackend) BackendIdentity() string { return "deterministic_fixture" }

func (deterministicProofBackend) VerifyDeterministic(proof []byte, publicInputsDigest trustpolicy.Digest) error {
	if len(proof) == 0 {
		return &zkproof.FeasibilityError{Code: "proof_invalid", Message: "empty proof"}
	}
	publicIdentity, err := publicInputsDigest.Identity()
	if err != nil {
		return err
	}
	want := sha256.Sum256(append([]byte("runecode.zkproof.fixture.proof.v0:"), []byte(publicIdentity)...))
	if len(proof) != len(want) {
		return &zkproof.FeasibilityError{Code: "proof_invalid", Message: "fixture proof length mismatch"}
	}
	if hex.EncodeToString(proof) != hex.EncodeToString(want[:]) {
		return &zkproof.FeasibilityError{Code: "proof_invalid", Message: "fixture proof bytes do not match canonical public_inputs_digest"}
	}
	return nil
}

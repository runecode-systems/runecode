package zkproof

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	bn254fr "github.com/consensys/gnark-crypto/ecc/bn254/fr"
	bn254poseidon2 "github.com/consensys/gnark-crypto/ecc/bn254/fr/poseidon2"
)

type poseidonBindingCommitmentDeriverV0 struct{}

func NewPoseidonBindingCommitmentDeriverV0() BindingCommitmentDeriver {
	return poseidonBindingCommitmentDeriverV0{}
}

func (poseidonBindingCommitmentDeriverV0) DeriveBindingCommitment(adapterProfileID string, normalized IsolateSessionBoundPrivateRemainder) (string, error) {
	if strings.TrimSpace(adapterProfileID) != SchemeAdapterGnarkGroth16IsolateSessionBoundV0 {
		return "", &FeasibilityError{Code: feasibilityCodeUnsupportedProfile, Message: fmt.Sprintf("unsupported scheme_adapter_id %q", adapterProfileID)}
	}
	folded, err := poseidonFoldPrivateRemainderV0(normalized)
	if err != nil {
		return "", err
	}
	foldedBytes := folded.Bytes()
	sum := sha256.Sum256(append([]byte("runecode.zkproof.binding_commitment.poseidon2.v0:"), foldedBytes[:]...))
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func poseidonFoldPrivateRemainderV0(normalized IsolateSessionBoundPrivateRemainder) (bn254fr.Element, error) {
	terms, err := privateRemainderPoseidonTermsV0(normalized)
	if err != nil {
		return bn254fr.Element{}, err
	}
	perm := bn254poseidon2.NewPermutation(2, 8, 56)
	acc := make([]byte, 32)
	for _, term := range terms {
		termBytes := term.Bytes()
		next, err := perm.Compress(acc, termBytes[:])
		if err != nil {
			return bn254fr.Element{}, &FeasibilityError{Code: feasibilityCodeUnsupportedCommitmentDeriver, Message: fmt.Sprintf("poseidon2 compress: %v", err)}
		}
		acc = append(acc[:0], next...)
	}
	var out bn254fr.Element
	out.SetBytes(acc)
	return out, nil
}

func privateRemainderPoseidonTermsV0(normalized IsolateSessionBoundPrivateRemainder) ([]bn254fr.Element, error) {
	run, err := digestToFieldElementV0(normalized.RunIDDigest, "run_id_digest")
	if err != nil {
		return nil, err
	}
	isolateID, err := digestToFieldElementV0(normalized.IsolateIDDigest, "isolate_id_digest")
	if err != nil {
		return nil, err
	}
	session, err := digestToFieldElementV0(normalized.SessionIDDigest, "session_id_digest")
	if err != nil {
		return nil, err
	}
	launch, err := digestToFieldElementV0(normalized.LaunchContextDigest, "launch_context_digest")
	if err != nil {
		return nil, err
	}
	handshake, err := digestToFieldElementV0(normalized.HandshakeTranscriptHashDigest, "handshake_transcript_hash_digest")
	if err != nil {
		return nil, err
	}
	return []bn254fr.Element{run, isolateID, session, uint16ToFieldElementV0(normalized.BackendKindCode), uint16ToFieldElementV0(normalized.IsolationAssuranceLevelCode), uint16ToFieldElementV0(normalized.ProvisioningPostureCode), launch, handshake}, nil
}

func digestToFieldElementV0(value interface{ Identity() (string, error) }, fieldName string) (bn254fr.Element, error) {
	identity, err := value.Identity()
	if err != nil {
		return bn254fr.Element{}, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("%s: %v", fieldName, err)}
	}
	parts := strings.SplitN(identity, ":", 2)
	raw, err := hex.DecodeString(parts[1])
	if err != nil {
		return bn254fr.Element{}, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("%s decode: %v", fieldName, err)}
	}
	var out bn254fr.Element
	out.SetBytes(raw)
	return out, nil
}

func uint16ToFieldElementV0(value uint16) bn254fr.Element {
	var out bn254fr.Element
	out.SetString(strconv.FormatUint(uint64(value), 10))
	return out
}

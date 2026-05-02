package zkproof

import (
	"encoding/hex"
	"fmt"
	"math/big"

	bn254fr "github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark/frontend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func buildCircuitWitnessV0(contract AuditIsolateSessionBoundAttestedRuntimeProofInputContract) (auditIsolateSessionBoundCircuitV0, error) {
	public, err := buildCircuitPublicWitnessV0(contract.PublicInputs)
	if err != nil {
		return auditIsolateSessionBoundCircuitV0{}, err
	}
	zeroMerkleWitnessV0(&public)
	if err := assignPrivateWitnessV0(&public, contract.WitnessInputs.PrivateRemainder); err != nil {
		return auditIsolateSessionBoundCircuitV0{}, err
	}
	if err := assignMerkleWitnessV0(&public, contract.WitnessInputs.MerkleAuthenticationPath); err != nil {
		return auditIsolateSessionBoundCircuitV0{}, err
	}
	return public, nil
}

func zeroMerkleWitnessV0(public *auditIsolateSessionBoundCircuitV0) {
	for i := 0; i < MaxMerklePathDepthV0; i++ {
		public.MerkleSiblingPosition[i] = 0
		for j := 0; j < 32; j++ {
			public.MerkleSiblingDigests[i][j] = 0
		}
	}
}

func assignPrivateWitnessV0(public *auditIsolateSessionBoundCircuitV0, private IsolateSessionBoundPrivateRemainder) error {
	run, err := digestToFieldBigIntV0(private.RunIDDigest, "run_id_digest")
	if err != nil {
		return err
	}
	isolateID, err := digestToFieldBigIntV0(private.IsolateIDDigest, "isolate_id_digest")
	if err != nil {
		return err
	}
	session, err := digestToFieldBigIntV0(private.SessionIDDigest, "session_id_digest")
	if err != nil {
		return err
	}
	launch, err := digestToFieldBigIntV0(private.LaunchContextDigest, "launch_context_digest")
	if err != nil {
		return err
	}
	handshake, err := digestToFieldBigIntV0(private.HandshakeTranscriptHashDigest, "handshake_transcript_hash_digest")
	if err != nil {
		return err
	}
	public.RunIDDigest = run
	public.IsolateIDDigest = isolateID
	public.SessionIDDigest = session
	public.BackendKindCode = uint64(private.BackendKindCode)
	public.IsolationAssuranceLevelCode = uint64(private.IsolationAssuranceLevelCode)
	public.ProvisioningPostureCode = uint64(private.ProvisioningPostureCode)
	public.LaunchContextDigest = launch
	public.HandshakeTranscriptDigest = handshake
	return nil
}

func assignMerkleWitnessV0(public *auditIsolateSessionBoundCircuitV0, path MerkleAuthenticationPath) error {
	public.MerklePathDepth = len(path.Steps)
	for i, step := range path.Steps {
		if i >= MaxMerklePathDepthV0 {
			break
		}
		code, err := merkleSiblingPositionCodeV0(step.SiblingPosition)
		if err != nil {
			return err
		}
		digestBytes, err := digestToPublicByteArrayV0(step.SiblingDigest, fmt.Sprintf("merkle_authentication_path.steps[%d].sibling_digest", i))
		if err != nil {
			return err
		}
		public.MerkleSiblingPosition[i] = code
		public.MerkleSiblingDigests[i] = digestBytes
	}
	return nil
}

func buildCircuitPublicWitnessV0(publicInputs AuditIsolateSessionBoundAttestedRuntimePublicInputs) (auditIsolateSessionBoundCircuitV0, error) {
	bindingDigest, err := parseDigestIdentity(publicInputs.BindingCommitment, "binding_commitment")
	if err != nil {
		return auditIsolateSessionBoundCircuitV0{}, err
	}
	bindingBytes, err := digestToPublicByteArrayV0(bindingDigest, "binding_commitment")
	if err != nil {
		return auditIsolateSessionBoundCircuitV0{}, err
	}
	merkleRootBytes, err := digestToPublicByteArrayV0(publicInputs.MerkleRoot, "merkle_root")
	if err != nil {
		return auditIsolateSessionBoundCircuitV0{}, err
	}
	auditRecordBytes, err := digestToPublicByteArrayV0(publicInputs.AuditRecordDigest, "audit_record_digest")
	if err != nil {
		return auditIsolateSessionBoundCircuitV0{}, err
	}
	return auditIsolateSessionBoundCircuitV0{BindingCommitment: bindingBytes, MerkleRoot: merkleRootBytes, AuditRecordDigest: auditRecordBytes}, nil
}

func digestToPublicByteArrayV0(d trustpolicy.Digest, fieldName string) ([32]frontend.Variable, error) {
	if _, err := d.Identity(); err != nil {
		return [32]frontend.Variable{}, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("%s: %v", fieldName, err)}
	}
	raw, err := hex.DecodeString(d.Hash)
	if err != nil {
		return [32]frontend.Variable{}, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("%s decode: %v", fieldName, err)}
	}
	if len(raw) != 32 {
		return [32]frontend.Variable{}, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("%s must be 32 bytes", fieldName)}
	}
	out := [32]frontend.Variable{}
	for i := range raw {
		out[i] = int(raw[i])
	}
	return out, nil
}

func merkleSiblingPositionCodeV0(position string) (frontend.Variable, error) {
	switch position {
	case merkleSiblingPositionLeft:
		return 0, nil
	case merkleSiblingPositionRight:
		return 1, nil
	case merkleSiblingPositionDuplicate:
		return 2, nil
	default:
		return nil, &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("unsupported sibling_position %q", position)}
	}
}

func digestToFieldBigIntV0(digest trustpolicy.Digest, fieldName string) (*big.Int, error) {
	if _, err := digest.Identity(); err != nil {
		return nil, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("%s: %v", fieldName, err)}
	}
	raw, err := hex.DecodeString(digest.Hash)
	if err != nil {
		return nil, &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("%s: %v", fieldName, err)}
	}
	var fe bn254fr.Element
	fe.SetBytes(raw)
	return fe.ToBigIntRegular(new(big.Int)), nil
}

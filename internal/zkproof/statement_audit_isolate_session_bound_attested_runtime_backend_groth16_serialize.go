package zkproof

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func serializeProofV0(proof groth16.Proof) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if _, err := proof.WriteTo(buf); err != nil {
		return nil, &FeasibilityError{Code: feasibilityCodeProofInvalid, Message: fmt.Sprintf("serialize proof: %v", err)}
	}
	return buf.Bytes(), nil
}

func hashConstraintSystemV0(cs constraint.ConstraintSystem) (trustpolicy.Digest, error) {
	buf := bytes.NewBuffer(nil)
	if _, err := cs.WriteTo(buf); err != nil {
		return trustpolicy.Digest{}, fmt.Errorf("serialize constraint system: %w", err)
	}
	return sha256DigestFromBytesV0(buf.Bytes()), nil
}

func hashProvingKeyV0(pk groth16.ProvingKey) (trustpolicy.Digest, error) {
	buf := bytes.NewBuffer(nil)
	if _, err := pk.WriteTo(buf); err != nil {
		return trustpolicy.Digest{}, fmt.Errorf("serialize proving key: %w", err)
	}
	return sha256DigestFromBytesV0(buf.Bytes()), nil
}

func hashVerifyingKeyV0(vk groth16.VerifyingKey) (trustpolicy.Digest, error) {
	buf := bytes.NewBuffer(nil)
	if _, err := vk.WriteTo(buf); err != nil {
		return trustpolicy.Digest{}, fmt.Errorf("serialize verifying key: %w", err)
	}
	return sha256DigestFromBytesV0(buf.Bytes()), nil
}

func sha256DigestFromBytesV0(b []byte) trustpolicy.Digest {
	sum := sha256.Sum256(b)
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}
}

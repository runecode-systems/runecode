package zkproof

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func validateSealEligibility(seal trustpolicy.AuditSegmentSealPayload) error {
	if seal.SchemaID != trustpolicy.AuditSegmentSealSchemaID {
		return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("segment seal schema_id must be %q", trustpolicy.AuditSegmentSealSchemaID)}
	}
	if seal.SchemaVersion != trustpolicy.AuditSegmentSealSchemaVersion {
		return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("segment seal schema_version must be %q", trustpolicy.AuditSegmentSealSchemaVersion)}
	}
	if seal.MerkleProfile != trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1 {
		return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("unsupported merkle_profile %q", seal.MerkleProfile)}
	}
	if _, err := seal.MerkleRoot.Identity(); err != nil {
		return &FeasibilityError{Code: feasibilityCodeMissingBoundedInput, Message: fmt.Sprintf("merkle_root: %v", err)}
	}
	return nil
}

func validateMerkleAuthenticationPath(path MerkleAuthenticationPath) error {
	if path.PathVersion != MerkleAuthenticationPathFormatV1 {
		return &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("merkle_authentication_path.path_version must be %q", MerkleAuthenticationPathFormatV1)}
	}
	if len(path.Steps) > MaxMerklePathDepthV0 {
		return &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("merkle_authentication_path depth %d exceeds max %d", len(path.Steps), MaxMerklePathDepthV0)}
	}
	for index, step := range path.Steps {
		if _, err := step.SiblingDigest.Identity(); err != nil {
			return &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("merkle_authentication_path.steps[%d].sibling_digest: %v", index, err)}
		}
		switch step.SiblingPosition {
		case merkleSiblingPositionLeft, merkleSiblingPositionRight, merkleSiblingPositionDuplicate:
		default:
			return &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("merkle_authentication_path.steps[%d].sibling_position %q is unsupported", index, step.SiblingPosition)}
		}
	}
	return nil
}

func DeriveAuditSegmentMerkleAuthenticationPathV0(recordDigests []trustpolicy.Digest, leafIndex int) (MerkleAuthenticationPath, error) {
	if err := validateMerklePathDerivationInputs(recordDigests, leafIndex); err != nil {
		return MerkleAuthenticationPath{}, err
	}
	level, err := buildMerkleLeafLevel(recordDigests)
	if err != nil {
		return MerkleAuthenticationPath{}, err
	}
	path := MerkleAuthenticationPath{PathVersion: MerkleAuthenticationPathFormatV1, LeafIndex: uint64(leafIndex), Steps: []MerkleAuthenticationStep{}}
	currentIndex := leafIndex
	for len(level) > 1 {
		if err := appendMerkleAuthenticationStep(&path, level, currentIndex); err != nil {
			return MerkleAuthenticationPath{}, err
		}
		level = buildNextMerkleLevel(level)
		currentIndex /= 2
	}
	return path, nil
}

func VerifyAuditSegmentMerkleAuthenticationPathAgainstSealV0(recordDigest trustpolicy.Digest, path MerkleAuthenticationPath, seal trustpolicy.AuditSegmentSealPayload) error {
	if err := validateMerkleVerificationInputs(path, seal); err != nil {
		return err
	}
	current, err := buildMerkleLeafHash(recordDigest)
	if err != nil {
		return err
	}
	for index, step := range path.Steps {
		current, err = replayMerkleAuthenticationStep(index, step, current)
		if err != nil {
			return err
		}
	}
	computedRoot := bytesToDigest(current)
	computedIdentity, _ := computedRoot.Identity()
	expectedIdentity, _ := seal.MerkleRoot.Identity()
	if computedIdentity != expectedIdentity {
		return &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("merkle_root mismatch from authentication path: got %q want %q", computedIdentity, expectedIdentity)}
	}
	return nil
}

func validateMerklePathDerivationInputs(recordDigests []trustpolicy.Digest, leafIndex int) error {
	if len(recordDigests) == 0 {
		return &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: "record digests are required for merkle authentication path derivation"}
	}
	if leafIndex < 0 || leafIndex >= len(recordDigests) {
		return &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("leaf_index %d out of range for %d record digests", leafIndex, len(recordDigests))}
	}
	return nil
}

func buildMerkleLeafLevel(recordDigests []trustpolicy.Digest) ([][32]byte, error) {
	level := make([][32]byte, 0, len(recordDigests))
	for index := range recordDigests {
		leafHash, err := buildMerkleLeafHash(recordDigests[index])
		if err != nil {
			return nil, &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("record_digests[%d]: %v", index, err)}
		}
		level = append(level, leafHash)
	}
	return level, nil
}

func buildMerkleLeafHash(recordDigest trustpolicy.Digest) ([32]byte, error) {
	recordDigestBytes, err := digestHexBytes(recordDigest)
	if err != nil {
		return [32]byte{}, &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("audit_record_digest: %v", err)}
	}
	return sha256.Sum256(append(append([]byte{}, []byte("runecode.audit.merkle.leaf.v1:")...), recordDigestBytes...)), nil
}

func appendMerkleAuthenticationStep(path *MerkleAuthenticationPath, level [][32]byte, currentIndex int) error {
	if len(path.Steps) >= MaxMerklePathDepthV0 {
		return &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("merkle_authentication_path depth exceeds max %d", MaxMerklePathDepthV0)}
	}
	path.Steps = append(path.Steps, merkleAuthenticationStepForIndex(level, currentIndex))
	return nil
}

func merkleAuthenticationStepForIndex(level [][32]byte, currentIndex int) MerkleAuthenticationStep {
	if currentIndex%2 == 1 {
		sibling := level[currentIndex-1]
		return MerkleAuthenticationStep{SiblingDigest: bytesToDigest(sibling), SiblingPosition: merkleSiblingPositionLeft}
	}
	if currentIndex+1 < len(level) {
		sibling := level[currentIndex+1]
		return MerkleAuthenticationStep{SiblingDigest: bytesToDigest(sibling), SiblingPosition: merkleSiblingPositionRight}
	}
	sibling := level[currentIndex]
	return MerkleAuthenticationStep{SiblingDigest: bytesToDigest(sibling), SiblingPosition: merkleSiblingPositionDuplicate}
}

func buildNextMerkleLevel(level [][32]byte) [][32]byte {
	next := make([][32]byte, 0, (len(level)+1)/2)
	for index := 0; index < len(level); index += 2 {
		left := level[index]
		right := left
		if index+1 < len(level) {
			right = level[index+1]
		}
		next = append(next, sha256.Sum256(append(append(append([]byte{}, []byte("runecode.audit.merkle.node.v1:")...), left[:]...), right[:]...)))
	}
	return next
}

func validateMerkleVerificationInputs(path MerkleAuthenticationPath, seal trustpolicy.AuditSegmentSealPayload) error {
	if err := validateMerkleAuthenticationPath(path); err != nil {
		return err
	}
	if seal.MerkleProfile != trustpolicy.AuditSegmentMerkleProfileOrderedDSEv1 {
		return &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("unsupported merkle_profile %q", seal.MerkleProfile)}
	}
	if _, err := seal.MerkleRoot.Identity(); err != nil {
		return &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("merkle_root: %v", err)}
	}
	return nil
}

func replayMerkleAuthenticationStep(index int, step MerkleAuthenticationStep, current [32]byte) ([32]byte, error) {
	siblingBytes, err := digestHexBytes(step.SiblingDigest)
	if err != nil {
		return [32]byte{}, &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("merkle_authentication_path.steps[%d].sibling_digest: %v", index, err)}
	}
	if len(siblingBytes) != sha256.Size {
		return [32]byte{}, &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("merkle_authentication_path.steps[%d].sibling_digest must be sha256 digest", index)}
	}
	sibling := [32]byte{}
	copy(sibling[:], siblingBytes)
	left, right, err := merkleReplayChildren(index, step.SiblingPosition, sibling, current)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(append(append(append([]byte{}, []byte("runecode.audit.merkle.node.v1:")...), left[:]...), right[:]...)), nil
}

func merkleReplayChildren(index int, position string, sibling, current [32]byte) ([32]byte, [32]byte, error) {
	switch position {
	case merkleSiblingPositionLeft:
		return sibling, current, nil
	case merkleSiblingPositionRight:
		return current, sibling, nil
	case merkleSiblingPositionDuplicate:
		if sibling != current {
			return [32]byte{}, [32]byte{}, &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("merkle_authentication_path.steps[%d] duplicate sibling must equal current node hash", index)}
		}
		return current, current, nil
	default:
		return [32]byte{}, [32]byte{}, &FeasibilityError{Code: feasibilityCodeInvalidMerklePath, Message: fmt.Sprintf("merkle_authentication_path.steps[%d].sibling_position %q is unsupported", index, position)}
	}
}

func digestHexBytes(digest trustpolicy.Digest) ([]byte, error) {
	if _, err := digest.Identity(); err != nil {
		return nil, err
	}
	decoded, err := hex.DecodeString(digest.Hash)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func bytesToDigest(value [32]byte) trustpolicy.Digest {
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(value[:])}
}

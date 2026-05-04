package trustpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func ComputeOrderedAuditSegmentMerkleRoot(recordDigests []Digest) (Digest, error) {
	if len(recordDigests) == 0 {
		return Digest{}, fmt.Errorf("record digests are required for merkle construction")
	}
	level, err := merkleLeafLevel(recordDigests)
	if err != nil {
		return Digest{}, err
	}
	for len(level) > 1 {
		level = merkleNextLevel(level)
	}
	return Digest{HashAlg: "sha256", Hash: hex.EncodeToString(level[0][:])}, nil
}

func ComputeOrderedAuditSegmentMerkleCompactPath(recordDigests []Digest, leafIndex int) ([]Digest, error) {
	if len(recordDigests) == 0 {
		return nil, fmt.Errorf("record digests are required for merkle construction")
	}
	if leafIndex < 0 || leafIndex >= len(recordDigests) {
		return nil, fmt.Errorf("leaf_index %d out of bounds", leafIndex)
	}
	level, err := merkleLeafLevel(recordDigests)
	if err != nil {
		return nil, err
	}
	path := make([]Digest, 0, 16)
	current := leafIndex
	for len(level) > 1 {
		path = append(path, merkleLevelSiblingDigest(level, current))
		level = merkleNextLevel(level)
		current /= 2
	}
	if len(path) == 0 {
		return nil, nil
	}
	return path, nil
}

func merkleLeafLevel(recordDigests []Digest) ([][32]byte, error) {
	level := make([][32]byte, 0, len(recordDigests))
	for index := range recordDigests {
		leafHash, err := merkleLeafHashFromRecordDigest(recordDigests[index])
		if err != nil {
			return nil, fmt.Errorf("record_digests[%d]: %w", index, err)
		}
		level = append(level, leafHash)
	}
	return level, nil
}

func merkleNextLevel(level [][32]byte) [][32]byte {
	next := make([][32]byte, 0, (len(level)+1)/2)
	for index := 0; index < len(level); index += 2 {
		left := level[index]
		right := left
		if index+1 < len(level) {
			right = level[index+1]
		}
		next = append(next, merkleNodeHash(left, right))
	}
	return next
}

func merkleLevelSiblingDigest(level [][32]byte, current int) Digest {
	siblingIndex := current ^ 1
	sibling := level[current]
	if siblingIndex < len(level) {
		sibling = level[siblingIndex]
	}
	return Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sibling[:])}
}

func VerifyOrderedAuditSegmentMerkleRoot(recordDigests []Digest, expected Digest) error {
	expectedIdentity, err := expected.Identity()
	if err != nil {
		return fmt.Errorf("expected merkle root: %w", err)
	}
	computed, err := ComputeOrderedAuditSegmentMerkleRoot(recordDigests)
	if err != nil {
		return err
	}
	computedIdentity, _ := computed.Identity()
	if computedIdentity != expectedIdentity {
		return fmt.Errorf("merkle root mismatch: got %q want %q", computedIdentity, expectedIdentity)
	}
	return nil
}

func VerifyOrderedAuditSegmentMerkleCompactPath(leafDigest Digest, leafIndex int, leafCount int, compactPath []Digest, expected Digest) error {
	if err := validateCompactMerklePathInputs(leafIndex, leafCount, compactPath); err != nil {
		return err
	}
	node, err := merkleLeafHashFromRecordDigest(leafDigest)
	if err != nil {
		return fmt.Errorf("leaf_digest: %w", err)
	}
	position := leafIndex
	for level := range compactPath {
		sibling, err := compactMerkleSiblingHash(compactPath[level], level)
		if err != nil {
			return err
		}
		if position%2 == 0 {
			node = merkleNodeHash(node, sibling)
		} else {
			node = merkleNodeHash(sibling, node)
		}
		position /= 2
	}
	return verifyComputedMerkleRoot(node, expected)
}

func validateCompactMerklePathInputs(leafIndex int, leafCount int, compactPath []Digest) error {
	if leafCount <= 0 {
		return fmt.Errorf("leaf_count must be positive")
	}
	if leafIndex < 0 || leafIndex >= leafCount {
		return fmt.Errorf("leaf_index %d out of bounds", leafIndex)
	}
	expectedLevels := 0
	for levelCount := leafCount; levelCount > 1; levelCount = (levelCount + 1) / 2 {
		expectedLevels++
	}
	if len(compactPath) != expectedLevels {
		return fmt.Errorf("compact path length %d does not match expected %d", len(compactPath), expectedLevels)
	}
	return nil
}

func compactMerkleSiblingHash(digest Digest, level int) ([32]byte, error) {
	siblingBytes, err := digestHexBytes(digest)
	if err != nil {
		return [32]byte{}, fmt.Errorf("compact_path[%d]: %w", level, err)
	}
	if len(siblingBytes) != sha256.Size {
		return [32]byte{}, fmt.Errorf("compact_path[%d]: digest must be 32 bytes", level)
	}
	sibling := [32]byte{}
	copy(sibling[:], siblingBytes)
	return sibling, nil
}

func verifyComputedMerkleRoot(node [32]byte, expected Digest) error {
	computed := Digest{HashAlg: "sha256", Hash: hex.EncodeToString(node[:])}
	computedIdentity, _ := computed.Identity()
	expectedIdentity, err := expected.Identity()
	if err != nil {
		return fmt.Errorf("expected merkle root: %w", err)
	}
	if computedIdentity != expectedIdentity {
		return fmt.Errorf("merkle root mismatch: got %q want %q", computedIdentity, expectedIdentity)
	}
	return nil
}

func merkleLeafHashFromRecordDigest(recordDigest Digest) ([32]byte, error) {
	recordDigestBytes, err := digestHexBytes(recordDigest)
	if err != nil {
		return [32]byte{}, err
	}
	leafMaterial := append(append([]byte{}, []byte("runecode.audit.merkle.leaf.v1:")...), recordDigestBytes...)
	return sha256.Sum256(leafMaterial), nil
}

func merkleNodeHash(left [32]byte, right [32]byte) [32]byte {
	nodeMaterial := append(append(append([]byte{}, []byte("runecode.audit.merkle.node.v1:")...), left[:]...), right[:]...)
	return sha256.Sum256(nodeMaterial)
}

func ComputeSegmentFileHash(rawFramedSegmentBytes []byte) (Digest, error) {
	if len(rawFramedSegmentBytes) == 0 {
		return Digest{}, fmt.Errorf("raw framed segment bytes are required")
	}
	sum := sha256.Sum256(rawFramedSegmentBytes)
	return Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, nil
}

func VerifySegmentFileHash(rawFramedSegmentBytes []byte, expected Digest) error {
	expectedIdentity, err := expected.Identity()
	if err != nil {
		return fmt.Errorf("expected segment file hash: %w", err)
	}
	computed, err := ComputeSegmentFileHash(rawFramedSegmentBytes)
	if err != nil {
		return err
	}
	computedIdentity, _ := computed.Identity()
	if computedIdentity != expectedIdentity {
		return fmt.Errorf("segment_file_hash mismatch: got %q want %q", computedIdentity, expectedIdentity)
	}
	return nil
}

func digestHexBytes(digest Digest) ([]byte, error) {
	if _, err := digest.Identity(); err != nil {
		return nil, err
	}
	decoded, err := hex.DecodeString(digest.Hash)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

package zkproof

import (
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/sha2"
	"github.com/consensys/gnark/std/math/cmp"
	"github.com/consensys/gnark/std/math/uints"
	poseidon2std "github.com/consensys/gnark/std/permutation/poseidon2"
)

type auditIsolateSessionBoundCircuitV0 struct {
	PublicInputsDigest            [32]frontend.Variable `gnark:",public"`
	AuditSegmentSealDigest        [32]frontend.Variable `gnark:",public"`
	BindingCommitment             [32]frontend.Variable `gnark:",public"`
	MerkleRoot                    [32]frontend.Variable `gnark:",public"`
	AuditRecordDigest             [32]frontend.Variable `gnark:",public"`
	ProtocolBundleManifestHash    [32]frontend.Variable `gnark:",public"`
	RuntimeImageDescriptorDigest  [32]frontend.Variable `gnark:",public"`
	AttestationEvidenceDigest     [32]frontend.Variable `gnark:",public"`
	AppliedHardeningPostureDigest [32]frontend.Variable `gnark:",public"`
	SessionBindingDigest          [32]frontend.Variable `gnark:",public"`

	RunIDDigest                 frontend.Variable
	IsolateIDDigest             frontend.Variable
	SessionIDDigest             frontend.Variable
	BackendKindCode             frontend.Variable
	IsolationAssuranceLevelCode frontend.Variable
	ProvisioningPostureCode     frontend.Variable
	LaunchContextDigest         frontend.Variable
	HandshakeTranscriptDigest   frontend.Variable

	MerklePathDepth       frontend.Variable
	MerkleSiblingPosition [MaxMerklePathDepthV0]frontend.Variable
	MerkleSiblingDigests  [MaxMerklePathDepthV0][32]frontend.Variable
}

func (c *auditIsolateSessionBoundCircuitV0) Define(api frontend.API) error {
	if err := c.assertPublicInputsDigest(api); err != nil {
		return err
	}
	if err := c.assertBindingCommitment(api); err != nil {
		return err
	}
	if err := c.assertMerkleMembership(api); err != nil {
		return err
	}
	return nil
}

func (c *auditIsolateSessionBoundCircuitV0) assertPublicInputsDigest(api frontend.API) error {
	preimage := make([]frontend.Variable, 0, 512)
	preimage = appendLabeledConstantBytesVarsV0(preimage, "statement_family", StatementFamilyAuditIsolateSessionBoundAttestedRuntimeMembershipV0)
	preimage = appendLabeledConstantBytesVarsV0(preimage, "statement_version", StatementVersionV0)
	preimage = appendLabeledConstantBytesVarsV0(preimage, "normalization_profile_id", NormalizationProfileAuditIsolateSessionBoundAttestedRuntimeV0)
	preimage = appendLabeledConstantBytesVarsV0(preimage, "scheme_adapter_id", SchemeAdapterGnarkGroth16IsolateSessionBoundV0)
	for _, field := range []struct {
		label string
		value [32]frontend.Variable
	}{
		{label: "audit_segment_seal_digest", value: c.AuditSegmentSealDigest},
		{label: "merkle_root", value: c.MerkleRoot},
		{label: "audit_record_digest", value: c.AuditRecordDigest},
		{label: "protocol_bundle_manifest_hash", value: c.ProtocolBundleManifestHash},
		{label: "runtime_image_descriptor_digest", value: c.RuntimeImageDescriptorDigest},
		{label: "attestation_evidence_digest", value: c.AttestationEvidenceDigest},
		{label: "applied_hardening_posture_digest", value: c.AppliedHardeningPostureDigest},
		{label: "session_binding_digest", value: c.SessionBindingDigest},
		{label: "binding_commitment", value: c.BindingCommitment},
	} {
		preimage = appendLabeledDigestBytesVarsV0(preimage, field.label, field.value)
	}
	digest, err := sha256BytesV0(api, preimage)
	if err != nil {
		return err
	}
	for i := 0; i < 32; i++ {
		api.AssertIsEqual(digest[i], c.PublicInputsDigest[i])
	}
	return nil
}

func (c *auditIsolateSessionBoundCircuitV0) assertBindingCommitment(api frontend.API) error {
	hasher, err := poseidon2std.NewPoseidon2FromParameters(api, 2, 8, 56)
	if err != nil {
		return err
	}
	acc := frontend.Variable(0)
	for _, term := range []frontend.Variable{c.RunIDDigest, c.IsolateIDDigest, c.SessionIDDigest, c.BackendKindCode, c.IsolationAssuranceLevelCode, c.ProvisioningPostureCode, c.LaunchContextDigest, c.HandshakeTranscriptDigest} {
		acc = hasher.Compress(acc, term)
	}
	foldedBytes := fieldElementToBytesBEV0(api, acc)
	bindingInput := append(constantBytesVarsV0(bindingCommitmentPrefixV0), foldedBytes...)
	bindingDigest, err := sha256BytesV0(api, bindingInput)
	if err != nil {
		return err
	}
	for i := 0; i < 32; i++ {
		api.AssertIsEqual(bindingDigest[i], c.BindingCommitment[i])
	}
	return nil
}

func (c *auditIsolateSessionBoundCircuitV0) assertMerkleMembership(api frontend.API) error {
	api.AssertIsLessOrEqual(c.MerklePathDepth, frontend.Variable(MaxMerklePathDepthV0))
	leafInput := append(constantBytesVarsV0(merkleLeafPrefixV0), c.AuditRecordDigest[:]...)
	current, err := sha256BytesV0(api, leafInput)
	if err != nil {
		return err
	}
	for i := 0; i < MaxMerklePathDepthV0; i++ {
		active := cmp.IsLess(api, frontend.Variable(i), c.MerklePathDepth)
		pos := c.MerkleSiblingPosition[i]
		posLeft := api.IsZero(pos)
		posRight := api.IsZero(api.Sub(pos, 1))
		posDuplicate := api.IsZero(api.Sub(pos, 2))
		posOneHot := api.Add(posLeft, posRight, posDuplicate)
		api.AssertIsEqual(api.Mul(active, posOneHot), active)

		leftBytes := make([]frontend.Variable, 32)
		rightBytes := make([]frontend.Variable, 32)
		for j := 0; j < 32; j++ {
			sibling := c.MerkleSiblingDigests[i][j]
			curr := current[j]
			leftBytes[j] = api.Add(api.Mul(sibling, posLeft), api.Mul(curr, api.Add(posRight, posDuplicate)))
			rightBytes[j] = api.Add(api.Mul(curr, api.Add(posLeft, posDuplicate)), api.Mul(sibling, posRight))
			api.AssertIsEqual(api.Mul(active, api.Mul(posDuplicate, api.Sub(sibling, curr))), 0)
		}
		nodeInput := append(constantBytesVarsV0(merkleNodePrefixV0), append(leftBytes, rightBytes...)...)
		next, err := sha256BytesV0(api, nodeInput)
		if err != nil {
			return err
		}
		for j := 0; j < 32; j++ {
			current[j] = api.Add(api.Mul(active, next[j]), api.Mul(api.Sub(1, active), current[j]))
		}
	}
	for i := 0; i < 32; i++ {
		api.AssertIsEqual(current[i], c.MerkleRoot[i])
	}
	return nil
}

func constantBytesVarsV0(s string) []frontend.Variable {
	out := make([]frontend.Variable, len(s))
	for i := range s {
		out[i] = int(s[i])
	}
	return out
}

func appendLabeledConstantBytesVarsV0(dst []frontend.Variable, label, value string) []frontend.Variable {
	dst = append(dst, constantBytesVarsV0(label)...)
	dst = append(dst, frontend.Variable('='))
	dst = append(dst, constantBytesVarsV0(value)...)
	return append(dst, frontend.Variable('|'))
}

func appendLabeledDigestBytesVarsV0(dst []frontend.Variable, label string, value [32]frontend.Variable) []frontend.Variable {
	dst = append(dst, constantBytesVarsV0(label)...)
	dst = append(dst, frontend.Variable('='))
	dst = append(dst, value[:]...)
	return append(dst, frontend.Variable('|'))
}

func fieldElementToBytesBEV0(api frontend.API, value frontend.Variable) []frontend.Variable {
	bits := api.ToBinary(value, 256)
	out := make([]frontend.Variable, 32)
	for i := 0; i < 32; i++ {
		start := (31 - i) * 8
		byteValue := frontend.Variable(0)
		for j := 0; j < 8; j++ {
			byteValue = api.Add(byteValue, api.Mul(bits[start+j], frontend.Variable(1<<j)))
		}
		out[i] = byteValue
	}
	return out
}

func sha256BytesV0(api frontend.API, input []frontend.Variable) ([]frontend.Variable, error) {
	bytesAPI, err := uints.NewBytes(api)
	if err != nil {
		return nil, err
	}
	u8Input := make([]uints.U8, len(input))
	for i := range input {
		u8Input[i] = bytesAPI.ValueOf(input[i])
	}
	h, err := sha2.New(api)
	if err != nil {
		return nil, err
	}
	h.Write(u8Input)
	sum := h.Sum()
	out := make([]frontend.Variable, len(sum))
	for i := range sum {
		out[i] = bytesAPI.Value(sum[i])
	}
	return out, nil
}

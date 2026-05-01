package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func externalAnchorProofSidecarPayload(record artifacts.ExternalAnchorPreparedMutationRecord, primaryTarget externalAnchorResolvedTarget, targetSet []externalAnchorResolvedTarget) map[string]any {
	return map[string]any{
		"schema_id":                primaryTarget.ProofSchemaID,
		"schema_version":           "0.1.0",
		"prepared_mutation_id":     strings.TrimSpace(record.PreparedMutationID),
		"execution_attempt_id":     strings.TrimSpace(record.LastExecuteAttemptID),
		"target_kind":              primaryTarget.TargetKind,
		"target_descriptor":        cloneStringAnyMap(primaryTarget.TargetDescriptor),
		"target_descriptor_digest": primaryTarget.TargetDescriptorDigest,
		"target_set":               externalAnchorProofSidecarTargetSet(targetSet),
		"seal_digest":              record.TypedRequest["seal_digest"],
		"outbound_payload_digest":  record.TypedRequest["outbound_payload_digest"],
	}
}

func externalAnchorProviderReceiptSidecarPayload(record artifacts.ExternalAnchorPreparedMutationRecord, primaryTarget externalAnchorResolvedTarget) map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.audit.anchor_provider_receipt.v0",
		"schema_version":           "0.1.0",
		"runtime_adapter":          primaryTarget.RuntimeAdapter,
		"target_kind":              primaryTarget.TargetKind,
		"target_descriptor":        cloneStringAnyMap(primaryTarget.TargetDescriptor),
		"target_descriptor_digest": primaryTarget.TargetDescriptorDigest,
		"execution_state":          strings.TrimSpace(record.ExecutionState),
		"execution_reason_code":    strings.TrimSpace(record.ExecutionReasonCode),
	}
}

func externalAnchorVerificationTranscriptSidecarPayload(record artifacts.ExternalAnchorPreparedMutationRecord, primaryTarget externalAnchorResolvedTarget) map[string]any {
	return map[string]any{
		"schema_id":                "runecode.protocol.audit.anchor_verification_transcript.v0",
		"schema_version":           "0.1.0",
		"target_kind":              primaryTarget.TargetKind,
		"target_descriptor_digest": primaryTarget.TargetDescriptorDigest,
		"checked_bindings":         []string{"seal_digest", "target_descriptor", "target_descriptor_digest", "target_set", "typed_request_hash", "approval", "lease"},
		"verification_result":      strings.TrimSpace(record.ExecutionState),
	}
}

func externalAnchorResolvedPrimaryTargetFromPreparedRecord(record artifacts.ExternalAnchorPreparedMutationRecord) (externalAnchorResolvedTarget, error) {
	primary, _, err := externalAnchorResolvedTargetsFromPreparedRecord(record)
	if err != nil {
		return externalAnchorResolvedTarget{}, err
	}
	return primary, nil
}

func externalAnchorResolvedTargetsFromPreparedRecord(record artifacts.ExternalAnchorPreparedMutationRecord) (externalAnchorResolvedTarget, []externalAnchorResolvedTarget, error) {
	primary, err := externalAnchorResolvedTargetFromPreparedBinding(record.PrimaryTarget, "primary_target")
	if err != nil {
		return externalAnchorResolvedTarget{}, nil, err
	}
	targets, err := externalAnchorResolvedTargetSetFromPreparedRecord(record.TargetSet)
	if err != nil {
		return externalAnchorResolvedTarget{}, nil, err
	}
	if len(targets) == 0 {
		return primary, []externalAnchorResolvedTarget{primary}, nil
	}
	return primary, targets, nil
}

func externalAnchorResolvedTargetSetFromPreparedRecord(targetSet []artifacts.ExternalAnchorPreparedTargetBinding) ([]externalAnchorResolvedTarget, error) {
	out := make([]externalAnchorResolvedTarget, 0, len(targetSet))
	for i := range targetSet {
		target, err := externalAnchorResolvedTargetFromPreparedBinding(targetSet[i], fmt.Sprintf("target_set[%d]", i))
		if err != nil {
			return nil, err
		}
		out = append(out, target)
	}
	return out, nil
}

func externalAnchorResolvedTargetFromPreparedBinding(binding artifacts.ExternalAnchorPreparedTargetBinding, field string) (externalAnchorResolvedTarget, error) {
	kind := strings.TrimSpace(binding.TargetKind)
	profile, err := externalAnchorImplementedProfile(kind)
	if err != nil {
		return externalAnchorResolvedTarget{}, fmt.Errorf("%s.target_kind: %w", field, err)
	}
	requirement := trustpolicy.NormalizeExternalAnchorTargetRequirement(binding.TargetRequirement)
	if err := trustpolicy.ValidateExternalAnchorTargetRequirement(requirement); err != nil {
		return externalAnchorResolvedTarget{}, fmt.Errorf("%s.target_requirement: %w", field, err)
	}
	digest, err := digestFromIdentity(strings.TrimSpace(binding.TargetDescriptorDigest))
	if err != nil {
		return externalAnchorResolvedTarget{}, fmt.Errorf("%s.target_descriptor_digest invalid: %w", field, err)
	}
	identity, _ := digest.Identity()
	if len(binding.TargetDescriptor) == 0 {
		return externalAnchorResolvedTarget{}, fmt.Errorf("%s.target_descriptor is required", field)
	}
	canonicalIdentity, err := externalAnchorCanonicalDescriptorDigestIdentity(binding.TargetDescriptor)
	if err != nil {
		return externalAnchorResolvedTarget{}, fmt.Errorf("%s.target_descriptor canonical hash failed: %w", field, err)
	}
	if canonicalIdentity != identity {
		return externalAnchorResolvedTarget{}, fmt.Errorf("%s.target_descriptor_digest must match canonical digest of %s.target_descriptor", field, field)
	}
	return externalAnchorResolvedTarget{
		TargetKind:               kind,
		TargetRequirement:        requirement,
		TargetDescriptor:         cloneStringAnyMap(binding.TargetDescriptor),
		TargetDescriptorDigest:   digest,
		TargetDescriptorIdentity: identity,
		RuntimeAdapter:           profile.runtimeAdapter,
		ReceiptKind:              profile.receiptKind,
		ProofKind:                profile.proofKind,
		ProofSchemaID:            profile.proofSchemaID,
	}, nil
}

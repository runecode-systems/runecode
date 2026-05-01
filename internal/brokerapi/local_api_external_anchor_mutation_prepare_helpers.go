package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func resolveExternalAnchorRequestKind(typedRequest map[string]any) string {
	return strings.TrimSpace(stringField(typedRequest, "request_kind"))
}

func validateExternalAnchorRequestKind(requestKind string) error {
	if requestKind == externalAnchorRequestKindSubmitV0 {
		return nil
	}
	return fmt.Errorf("typed_request.request_kind must be %s", externalAnchorRequestKindSubmitV0)
}

func canonicalizeExternalAnchorTypedRequest(typedRequest map[string]any) (trustpolicy.Digest, string, error) {
	typedRequestHashIdentity, err := canonicalExternalAnchorTypedRequestHash(typedRequest)
	if err != nil {
		return trustpolicy.Digest{}, "", fmt.Errorf("typed_request canonical hash failed: %w", err)
	}
	typedRequestHash, err := digestFromIdentity(typedRequestHashIdentity)
	if err != nil {
		return trustpolicy.Digest{}, "", fmt.Errorf("typed_request hash identity invalid: %w", err)
	}
	return typedRequestHash, typedRequestHashIdentity, nil
}

func canonicalExternalAnchorTypedRequestHash(request map[string]any) (string, error) {
	b, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	return policyengine.CanonicalHashBytes(b)
}

func externalAnchorCanonicalTargetDigest(typedRequest map[string]any) (trustpolicy.Digest, string, error) {
	primary, _, err := externalAnchorCanonicalTargetsFromTypedRequest(typedRequest)
	if err != nil {
		return trustpolicy.Digest{}, "", err
	}
	return primary.TargetDescriptorDigest, primary.TargetDescriptorIdentity, nil
}

func externalAnchorTargetKind(typedRequest map[string]any) string {
	return strings.TrimSpace(stringField(typedRequest, "target_kind"))
}

func externalAnchorSealDigest(typedRequest map[string]any) (trustpolicy.Digest, string, error) {
	seal, err := digestField(typedRequest, "seal_digest")
	if err != nil {
		return trustpolicy.Digest{}, "", fmt.Errorf("typed_request.seal_digest invalid: %w", err)
	}
	identity, _ := seal.Identity()
	return seal, identity, nil
}

func externalAnchorPayloadDigest(typedRequest map[string]any) (trustpolicy.Digest, string, error) {
	payload, err := digestField(typedRequest, "outbound_payload_digest")
	if err != nil {
		return trustpolicy.Digest{}, "", fmt.Errorf("typed_request.outbound_payload_digest invalid: %w", err)
	}
	identity, _ := payload.Identity()
	return payload, identity, nil
}

func externalAnchorDestinationRefFromTargetDescriptorDigest(digestIdentity string) string {
	trimmed := strings.TrimSpace(strings.TrimPrefix(digestIdentity, "sha256:"))
	if trimmed == "" {
		return ""
	}
	return "sha256/" + trimmed
}

func externalAnchorCanonicalTargetsFromTypedRequest(typedRequest map[string]any) (externalAnchorResolvedTarget, []externalAnchorResolvedTarget, error) {
	primary, err := externalAnchorPrimaryTargetFromTypedRequest(typedRequest)
	if err != nil {
		return externalAnchorResolvedTarget{}, nil, err
	}
	targetSet, err := externalAnchorTargetSetFromTypedRequest(typedRequest, primary)
	if err != nil {
		return externalAnchorResolvedTarget{}, nil, err
	}
	return primary, targetSet, nil
}

func externalAnchorPrimaryTargetFromTypedRequest(typedRequest map[string]any) (externalAnchorResolvedTarget, error) {
	targetKind := externalAnchorTargetKind(typedRequest)
	if targetKind == "" {
		return externalAnchorResolvedTarget{}, fmt.Errorf("typed_request.target_kind is required")
	}
	profile, err := externalAnchorImplementedProfile(targetKind)
	if err != nil {
		return externalAnchorResolvedTarget{}, err
	}
	requirement := trustpolicy.NormalizeExternalAnchorTargetRequirement(stringField(typedRequest, "target_requirement"))
	if err := trustpolicy.ValidateExternalAnchorTargetRequirement(requirement); err != nil {
		return externalAnchorResolvedTarget{}, fmt.Errorf("typed_request.target_requirement: %w", err)
	}
	descriptor, descriptorDigest, descriptorIdentity, err := externalAnchorCanonicalDescriptorBinding(typedRequest, "target_descriptor", "target_descriptor_digest")
	if err != nil {
		return externalAnchorResolvedTarget{}, err
	}
	return externalAnchorResolvedTarget{
		TargetKind:               targetKind,
		TargetRequirement:        requirement,
		TargetDescriptor:         descriptor,
		TargetDescriptorDigest:   descriptorDigest,
		TargetDescriptorIdentity: descriptorIdentity,
		RuntimeAdapter:           profile.runtimeAdapter,
		ReceiptKind:              profile.receiptKind,
		ProofKind:                profile.proofKind,
		ProofSchemaID:            profile.proofSchemaID,
	}, nil
}

func externalAnchorTargetSetFromTypedRequest(typedRequest map[string]any, primary externalAnchorResolvedTarget) ([]externalAnchorResolvedTarget, error) {
	entries, err := externalAnchorTypedRequestTargetSetEntries(typedRequest)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return []externalAnchorResolvedTarget{primary}, nil
	}
	out := make([]externalAnchorResolvedTarget, 0, len(entries))
	seen := map[string]struct{}{}
	hasPrimary := false
	for i := range entries {
		entry, isPrimary, err := externalAnchorResolvedTargetSetEntry(entries[i], i, primary, seen)
		if err != nil {
			return nil, err
		}
		hasPrimary = hasPrimary || isPrimary
		out = append(out, entry)
	}
	if !hasPrimary {
		return nil, fmt.Errorf("typed_request.target_set must include primary target_kind/target_descriptor_digest")
	}
	return sortExternalAnchorResolvedTargets(out), nil
}

func externalAnchorResolvedTargetSetEntry(raw any, index int, primary externalAnchorResolvedTarget, seen map[string]struct{}) (externalAnchorResolvedTarget, bool, error) {
	entryMap, ok := raw.(map[string]any)
	if !ok {
		return externalAnchorResolvedTarget{}, false, fmt.Errorf("typed_request.target_set[%d] invalid", index)
	}
	entry, err := externalAnchorTargetSetEntryFromTypedRequest(entryMap, index, primary)
	if err != nil {
		return externalAnchorResolvedTarget{}, false, err
	}
	key := externalAnchorResolvedTargetIdentityKey(entry)
	if _, dup := seen[key]; dup {
		return externalAnchorResolvedTarget{}, false, fmt.Errorf("typed_request.target_set[%d] duplicates target identity", index)
	}
	seen[key] = struct{}{}
	isPrimary := entry.TargetKind == primary.TargetKind && entry.TargetDescriptorIdentity == primary.TargetDescriptorIdentity
	if isPrimary && entry.TargetRequirement != primary.TargetRequirement {
		return externalAnchorResolvedTarget{}, false, fmt.Errorf("typed_request.target_set[%d].target_requirement conflicts with primary target_requirement", index)
	}
	return entry, isPrimary, nil
}

func externalAnchorTargetSetEntryFromTypedRequest(entry map[string]any, index int, primary externalAnchorResolvedTarget) (externalAnchorResolvedTarget, error) {
	prefix := fmt.Sprintf("typed_request.target_set[%d]", index)
	targetKind := strings.TrimSpace(stringField(entry, "target_kind"))
	if targetKind == "" {
		return externalAnchorResolvedTarget{}, fmt.Errorf("%s.target_kind is required", prefix)
	}
	profile, err := externalAnchorImplementedProfile(targetKind)
	if err != nil {
		return externalAnchorResolvedTarget{}, fmt.Errorf("%s.target_kind: %w", prefix, err)
	}
	requirement := trustpolicy.NormalizeExternalAnchorTargetRequirement(stringField(entry, "target_requirement"))
	if err := trustpolicy.ValidateExternalAnchorTargetRequirement(requirement); err != nil {
		return externalAnchorResolvedTarget{}, fmt.Errorf("%s.target_requirement: %w", prefix, err)
	}
	descriptor, descriptorDigest, descriptorIdentity, err := externalAnchorTargetSetEntryDescriptorBinding(entry, primary, prefix)
	if err != nil {
		return externalAnchorResolvedTarget{}, err
	}
	return externalAnchorResolvedTarget{
		TargetKind:               targetKind,
		TargetRequirement:        requirement,
		TargetDescriptor:         descriptor,
		TargetDescriptorDigest:   descriptorDigest,
		TargetDescriptorIdentity: descriptorIdentity,
		RuntimeAdapter:           profile.runtimeAdapter,
		ReceiptKind:              profile.receiptKind,
		ProofKind:                profile.proofKind,
		ProofSchemaID:            profile.proofSchemaID,
	}, nil
}

func externalAnchorTargetSetEntryDescriptorBinding(entry map[string]any, primary externalAnchorResolvedTarget, prefix string) (map[string]any, trustpolicy.Digest, string, error) {
	_, hasDescriptor := entry["target_descriptor"]
	if hasDescriptor {
		typed := map[string]any{
			"target_descriptor":        entry["target_descriptor"],
			"target_descriptor_digest": entry["target_descriptor_digest"],
		}
		descriptor, digest, identity, err := externalAnchorCanonicalDescriptorBinding(typed, "target_descriptor", "target_descriptor_digest")
		if err != nil {
			return nil, trustpolicy.Digest{}, "", fmt.Errorf("%s: %w", prefix, err)
		}
		return descriptor, digest, identity, nil
	}
	digest, identity, err := externalAnchorTargetDigestFromMap(entry, "target_descriptor_digest")
	if err != nil {
		return nil, trustpolicy.Digest{}, "", fmt.Errorf("%s target_descriptor_digest invalid: %w", prefix, err)
	}
	if strings.TrimSpace(stringField(entry, "target_kind")) == primary.TargetKind && identity == primary.TargetDescriptorIdentity {
		return cloneStringAnyMap(primary.TargetDescriptor), digest, identity, nil
	}
	return nil, trustpolicy.Digest{}, "", fmt.Errorf("%s.target_descriptor is required for non-primary targets", prefix)
}

func externalAnchorCanonicalDescriptorBinding(source map[string]any, descriptorField, digestFieldName string) (map[string]any, trustpolicy.Digest, string, error) {
	descriptorRaw, ok := source[descriptorField].(map[string]any)
	if !ok || len(descriptorRaw) == 0 {
		return nil, trustpolicy.Digest{}, "", fmt.Errorf("typed_request.%s is required", descriptorField)
	}
	descriptor := cloneStringAnyMap(descriptorRaw)
	canonicalIdentity, err := externalAnchorCanonicalDescriptorDigestIdentity(descriptor)
	if err != nil {
		return nil, trustpolicy.Digest{}, "", fmt.Errorf("typed_request.%s canonical hash failed: %w", descriptorField, err)
	}
	providedDigest, providedIdentity, err := externalAnchorTargetDigestFromMap(source, digestFieldName)
	if err != nil {
		return nil, trustpolicy.Digest{}, "", err
	}
	if providedIdentity != canonicalIdentity {
		return nil, trustpolicy.Digest{}, "", fmt.Errorf("typed_request.%s must match canonical digest of typed_request.%s", digestFieldName, descriptorField)
	}
	return descriptor, providedDigest, providedIdentity, nil
}

func externalAnchorCanonicalDescriptorDigestIdentity(descriptor map[string]any) (string, error) {
	b, err := json.Marshal(descriptor)
	if err != nil {
		return "", err
	}
	return policyengine.CanonicalHashBytes(b)
}

func externalAnchorTargetDigestFromMap(value map[string]any, field string) (trustpolicy.Digest, string, error) {
	d, err := digestField(value, field)
	if err != nil {
		return trustpolicy.Digest{}, "", fmt.Errorf("typed_request.%s invalid: %w", field, err)
	}
	identity, err := d.Identity()
	if err != nil {
		return trustpolicy.Digest{}, "", fmt.Errorf("typed_request.%s invalid: %w", field, err)
	}
	return d, identity, nil
}

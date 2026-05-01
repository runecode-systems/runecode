package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	externalAnchorRequestKindSubmitV0 = "external_anchor_submit_v0"
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
	ref, ok := typedRequest["target_descriptor_digest"].(map[string]any)
	if !ok {
		return trustpolicy.Digest{}, "", fmt.Errorf("typed_request.target_descriptor_digest is required")
	}
	d := trustpolicy.Digest{}
	if err := remarshalValue(ref, &d); err != nil {
		return trustpolicy.Digest{}, "", fmt.Errorf("typed_request.target_descriptor_digest invalid: %w", err)
	}
	identity, err := d.Identity()
	if err != nil {
		return trustpolicy.Digest{}, "", fmt.Errorf("typed_request.target_descriptor_digest invalid: %w", err)
	}
	return d, identity, nil
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

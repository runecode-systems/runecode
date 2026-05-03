package trustpolicy

import (
	"fmt"
	"strings"
)

func validateProviderInvocationReceiptPayload(receipt auditReceiptPayloadStrict) error {
	if receipt.ReceiptPayloadSchema != auditReceiptPayloadSchemaProviderInvocationV0 {
		return fmt.Errorf("%s receipts require provider invocation payload schema", receipt.AuditReceiptKind)
	}
	payload := providerInvocationReceiptPayload{}
	if err := unmarshalJSONStrict(receipt.ReceiptPayload, &payload); err != nil {
		return fmt.Errorf("decode provider invocation payload: %w", err)
	}
	if err := validateProviderInvocationOutcome(receipt.AuditReceiptKind, payload.AuthorizationOutcome); err != nil {
		return err
	}
	if err := validateProviderInvocationRequiredStrings(payload); err != nil {
		return err
	}
	if err := validateProviderInvocationNetworkTarget(payload.NetworkTarget, payload.DestinationKind); err != nil {
		return err
	}
	if err := validateNetworkTargetDigest(payload.NetworkTarget, payload.NetworkTargetDigest); err != nil {
		return err
	}
	if err := validateProviderInvocationOptionalDigests(payload); err != nil {
		return err
	}
	if err := validateRequestPayloadBinding(payload); err != nil {
		return err
	}
	return nil
}

func validateProviderInvocationRequiredStrings(payload providerInvocationReceiptPayload) error {
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "provider_kind", value: payload.ProviderKind},
		{name: "gateway_role_kind", value: payload.GatewayRoleKind},
		{name: "destination_kind", value: payload.DestinationKind},
		{name: "operation", value: payload.Operation},
	} {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("%s is required", field.name)
		}
	}
	return nil
}

func validateNetworkTargetDigest(target networkTargetDescriptor, digest Digest) error {
	computedTargetDigest, err := computeJSONCanonicalDigest(marshalJSONOrNull(target))
	if err != nil {
		return fmt.Errorf("network_target canonical digest: %w", err)
	}
	if mustDigestIdentity(digest) != mustDigestIdentity(computedTargetDigest) {
		return fmt.Errorf("network_target_digest does not match canonical network_target digest")
	}
	return nil
}

func validateProviderInvocationOptionalDigests(payload providerInvocationReceiptPayload) error {
	for _, field := range []struct {
		name   string
		digest *Digest
	}{
		{name: "request_digest", digest: payload.RequestDigest},
		{name: "response_digest", digest: payload.ResponseDigest},
		{name: "payload_digest", digest: payload.PayloadDigest},
		{name: "policy_decision_digest", digest: payload.PolicyDecisionDigest},
		{name: "allowlist_ref_digest", digest: payload.AllowlistRefDigest},
		{name: "lease_id_digest", digest: payload.LeaseIDDigest},
		{name: "run_id_digest", digest: payload.RunIDDigest},
	} {
		if err := validateOptionalReceiptDigestField(field.digest, field.name); err != nil {
			return err
		}
	}
	return nil
}

func validateRequestPayloadBinding(payload providerInvocationReceiptPayload) error {
	if payload.RequestPayloadBound == nil {
		return nil
	}
	if payload.RequestDigest == nil || payload.PayloadDigest == nil {
		return fmt.Errorf("request_payload_digest_bound requires request_digest and payload_digest")
	}
	if *payload.RequestPayloadBound != (mustDigestIdentity(*payload.RequestDigest) == mustDigestIdentity(*payload.PayloadDigest)) {
		return fmt.Errorf("request_payload_digest_bound does not match request/payload digest equality")
	}
	return nil
}

func validateProviderInvocationOutcome(kind, outcome string) error {
	outcome = strings.TrimSpace(outcome)
	if outcome != "authorized" && outcome != "denied" {
		return fmt.Errorf("authorization_outcome must be authorized or denied")
	}
	if kind == auditReceiptKindProviderInvocationAuthorized && outcome != "authorized" {
		return fmt.Errorf("authorization_outcome must be authorized for %s", kind)
	}
	if kind == auditReceiptKindProviderInvocationDenied && outcome != "denied" {
		return fmt.Errorf("authorization_outcome must be denied for %s", kind)
	}
	return nil
}

func validateProviderInvocationNetworkTarget(target networkTargetDescriptor, destinationKind string) error {
	if target.DescriptorSchemaID != networkTargetDescriptorSchemaGatewayDestinationV0 {
		return fmt.Errorf("network_target.descriptor_schema_id must be %q", networkTargetDescriptorSchemaGatewayDestinationV0)
	}
	if strings.TrimSpace(target.DestinationKind) == "" {
		return fmt.Errorf("network_target.destination_kind is required")
	}
	if strings.TrimSpace(destinationKind) != "" && target.DestinationKind != destinationKind {
		return fmt.Errorf("network_target.destination_kind must match destination_kind")
	}
	if strings.TrimSpace(target.Host) == "" && strings.TrimSpace(target.DestinationRef) == "" {
		return fmt.Errorf("network_target.host or network_target.destination_ref is required")
	}
	if target.Port != nil && (*target.Port < 1 || *target.Port > 65535) {
		return fmt.Errorf("network_target.port must be 1..65535 when provided")
	}
	return nil
}

package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func providerReceiptOutcome(decisionOutcome string) (string, string) {
	if strings.TrimSpace(strings.ToLower(decisionOutcome)) == "allow" {
		return "provider_invocation_authorized", "authorized"
	}
	return "provider_invocation_denied", "denied"
}

func providerInvocationReceiptPayloadMap(runID string, outcome string, decisionReason string, payload gatewayActionPayloadRuntime, match gatewayAllowlistMatch) (map[string]any, error) {
	networkTarget, err := providerNetworkTargetDescriptor(payload)
	if err != nil {
		return nil, err
	}
	networkTargetDigest, err := providerNetworkTargetDigest(networkTarget)
	if err != nil {
		return nil, err
	}
	receiptPayload := map[string]any{
		"authorization_outcome": outcome,
		"provider_kind":         providerKindForGatewayPayload(payload),
		"provider_profile_id":   strings.TrimSpace(payload.ProviderProfileID),
		"model_id":              strings.TrimSpace(payload.ModelID),
		"endpoint_identity":     strings.TrimSpace(payload.EndpointIdentity),
		"gateway_role_kind":     strings.TrimSpace(payload.GatewayRoleKind),
		"destination_kind":      strings.TrimSpace(payload.DestinationKind),
		"operation":             strings.TrimSpace(payload.Operation),
		"decision_reason_code":  strings.TrimSpace(decisionReason),
		"network_target":        networkTarget,
		"network_target_digest": networkTargetDigest,
		"run_id_digest":         hashIdentityDigest(strings.TrimSpace(runID)),
	}
	appendProviderInvocationAuditContext(receiptPayload, payload)
	appendProviderInvocationAllowlist(receiptPayload, match)
	if strings.TrimSpace(payload.ProviderProfileID) == "" {
		delete(receiptPayload, "provider_profile_id")
	}
	if strings.TrimSpace(payload.ModelID) == "" {
		delete(receiptPayload, "model_id")
	}
	if strings.TrimSpace(payload.EndpointIdentity) == "" {
		delete(receiptPayload, "endpoint_identity")
	}
	return receiptPayload, nil
}

func appendProviderInvocationAuditContext(receiptPayload map[string]any, payload gatewayActionPayloadRuntime) {
	leaseID := ""
	if payload.AuditContext != nil {
		leaseID = strings.TrimSpace(payload.AuditContext.LeaseID)
	}
	if leaseDigest := hashIdentityDigest(leaseID); leaseDigest != nil {
		receiptPayload["lease_id_digest"] = leaseDigest
	}
	if payload.PayloadHash != nil {
		receiptPayload["payload_digest"] = payload.PayloadHash
	}
	if payload.AuditContext == nil {
		return
	}
	if payload.AuditContext.RequestHash != nil {
		receiptPayload["request_digest"] = payload.AuditContext.RequestHash
	}
	if payload.AuditContext.ResponseHash != nil {
		receiptPayload["response_digest"] = payload.AuditContext.ResponseHash
	}
	if payload.AuditContext.PolicyDecisionHash != nil {
		receiptPayload["policy_decision_digest"] = payload.AuditContext.PolicyDecisionHash
	}
	if payload.PayloadHash != nil && payload.AuditContext.RequestHash != nil {
		receiptPayload["request_payload_digest_bound"] = mustDigestIdentity(*payload.PayloadHash) == mustDigestIdentity(*payload.AuditContext.RequestHash)
	}
}

func appendProviderInvocationAllowlist(receiptPayload map[string]any, match gatewayAllowlistMatch) {
	if digest := digestPointerFromIdentity(match.AllowlistRef); digest != nil {
		receiptPayload["allowlist_ref_digest"] = digest
	}
	if entryID := strings.TrimSpace(match.EntryID); entryID != "" {
		receiptPayload["allowlist_entry_id"] = entryID
	}
}

func providerNetworkTargetDescriptor(payload gatewayActionPayloadRuntime) (map[string]any, error) {
	host, port, path := runtimeParseDestinationRef(payload.DestinationRef)
	d := map[string]any{
		"descriptor_schema_id": "runecode.protocol.audit.network_target.gateway_destination.v0",
		"destination_kind":     strings.TrimSpace(payload.DestinationKind),
		"destination_ref":      strings.TrimSpace(payload.DestinationRef),
	}
	if host != "" {
		d["host"] = host
	}
	if path = strings.TrimSpace(path); path != "" {
		d["path_prefix"] = path
	}
	if port != nil {
		d["port"] = *port
	}
	if host == "" && strings.TrimSpace(payload.DestinationRef) == "" {
		return nil, fmt.Errorf("empty destination reference")
	}
	return d, nil
}

func providerNetworkTargetDigest(target map[string]any) (trustpolicy.Digest, error) {
	raw, err := json.Marshal(target)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	canonical, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	sum := sha256.Sum256(canonical)
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, nil
}

func secretLeaseReceiptPayloadMap(runID string, kind string, lease secretsd.Lease) map[string]any {
	action := "issued"
	if kind == "secret_lease_revoked" {
		action = "revoked"
	}
	receiptPayload := map[string]any{
		"lease_action":       action,
		"lease_id_digest":    hashIdentityDigest(lease.LeaseID),
		"secret_ref_digest":  hashIdentityDigest(lease.SecretRef),
		"consumer_id_digest": hashIdentityDigest(lease.ConsumerID),
		"role_kind":          strings.TrimSpace(lease.RoleKind),
		"scope_digest":       hashIdentityDigest(lease.Scope),
		"delivery_kind":      strings.TrimSpace(lease.DeliveryKind),
		"issued_at":          lease.IssuedAt.UTC().Format(time.RFC3339),
		"run_id_digest":      hashIdentityDigest(strings.TrimSpace(runID)),
	}
	if lease.RevokedAt != nil {
		receiptPayload["revoked_at"] = lease.RevokedAt.UTC().Format(time.RFC3339)
	}
	if reason := strings.TrimSpace(lease.Reason); reason != "" {
		receiptPayload["reason_digest"] = hashIdentityDigest(reason)
	}
	if lease.GitBinding != nil {
		receiptPayload["repository_identity_digest"] = hashIdentityDigest(lease.GitBinding.RepositoryIdentity)
		if actionHash := hashIdentityDigest(lease.GitBinding.ActionRequestHash); actionHash != nil {
			receiptPayload["action_request_digest"] = actionHash
		}
		if policyHash := hashIdentityDigest(lease.GitBinding.PolicyContextHash); policyHash != nil {
			receiptPayload["policy_context_digest"] = policyHash
		}
	}
	return receiptPayload
}

func digestPointerFromIdentity(identity string) *trustpolicy.Digest {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return nil
	}
	digest, err := digestFromIdentity(identity)
	if err != nil {
		return nil
	}
	return &digest
}

func hashIdentityDigest(value string) *trustpolicy.Digest {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(value))
	d := trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}
	return &d
}

func providerKindForGatewayPayload(payload gatewayActionPayloadRuntime) string {
	if strings.TrimSpace(payload.GatewayRoleKind) == "git-gateway" {
		return "git"
	}
	if strings.TrimSpace(payload.GatewayRoleKind) == "auth-gateway" {
		return "auth"
	}
	return "llm"
}

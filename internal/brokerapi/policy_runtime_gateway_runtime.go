package brokerapi

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type hostResolver interface {
	LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
}

type modelGatewayRuntime struct {
	resolver hostResolver
	quota    *gatewayQuotaBackend
	auditFn  func(eventType, actor string, details map[string]interface{}) error
}

func newModelGatewayRuntime(quota *gatewayQuotaBackend) *modelGatewayRuntime {
	if quota == nil {
		quota = newGatewayQuotaBackend()
	}
	return &modelGatewayRuntime{resolver: net.DefaultResolver, quota: quota}
}

func (g *modelGatewayRuntime) runtimeDestinationNetworkReason(payload gatewayActionPayloadRuntime, entry policyengine.GatewayScopeRule) (string, map[string]any) {
	host, port, _ := runtimeParseDestinationRef(payload.DestinationRef)
	if host == "" {
		return "runtime_gateway_destination_ref_invalid", nil
	}
	if reason, details := runtimeTLSPortReason(entry, port); reason != "" {
		return reason, details
	}
	if ip := net.ParseIP(host); ip != nil {
		if isDeniedRuntimeIP(ip) {
			return "runtime_gateway_private_range_destination_blocked", map[string]any{"ip": ip.String()}
		}
		return "", nil
	}
	return g.runtimeDNSReason(host)
}

func runtimeTLSPortReason(entry policyengine.GatewayScopeRule, port *int) (string, map[string]any) {
	if entry.Destination.TLSRequired && port != nil && *port == 80 {
		return "runtime_gateway_tls_required", map[string]any{"port": *port}
	}
	return "", nil
}

func (g *modelGatewayRuntime) runtimeDNSReason(host string) (string, map[string]any) {
	lookupCtx, cancel := context.WithTimeout(context.Background(), gatewayDNSLookupTimeout)
	defer cancel()
	ips, err := g.resolver.LookupIP(lookupCtx, "ip", host)
	if err != nil {
		return "runtime_gateway_dns_resolution_failed", map[string]any{"host": host, "error": err.Error()}
	}
	if len(ips) == 0 {
		return "runtime_gateway_dns_resolution_empty", map[string]any{"host": host}
	}
	for _, ip := range ips {
		if isDeniedRuntimeIP(ip) {
			return "runtime_gateway_dns_rebinding_or_private_ip_blocked", map[string]any{"host": host, "ip": ip.String()}
		}
	}
	return "", nil
}

func (g *modelGatewayRuntime) runtimeQuotaReason(runID string, payload gatewayActionPayloadRuntime) (string, map[string]any) {
	if payload.QuotaContext == nil {
		return "", nil
	}
	quotaKey := runtimeQuotaStateKey(runID, payload)
	reason, details, blocked := g.quota.evaluateAndApply(quotaKey, *payload.QuotaContext)
	if !blocked {
		return "", nil
	}
	if details == nil {
		details = map[string]any{}
	}
	details["quota_profile_kind"] = payload.QuotaContext.QuotaProfileKind
	details["quota_phase"] = payload.QuotaContext.Phase
	return reason, details
}

func runtimeQuotaStateKey(runID string, payload gatewayActionPayloadRuntime) string {
	return fmt.Sprintf("%s:%s:%s:%s:%s", runID, payload.GatewayRoleKind, payload.DestinationKind, payload.DestinationRef, payload.QuotaContext.QuotaProfileKind)
}

func (g *modelGatewayRuntime) releaseQuotaUsage(runID string, payload gatewayActionPayloadRuntime) {
	if g == nil || g.quota == nil || payload.QuotaContext == nil {
		return
	}
	if !gatewayAuditOutcomeReleasesConcurrency(payload.AuditContext) {
		return
	}
	if payload.QuotaContext.Meters.ConcurrencyUnits == nil || *payload.QuotaContext.Meters.ConcurrencyUnits <= 0 {
		return
	}
	g.quota.release(runtimeQuotaStateKey(runID, payload), gatewayQuotaRelease{ConcurrencyUnits: *payload.QuotaContext.Meters.ConcurrencyUnits})
}

func gatewayAuditOutcomeReleasesConcurrency(audit *gatewayAuditContextPayload) bool {
	if audit == nil {
		return false
	}
	switch audit.Outcome {
	case "admission_denied", "streaming_truncated", "succeeded", "failed", "timeout":
		return true
	default:
		return false
	}
}

func (g *modelGatewayRuntime) runtimeEnforcementDenyReason(runID string, entry policyengine.GatewayScopeRule, payload gatewayActionPayloadRuntime) (string, map[string]any, bool) {
	reason, details, denied := runtimeGatewayRuntimeDenyReason(entry, payload)
	if denied {
		g.releaseQuotaUsage(runID, payload)
		return reason, details, true
	}
	reason, details = g.runtimeDestinationNetworkReason(payload, entry)
	if reason != "" {
		g.releaseQuotaUsage(runID, payload)
		return reason, details, true
	}
	reason, details = g.runtimeQuotaReason(runID, payload)
	if reason != "" {
		g.releaseQuotaUsage(runID, payload)
		return reason, details, true
	}
	return "", nil, false
}

func (g *modelGatewayRuntime) emitGatewayAuditEvent(runID string, decision policyengine.PolicyDecision, payload gatewayActionPayloadRuntime, match gatewayAllowlistMatch) error {
	if payload.AuditContext == nil {
		if isGatewayRemoteMutationOperation(payload.Operation) {
			return fmt.Errorf("git remote mutation audit context required")
		}
		return nil
	}
	if g.auditFn == nil {
		return fmt.Errorf("gateway audit sink unavailable")
	}
	eventType := "model_egress"
	if payload.GatewayRoleKind == "auth-gateway" {
		eventType = "auth_egress"
	} else if payload.GatewayRoleKind == "git-gateway" {
		eventType = "git_egress"
	}
	details := gatewayAuditDetails(runID, decision, payload, match)
	return g.auditFn(eventType, "brokerapi", toInterfaceMap(details))
}

func gatewayAuditDetails(runID string, decision policyengine.PolicyDecision, payload gatewayActionPayloadRuntime, match gatewayAllowlistMatch) map[string]any {
	details := map[string]any{
		"run_id":              runID,
		"gateway_role_kind":   payload.GatewayRoleKind,
		"destination_kind":    payload.DestinationKind,
		"destination_ref":     payload.DestinationRef,
		"operation":           payload.Operation,
		"audit_outcome":       payload.AuditContext.Outcome,
		"outbound_bytes":      payload.AuditContext.OutboundBytes,
		"started_at":          payload.AuditContext.StartedAt,
		"completed_at":        payload.AuditContext.CompletedAt,
		"action_request_hash": decision.ActionRequestHash,
	}
	addGatewayQuotaAuditDetails(details, payload.QuotaContext)
	addGatewayDigestIdentity(details, "payload_hash", payload.PayloadHash)
	addGatewayDigestIdentity(details, "request_hash", payload.AuditContext.RequestHash)
	addGatewayDigestIdentity(details, "response_hash", payload.AuditContext.ResponseHash)
	if payload.PayloadHash != nil && payload.AuditContext.RequestHash != nil {
		addGatewayPayloadRequestBinding(details, payload.PayloadHash, payload.AuditContext.RequestHash)
	}
	if strings.TrimSpace(payload.AuditContext.LeaseID) != "" {
		details["lease_id"] = payload.AuditContext.LeaseID
	}
	addGatewayDigestIdentity(details, "policy_decision_hash", payload.AuditContext.PolicyDecisionHash)
	if payload.AuditContext != nil && payload.AuditContext.PolicyDecisionHash == nil {
		if decisionHash := firstString(decision.PolicyInputHashes); decisionHash != "" {
			details["policy_ref"] = decisionHash
		}
	}
	if strings.TrimSpace(match.AllowlistRef) != "" {
		details["matched_allowlist_ref"] = match.AllowlistRef
	}
	if strings.TrimSpace(match.EntryID) != "" {
		details["matched_allowlist_entry_id"] = match.EntryID
	}
	addGitRuntimeProofAuditDetails(details, payload)
	return details
}

func addGatewayDigestIdentityValue(details map[string]any, key string, digest trustpolicy.Digest) {
	identity, err := digest.Identity()
	if err != nil {
		return
	}
	details[key] = identity
}

func addGatewayQuotaAuditDetails(details map[string]any, quota *gatewayQuotaContextPayload) {
	if quota == nil {
		return
	}
	details["quota_profile_kind"] = quota.QuotaProfileKind
	details["quota_phase"] = quota.Phase
}

func addGatewayDigestIdentity(details map[string]any, key string, digest *trustpolicy.Digest) {
	if digest == nil {
		return
	}
	identity, err := digest.Identity()
	if err != nil {
		return
	}
	details[key] = identity
}

func addGatewayPayloadRequestBinding(details map[string]any, payloadHash *trustpolicy.Digest, requestHash *trustpolicy.Digest) {
	if payloadHash == nil || requestHash == nil {
		return
	}
	payloadIdentity, payloadErr := payloadHash.Identity()
	requestIdentity, requestErr := requestHash.Identity()
	if payloadErr != nil || requestErr != nil {
		return
	}
	details["request_payload_hash_bound"] = payloadIdentity == requestIdentity
}

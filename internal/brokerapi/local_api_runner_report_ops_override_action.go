package brokerapi

import (
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

type parsedOverrideActionDetails struct {
	ExpiresAt     string
	TicketRef     string
	OverrideMode  string
	Justification string
}

func overrideActionForResult(report RunnerResultReport, details map[string]any) (policyengine.ActionRequest, error) {
	if err := validateOverrideActionReportBinding(report); err != nil {
		return policyengine.ActionRequest{}, err
	}
	policyContextHash, err := requiredOverridePolicyContextHash(details, report.NormalizedInputDigests)
	if err != nil {
		return policyengine.ActionRequest{}, err
	}
	overrideDetails, err := parseOverrideActionDetails(details)
	if err != nil {
		return policyengine.ActionRequest{}, err
	}
	return policyengine.NewGateOverrideAction(policyengine.GateOverrideActionInput{
		ActionEnvelope: policyengine.ActionEnvelope{
			CapabilityID: "cap_gate_override",
			Actor: policyengine.ActionActor{
				ActorKind:  "daemon",
				RoleFamily: "workspace",
				RoleKind:   "workspace-edit",
			},
		},
		GateID:                    strings.TrimSpace(report.GateID),
		GateKind:                  strings.TrimSpace(report.GateKind),
		GateVersion:               strings.TrimSpace(report.GateVersion),
		GateAttemptID:             strings.TrimSpace(report.GateAttemptID),
		OverriddenFailedResultRef: strings.TrimSpace(report.OverriddenFailedResultRef),
		PolicyContextHash:         policyContextHash,
		OverrideMode:              overrideDetails.OverrideMode,
		Justification:             overrideDetails.Justification,
		ExpiresAt:                 overrideDetails.ExpiresAt,
		TicketRef:                 overrideDetails.TicketRef,
	}), nil
}

func validateOverrideActionReportBinding(report RunnerResultReport) error {
	if strings.TrimSpace(report.GateID) == "" || strings.TrimSpace(report.GateKind) == "" || strings.TrimSpace(report.GateVersion) == "" || strings.TrimSpace(report.GateAttemptID) == "" || strings.TrimSpace(report.OverriddenFailedResultRef) == "" {
		return fmt.Errorf("gate override action requires gate identity, gate_attempt_id, and overridden_failed_result_ref")
	}
	return nil
}

func requiredOverridePolicyContextHash(details map[string]any, normalizedInputDigests []string) (string, error) {
	policyContextHash, _ := details["policy_context_hash"].(string)
	policyContextHash = strings.TrimSpace(policyContextHash)
	if !isValidDigestIdentity(policyContextHash) {
		return "", fmt.Errorf("gate override result requires details.policy_context_hash digest")
	}
	if !hasPolicyContextDigest(details, normalizedInputDigests) {
		return "", fmt.Errorf("details.policy_context_hash must be present in normalized_input_digests")
	}
	return policyContextHash, nil
}

func parseOverrideActionDetails(details map[string]any) (parsedOverrideActionDetails, error) {
	expiresAt, err := requiredOverrideExpiry(details)
	if err != nil {
		return parsedOverrideActionDetails{}, err
	}
	ticketRef := optionalTrimmedDetail(details, "ticket_ref")
	overrideMode, err := parsedOverrideMode(details)
	if err != nil {
		return parsedOverrideActionDetails{}, err
	}
	justification, err := parsedOverrideJustification(details)
	if err != nil {
		return parsedOverrideActionDetails{}, err
	}
	if len(ticketRef) > 256 {
		return parsedOverrideActionDetails{}, fmt.Errorf("details.ticket_ref exceeds max length 256")
	}
	return parsedOverrideActionDetails{ExpiresAt: expiresAt, TicketRef: ticketRef, OverrideMode: overrideMode, Justification: justification}, nil
}

func requiredOverrideExpiry(details map[string]any) (string, error) {
	expiresAt := optionalTrimmedDetail(details, "override_expires_at")
	if expiresAt == "" {
		return "", fmt.Errorf("gate override result requires details.override_expires_at")
	}
	if _, err := time.Parse(time.RFC3339, expiresAt); err != nil {
		return "", fmt.Errorf("details.override_expires_at must be RFC3339")
	}
	return expiresAt, nil
}

func parsedOverrideMode(details map[string]any) (string, error) {
	overrideMode := optionalTrimmedDetail(details, "override_mode")
	if overrideMode == "" {
		overrideMode = "break_glass"
	}
	if overrideMode != "break_glass" && overrideMode != "temporary_allow" {
		return "", fmt.Errorf("details.override_mode must be one of: break_glass, temporary_allow")
	}
	return overrideMode, nil
}

func parsedOverrideJustification(details map[string]any) (string, error) {
	justification := optionalTrimmedDetail(details, "override_reason")
	if justification == "" {
		justification = "gate override continuation"
	}
	if len(justification) > 512 {
		return "", fmt.Errorf("details.override_reason exceeds max length 512")
	}
	return justification, nil
}

func optionalTrimmedDetail(details map[string]any, key string) string {
	value, _ := details[key].(string)
	return strings.TrimSpace(value)
}

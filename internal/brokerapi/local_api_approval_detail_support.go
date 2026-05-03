package brokerapi

import (
	"encoding/json"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func approvalBoundIdentityFromRecord(record approvalRecord, bindingKind, boundActionHash, boundStageSummaryHash string) ApprovalBoundIdentity {
	identity := ApprovalBoundIdentity{
		SchemaID:               "runecode.protocol.v0.ApprovalBoundIdentity",
		SchemaVersion:          "0.1.0",
		ApprovalRequestDigest:  strings.TrimSpace(record.Summary.RequestDigest),
		ApprovalDecisionDigest: strings.TrimSpace(record.Summary.DecisionDigest),
		PolicyDecisionHash:     strings.TrimSpace(record.Summary.PolicyDecisionHash),
		ManifestHash:           strings.TrimSpace(record.ManifestHash),
		RelevantArtifactHashes: append([]string{}, record.RelevantArtifactHashes...),
		BindingKind:            bindingKind,
		BoundActionHash:        strings.TrimSpace(boundActionHash),
		BoundStageSummaryHash:  strings.TrimSpace(boundStageSummaryHash),
	}
	if identity.ApprovalRequestDigest == "" {
		identity.ApprovalRequestDigest = strings.TrimSpace(record.Summary.ApprovalID)
	}
	if decision := approvalDecisionFromEnvelope(record.DecisionEnvelope); decision != nil {
		identity.DecisionApprover = &decision.Approver
	}
	if record.DecisionEnvelope != nil {
		identity.DecisionVerifierKeyID = strings.TrimSpace(record.DecisionEnvelope.Signature.KeyID)
		identity.DecisionVerifierKeyIDValue = strings.TrimSpace(record.DecisionEnvelope.Signature.KeyIDValue)
	}
	identity.ScopeDigest = strings.TrimSpace(record.Summary.ScopeDigest)
	identity.ArtifactSetDigest = strings.TrimSpace(record.Summary.ArtifactSetDigest)
	identity.DiffDigest = strings.TrimSpace(record.Summary.DiffDigest)
	identity.SummaryPreviewDigest = strings.TrimSpace(record.Summary.SummaryPreviewDigest)
	identity.ConsumedActionHash = strings.TrimSpace(record.Summary.ConsumedActionHash)
	identity.ConsumedArtifactDigest = strings.TrimSpace(record.Summary.ConsumedArtifactDigest)
	identity.ConsumptionLinkDigest = strings.TrimSpace(record.Summary.ConsumptionLinkDigest)
	return identity
}

func approvalEffectKindForActionKind(actionKind string) string {
	switch strings.TrimSpace(actionKind) {
	case "stage_summary_sign_off":
		return "stage_sign_off"
	case "promotion":
		return "promotion"
	case "backend_posture_change":
		return "backend_posture_change"
	default:
		return "action_execution"
	}
}

func approvalBlockedScopeKind(bound ApprovalBoundScope) string {
	if strings.TrimSpace(bound.StepID) != "" {
		return "step"
	}
	if strings.TrimSpace(bound.StageID) != "" {
		return "stage"
	}
	if strings.TrimSpace(bound.RunID) != "" {
		return "run"
	}
	if strings.TrimSpace(bound.WorkspaceID) != "" {
		return "workspace"
	}
	return "action_kind"
}

func approvalDecisionFromEnvelope(envelope *trustpolicy.SignedObjectEnvelope) *trustpolicy.ApprovalDecision {
	if envelope == nil {
		return nil
	}
	decoded, err := decodeApprovalDecision(*envelope)
	if err != nil {
		return nil
	}
	return &decoded
}

func stageSummaryHashForApprovalDetail(record approvalRecord) string {
	if record.RequestEnvelope == nil {
		return ""
	}
	payload := map[string]any{}
	if err := json.Unmarshal(record.RequestEnvelope.Payload, &payload); err != nil {
		return ""
	}
	details, _ := payload["details"].(map[string]any)
	if details == nil {
		return ""
	}
	digest, err := digestIdentityFromPayloadObject(details, "stage_summary_hash")
	if err != nil {
		return ""
	}
	return digest
}

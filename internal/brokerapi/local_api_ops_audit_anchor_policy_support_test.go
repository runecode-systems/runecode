package brokerapi

import (
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func mustSeedAnchorPolicyDecision(t *testing.T, service *Service, sealDigest trustpolicy.Digest, outcome policyengine.DecisionOutcome, requiredAssurance string) string {
	t.Helper()
	actionHash, err := anchorActionRequestHash(sealDigest)
	if err != nil {
		t.Fatalf("anchorActionRequestHash returned error: %v", err)
	}
	decision := seededAnchorPolicyDecision(actionHash, outcome)
	if outcome == policyengine.DecisionRequireHumanApproval {
		configureSeededAnchorRequiredApproval(&decision, requiredAssurance)
	}
	if err := service.RecordPolicyDecision(anchorApprovalPolicySelectorRunID, "", decision); err != nil {
		t.Fatalf("RecordPolicyDecision returned error: %v", err)
	}
	return mustFindAnchorPolicyDecisionRefForActionHash(t, service, actionHash)
}

func seededAnchorPolicyDecision(actionHash string, outcome policyengine.DecisionOutcome) policyengine.PolicyDecision {
	return policyengine.PolicyDecision{
		SchemaID:               "runecode.protocol.v0.PolicyDecision",
		SchemaVersion:          "0.3.0",
		DecisionOutcome:        outcome,
		PolicyReasonCode:       "allow_manifest_opt_in",
		ManifestHash:           "sha256:" + strings.Repeat("1", 64),
		ActionRequestHash:      actionHash,
		PolicyInputHashes:      []string{"sha256:" + strings.Repeat("2", 64)},
		RelevantArtifactHashes: []string{},
		DetailsSchemaID:        "runecode.protocol.details.policy.evaluation.v0",
		Details:                map[string]any{"precedence": "test_seed"},
	}
}

func configureSeededAnchorRequiredApproval(decision *policyengine.PolicyDecision, requiredAssurance string) {
	if decision == nil {
		return
	}
	decision.PolicyReasonCode = "approval_required"
	decision.RequiredApprovalSchemaID = "runecode.protocol.details.policy.required_approval.hard_floor.v0"
	required := map[string]any{
		"approval_trigger_code": "system_command_execution",
		"presence_mode":         "hardware_touch",
		"scope": map[string]any{
			"schema_id":      "runecode.protocol.v0.ApprovalBoundScope",
			"schema_version": "0.1.0",
			"run_id":         anchorApprovalPolicySelectorRunID,
			"action_kind":    policyengine.ActionKindPromotion,
		},
		"changes_if_approved":  "Approve anchor action for this exact seal digest.",
		"approval_ttl_seconds": 1800,
	}
	if strings.TrimSpace(requiredAssurance) != "" {
		required["approval_assurance_level"] = strings.TrimSpace(requiredAssurance)
	}
	decision.RequiredApproval = required
}

func mustFindAnchorPolicyDecisionRefForActionHash(t *testing.T, service *Service, actionHash string) string {
	t.Helper()
	latestRef := ""
	for _, ref := range service.PolicyDecisionRefsForRun(anchorApprovalPolicySelectorRunID) {
		rec, ok := service.PolicyDecisionGet(ref)
		if !ok {
			continue
		}
		if strings.TrimSpace(rec.ActionRequestHash) == actionHash {
			latestRef = ref
		}
	}
	if strings.TrimSpace(latestRef) == "" {
		t.Fatal("missing policy decision reference for anchor action")
	}
	return latestRef
}

func mustSeedConsumedApprovalForRequiredAnchorPolicy(t *testing.T, service *Service, sealDigest trustpolicy.Digest, requiredAssurance string) trustpolicy.Digest {
	t.Helper()
	policyRef := mustSeedAnchorPolicyDecision(t, service, sealDigest, policyengine.DecisionRequireHumanApproval, requiredAssurance)
	requestEnv, decisionEnv, verifiers := signedApprovalArtifactsForBrokerTestsWithOutcome(t, "human", "sha256:"+strings.Repeat("d", 64), "approve")
	for _, verifier := range verifiers {
		if err := putTrustedVerifierRecordForService(service, verifier); err != nil {
			t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
		}
	}
	approvalID := pendingApprovalIDForPolicyRef(service, policyRef)
	if strings.TrimSpace(approvalID) == "" {
		t.Fatal("missing derived pending approval for required anchor policy")
	}
	decisionDigestID := mustConsumeApprovalForAnchorPolicy(t, service, approvalID, requestEnv, decisionEnv)
	decisionDigest, err := digestFromIdentity(decisionDigestID)
	if err != nil {
		t.Fatalf("digestFromIdentity returned error: %v", err)
	}
	return decisionDigest
}

func pendingApprovalIDForPolicyRef(service *Service, policyRef string) string {
	for _, rec := range service.ApprovalList() {
		if strings.TrimSpace(rec.PolicyDecisionHash) == strings.TrimSpace(policyRef) && rec.Status == "pending" {
			return rec.ApprovalID
		}
	}
	return ""
}

func mustConsumeApprovalForAnchorPolicy(t *testing.T, service *Service, approvalID string, requestEnv *trustpolicy.SignedObjectEnvelope, decisionEnv *trustpolicy.SignedObjectEnvelope) string {
	t.Helper()
	stored, ok := service.ApprovalGet(approvalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing", approvalID)
	}
	stored.RequestEnvelope = requestEnv
	stored.Status = "consumed"
	stored.DecisionEnvelope = decisionEnv
	decisionDigestID, err := signedEnvelopeDigest(*decisionEnv)
	if err != nil {
		t.Fatalf("signedEnvelopeDigest returned error: %v", err)
	}
	now := time.Now().UTC()
	stored.DecisionDigest = decisionDigestID
	stored.DecidedAt = &now
	stored.ConsumedAt = &now
	if err := service.RecordApproval(stored); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	return decisionDigestID
}

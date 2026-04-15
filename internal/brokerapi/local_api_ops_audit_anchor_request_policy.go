package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

const anchorApprovalPolicySelectorRunID = "audit-anchor"

func (s *Service) anchorApprovalRequirement(sealDigest trustpolicy.Digest) (anchorApprovalRequirement, error) {
	actionHash, err := anchorActionRequestHash(sealDigest)
	if err != nil {
		return anchorApprovalRequirement{}, err
	}
	policyRef, decision, found := s.latestAnchorPolicyDecisionByActionHash(actionHash)
	if !found {
		return anchorApprovalRequirement{}, nil
	}
	return anchorApprovalRequirementFromDecision(policyRef, decision)
}

func anchorApprovalRequirementFromDecision(policyRef string, decision artifacts.PolicyDecisionRecord) (anchorApprovalRequirement, error) {
	switch strings.TrimSpace(decision.DecisionOutcome) {
	case string(policyengine.DecisionAllow):
		return anchorApprovalRequirement{}, nil
	case string(policyengine.DecisionRequireHumanApproval):
		return anchorApprovalRequirement{
			Required:          true,
			RequiredAssurance: requiredApprovalField(decision.RequiredApproval, "approval_assurance_level"),
			PolicyDecisionRef: strings.TrimSpace(policyRef),
		}, nil
	case string(policyengine.DecisionDeny):
		return anchorApprovalRequirement{}, errors.New("audit anchor denied by policy decision")
	default:
		return anchorApprovalRequirement{}, fmt.Errorf("unsupported anchor policy decision outcome %q", decision.DecisionOutcome)
	}
}

func (s *Service) latestAnchorPolicyDecisionByActionHash(actionHash string) (string, artifacts.PolicyDecisionRecord, bool) {
	if s == nil || s.store == nil {
		return "", artifacts.PolicyDecisionRecord{}, false
	}
	latestRef := ""
	latestRecord := artifacts.PolicyDecisionRecord{}
	for _, ref := range s.PolicyDecisionRefsForRun(anchorApprovalPolicySelectorRunID) {
		rec, ok := s.PolicyDecisionGet(ref)
		if !ok {
			continue
		}
		if strings.TrimSpace(rec.ActionRequestHash) != actionHash {
			continue
		}
		latestRef = ref
		latestRecord = rec
	}
	if strings.TrimSpace(latestRef) == "" {
		return "", artifacts.PolicyDecisionRecord{}, false
	}
	return latestRef, latestRecord, true
}

func requiredApprovalField(required map[string]any, key string) string {
	v, _ := required[key].(string)
	return strings.TrimSpace(v)
}

func anchorActionRequestHash(sealDigest trustpolicy.Digest) (string, error) {
	payload := map[string]any{
		"schema_id":      "runecode.brokerapi.anchor_action.v0",
		"schema_version": "0.1.0",
		"action_kind":    "audit_anchor_segment",
		"seal_digest":    map[string]any{"hash_alg": sealDigest.HashAlg, "hash": sealDigest.Hash},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (s *Service) resolveAnchorApprovalContext(req AuditAnchorSegmentRequest, required anchorApprovalRequirement) (*trustpolicy.Digest, *trustpolicy.ApprovalDecision, string, error) {
	requestedAssurance := strings.TrimSpace(req.ApprovalAssuranceLevel)
	if req.ApprovalDecisionDigest == nil {
		return resolveAnchorApprovalWithoutDecision(requestedAssurance, required)
	}
	decision, err := s.loadConsumedAnchorApprovalDecision(*req.ApprovalDecisionDigest, required)
	if err != nil {
		return nil, nil, "", err
	}
	assurance, err := resolveAnchorApprovalAssurance(requestedAssurance, required.RequiredAssurance, decision.ApprovalAssuranceLevel)
	if err != nil {
		return nil, nil, "", err
	}
	resolvedDigest := *req.ApprovalDecisionDigest
	return &resolvedDigest, &decision, assurance, nil
}

func resolveAnchorApprovalWithoutDecision(requestedAssurance string, required anchorApprovalRequirement) (*trustpolicy.Digest, *trustpolicy.ApprovalDecision, string, error) {
	if required.Required {
		return nil, nil, "", errors.New("approval decision digest is required by policy")
	}
	return nil, nil, requestedAssurance, nil
}

func (s *Service) loadConsumedAnchorApprovalDecision(decisionDigest trustpolicy.Digest, required anchorApprovalRequirement) (trustpolicy.ApprovalDecision, error) {
	decisionDigestIdentity, err := decisionDigest.Identity()
	if err != nil {
		return trustpolicy.ApprovalDecision{}, err
	}
	approval, found := s.findApprovalByDecisionDigest(decisionDigestIdentity)
	if !found {
		return trustpolicy.ApprovalDecision{}, errors.New("approval decision digest is not available")
	}
	if strings.TrimSpace(approval.Summary.Status) != "consumed" {
		return trustpolicy.ApprovalDecision{}, errors.New("approval decision is not consumed")
	}
	if required.Required && strings.TrimSpace(approval.Summary.PolicyDecisionHash) != strings.TrimSpace(required.PolicyDecisionRef) {
		return trustpolicy.ApprovalDecision{}, errors.New("approval decision is not bound to required policy decision")
	}
	if approval.DecisionEnvelope == nil {
		return trustpolicy.ApprovalDecision{}, errors.New("approval decision envelope is missing")
	}
	return decodeApprovalDecision(*approval.DecisionEnvelope)
}

func resolveAnchorApprovalAssurance(requested, required, decision string) (string, error) {
	derived := strings.TrimSpace(decision)
	if requested = strings.TrimSpace(requested); requested != "" && requested != derived {
		return "", errors.New("approval_assurance_level does not match approval decision")
	}
	if required = strings.TrimSpace(required); required != "" && derived != required {
		return "", errors.New("approval_assurance_level does not satisfy required policy assurance")
	}
	if requested != "" {
		return requested, nil
	}
	return derived, nil
}

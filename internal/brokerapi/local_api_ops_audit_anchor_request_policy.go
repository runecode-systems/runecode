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
		return anchorApprovalRequirement{PolicyDecisionRef: strings.TrimSpace(policyRef)}, nil
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
		if shouldReplaceAnchorPolicyDecision(latestRef, latestRecord, ref, rec) {
			latestRef = ref
			latestRecord = rec
		}
	}
	if strings.TrimSpace(latestRef) == "" {
		return "", artifacts.PolicyDecisionRecord{}, false
	}
	return latestRef, latestRecord, true
}

func shouldReplaceAnchorPolicyDecision(currentRef string, currentRecord artifacts.PolicyDecisionRecord, candidateRef string, candidateRecord artifacts.PolicyDecisionRecord) bool {
	if strings.TrimSpace(currentRef) == "" {
		return true
	}
	if candidateRecord.RecordedAt.After(currentRecord.RecordedAt) {
		return true
	}
	if candidateRecord.RecordedAt.Before(currentRecord.RecordedAt) {
		return false
	}
	if candidateRecord.AuditEventSeq > currentRecord.AuditEventSeq {
		return true
	}
	if candidateRecord.AuditEventSeq < currentRecord.AuditEventSeq {
		return false
	}
	if candidateRank, currentRank := anchorPolicyDecisionRestrictiveness(candidateRecord.DecisionOutcome), anchorPolicyDecisionRestrictiveness(currentRecord.DecisionOutcome); candidateRank > currentRank {
		return true
	} else if candidateRank < currentRank {
		return false
	}
	candidateRef = strings.TrimSpace(candidateRef)
	currentRef = strings.TrimSpace(currentRef)
	if candidateRef == "" {
		return false
	}
	return candidateRef < currentRef
}

func anchorPolicyDecisionRestrictiveness(outcome string) int {
	switch strings.TrimSpace(outcome) {
	case string(policyengine.DecisionAllow):
		return 1
	case string(policyengine.DecisionRequireHumanApproval):
		return 2
	case string(policyengine.DecisionDeny):
		return 3
	default:
		return 4
	}
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
	decision, err := s.loadConsumedAnchorApprovalDecision(*req.ApprovalDecisionDigest, req.SealDigest, required)
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
	if strings.TrimSpace(requestedAssurance) != "" {
		return nil, nil, "", errors.New("approval_assurance_level requires approval decision digest")
	}
	return nil, nil, requestedAssurance, nil
}

func (s *Service) loadConsumedAnchorApprovalDecision(decisionDigest trustpolicy.Digest, sealDigest trustpolicy.Digest, required anchorApprovalRequirement) (trustpolicy.ApprovalDecision, error) {
	decisionDigestIdentity, err := decisionDigest.Identity()
	if err != nil {
		return trustpolicy.ApprovalDecision{}, err
	}
	approval, err := s.lookupConsumedAnchorApproval(decisionDigestIdentity)
	if err != nil {
		return trustpolicy.ApprovalDecision{}, err
	}
	if err := validateConsumedAnchorApprovalBinding(approval, sealDigest, required); err != nil {
		return trustpolicy.ApprovalDecision{}, err
	}
	if approval.DecisionEnvelope == nil {
		return trustpolicy.ApprovalDecision{}, errors.New("approval decision envelope is missing")
	}
	return decodeApprovalDecision(*approval.DecisionEnvelope)
}

func (s *Service) lookupConsumedAnchorApproval(decisionDigestIdentity string) (approvalRecord, error) {
	approval, found := s.findApprovalByDecisionDigest(decisionDigestIdentity)
	if !found {
		return approvalRecord{}, errors.New("approval decision digest is not available")
	}
	if strings.TrimSpace(approval.Summary.Status) != "consumed" {
		return approvalRecord{}, errors.New("approval decision is not consumed")
	}
	return approval, nil
}

func validateConsumedAnchorApprovalBinding(approval approvalRecord, sealDigest trustpolicy.Digest, required anchorApprovalRequirement) error {
	expectedActionHash, err := anchorActionRequestHash(sealDigest)
	if err != nil {
		return err
	}
	if actionHash := strings.TrimSpace(approval.ActionRequestHash); actionHash == "" || actionHash != expectedActionHash {
		return errors.New("approval decision is not bound to current anchor action")
	}
	policyDecisionRef := strings.TrimSpace(required.PolicyDecisionRef)
	if required.Required && policyDecisionRef == "" {
		return errors.New("required policy decision ref is missing")
	}
	if policyDecisionRef != "" && strings.TrimSpace(approval.Summary.PolicyDecisionHash) != policyDecisionRef {
		return errors.New("approval decision is not bound to required policy decision")
	}
	return nil
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

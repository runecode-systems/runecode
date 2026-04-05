package artifacts

import (
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func validatePutRequest(req PutRequest, policy Policy) error {
	if _, ok := allDataClasses[req.DataClass]; !ok {
		return ErrInvalidDataClass
	}
	if isReservedDataClass(req.DataClass) && !policy.ReservedClassesEnabled {
		return ErrReservedDataClassDisabled
	}
	if req.ContentType == "" {
		return fmt.Errorf("content_type is required")
	}
	if !isValidDigest(req.ProvenanceReceiptHash) {
		return ErrInvalidDigest
	}
	return nil
}

func validatePolicy(policy Policy) error {
	if err := validatePolicyBasics(policy); err != nil {
		return err
	}
	return validateFlowMatrix(policy)
}

func validatePolicyBasics(policy Policy) error {
	if policy.HandOffReferenceMode != "hash_only" {
		return ErrHashOnlyHandoffRequired
	}
	if !policy.EncryptedAtRestDefault && !policy.DevPlaintextOverride {
		return fmt.Errorf("plaintext storage requires explicit dev override")
	}
	if policy.MaxPromotionRequestBytes <= 0 {
		return fmt.Errorf("max promotion request bytes must be positive")
	}
	if policy.MaxPromotionRequestsPerMinute <= 0 {
		return fmt.Errorf("max promotion requests per minute must be positive")
	}
	if policy.UnreferencedTTLSeconds <= 0 {
		return fmt.Errorf("unreferenced ttl must be positive")
	}
	return nil
}

func validateFlowMatrix(policy Policy) error {
	for _, rule := range policy.FlowMatrix {
		if err := validateFlowRule(policy, rule); err != nil {
			return err
		}
	}
	return nil
}

func validateFlowRule(policy Policy, rule FlowRule) error {
	if rule.ProducerRole == "" || rule.ConsumerRole == "" {
		return fmt.Errorf("flow rule roles cannot be empty")
	}
	for _, c := range rule.AllowedDataClasses {
		if _, ok := allDataClasses[c]; !ok {
			return ErrInvalidDataClass
		}
		if isReservedDataClass(c) && !policy.ReservedClassesEnabled {
			return ErrReservedDataClassDisabled
		}
	}
	return nil
}

func validateFlowInputs(policy Policy, req FlowCheckRequest) error {
	if policy.HandOffReferenceMode != "hash_only" {
		return ErrHashOnlyHandoffRequired
	}
	if _, ok := allDataClasses[req.DataClass]; !ok {
		return ErrInvalidDataClass
	}
	if isReservedDataClass(req.DataClass) && !policy.ReservedClassesEnabled {
		return ErrReservedDataClassDisabled
	}
	return nil
}

func enforceEgressRestrictions(policy Policy, req FlowCheckRequest, appendAudit func(string, string, map[string]interface{}) error) error {
	if err := enforceUnapprovedEgressRestriction(policy, req, appendAudit); err != nil {
		return err
	}
	if err := enforceApprovedManifestOptIn(policy, req, appendAudit); err != nil {
		return err
	}
	if err := enforceApprovedRevocationRestriction(policy, req, appendAudit); err != nil {
		return err
	}
	return nil
}

func enforceUnapprovedEgressRestriction(policy Policy, req FlowCheckRequest, appendAudit func(string, string, map[string]interface{}) error) error {
	if !(req.IsEgress && req.DataClass == DataClassUnapprovedFileExcerpts && policy.UnapprovedExcerptEgressDenied) {
		return nil
	}
	if err := appendAudit("artifact_flow_blocked", req.ProducerRole, map[string]interface{}{"reason": "unapproved_excerpt_egress_denied", "digest": req.Digest}); err != nil {
		return err
	}
	return ErrUnapprovedEgressDenied
}

func enforceApprovedManifestOptIn(policy Policy, req FlowCheckRequest, appendAudit func(string, string, map[string]interface{}) error) error {
	if !(req.IsEgress && req.DataClass == DataClassApprovedFileExcerpts && policy.ApprovedExcerptEgressOptInOnly && !req.ManifestOptIn) {
		return nil
	}
	if err := appendAudit("artifact_flow_blocked", req.ProducerRole, map[string]interface{}{"reason": "approved_excerpt_requires_manifest_opt_in", "digest": req.Digest}); err != nil {
		return err
	}
	return ErrApprovedEgressRequiresManifest
}

func enforceApprovedRevocationRestriction(policy Policy, req FlowCheckRequest, appendAudit func(string, string, map[string]interface{}) error) error {
	if !(req.DataClass == DataClassApprovedFileExcerpts && policy.RevokedApprovedExcerptHashes[req.Digest]) {
		return nil
	}
	if err := appendAudit("artifact_flow_blocked", req.ProducerRole, map[string]interface{}{"reason": "approved_excerpt_revoked", "digest": req.Digest}); err != nil {
		return err
	}
	return ErrApprovedExcerptRevoked
}

func flowAllowed(rules []FlowRule, req FlowCheckRequest) bool {
	matchedRolePair := false
	for _, rule := range rules {
		if rule.ProducerRole != req.ProducerRole || rule.ConsumerRole != req.ConsumerRole {
			continue
		}
		matchedRolePair = true
		for _, cls := range rule.AllowedDataClasses {
			if cls == req.DataClass {
				return true
			}
		}
	}
	if !matchedRolePair {
		return false
	}
	return false
}

func validatePromotionRequest(policy Policy, source ArtifactRecord, req PromotionRequest, trustedVerifiers []trustpolicy.VerifierRecord) error {
	if source.Reference.DataClass != DataClassUnapprovedFileExcerpts {
		return ErrPromotionSourceNotUnapproved
	}
	if err := validatePromotionApproval(policy, req); err != nil {
		return err
	}
	if err := verifySignedApprovalDecision(req, trustedVerifiers); err != nil {
		return err
	}
	if err := validatePromotionMetadata(req); err != nil {
		return err
	}
	if policy.MaxPromotionRequestBytes > 0 && source.Reference.SizeBytes > policy.MaxPromotionRequestBytes {
		return ErrPromotionTooLarge
	}
	return nil
}

func validatePromotionApproval(policy Policy, req PromotionRequest) error {
	if policy.ExplicitHumanApprovalRequired && req.Approver == "" {
		return ErrPromotionRequiresApproval
	}
	if policy.BulkPromotionRequiresSeparateReview && req.BulkRequest && !req.BulkApprovalConfirmed {
		return ErrApprovalBulkConfirmationNeeded
	}
	if policy.RequireFullContentVisibility && !req.FullContentVisible && !req.ExplicitViewFull {
		return ErrApprovalContentNotVisible
	}
	return nil
}

func validatePromotionMetadata(req PromotionRequest) error {
	if req.RepoPath == "" || req.Commit == "" || req.ExtractorToolVersion == "" {
		return ErrMissingApprovalMetadata
	}
	return nil
}

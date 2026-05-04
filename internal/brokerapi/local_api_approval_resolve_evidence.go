package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type resolvedApprovalEvidenceFields struct {
	requestedAt            string
	changesIfApproved      string
	scopeDigest            string
	artifactSetDigest      string
	diffDigest             string
	summaryPreviewDigest   string
	consumedActionHash     string
	consumedArtifactDigest string
	consumptionLinkDigest  string
}

func resolvedApprovalEvidence(req ApprovalResolveRequest, prior approvalRecord, approvalID, decisionDigest, status, now string) resolvedApprovalEvidenceFields {
	fields := resolvedApprovalEvidenceFields{
		requestedAt:       fallbackApprovalRequestedAt(prior, now),
		changesIfApproved: fallbackApprovalChangesIfApproved(prior),
	}
	fields.consumedActionHash, fields.consumedArtifactDigest = resolvedApprovalConsumptionTargets(prior, status)
	fields.summaryPreviewDigest = fallbackApprovalSummaryPreviewDigest(prior, req)
	fields.scopeDigest = fallbackApprovalScopeDigest(prior)
	fields.artifactSetDigest = fallbackApprovalArtifactSetDigest(prior)
	fields.diffDigest = fallbackApprovalDiffDigest(prior)
	fields.consumptionLinkDigest = fallbackApprovalConsumptionLinkDigest(prior, approvalID, decisionDigest, status, fields.consumedActionHash, fields.consumedArtifactDigest)
	return fields
}

func fallbackApprovalRequestedAt(prior approvalRecord, now string) string {
	if prior.Summary.RequestedAt != "" {
		return prior.Summary.RequestedAt
	}
	return now
}

func fallbackApprovalChangesIfApproved(prior approvalRecord) string {
	if prior.Summary.ChangesIfApproved != "" {
		return prior.Summary.ChangesIfApproved
	}
	return approvalChangesIfApprovedDefault
}

func resolvedApprovalConsumptionTargets(prior approvalRecord, status string) (string, string) {
	consumedActionHash := strings.TrimSpace(prior.Summary.ConsumedActionHash)
	consumedArtifactDigest := strings.TrimSpace(prior.Summary.ConsumedArtifactDigest)
	if strings.TrimSpace(status) != "consumed" {
		return consumedActionHash, consumedArtifactDigest
	}
	if consumedActionHash == "" {
		consumedActionHash = strings.TrimSpace(prior.ActionRequestHash)
	}
	if consumedArtifactDigest == "" {
		consumedArtifactDigest = strings.TrimSpace(prior.SourceDigest)
	}
	return consumedActionHash, consumedArtifactDigest
}

func fallbackApprovalSummaryPreviewDigest(prior approvalRecord, req ApprovalResolveRequest) string {
	if summaryPreviewDigest := strings.TrimSpace(prior.Summary.SummaryPreviewDigest); summaryPreviewDigest != "" {
		return summaryPreviewDigest
	}
	return approvalSummaryPreviewDigest(reqSignedApprovalRequestPtr(req))
}

func fallbackApprovalScopeDigest(prior approvalRecord) string {
	if scopeDigest := strings.TrimSpace(prior.Summary.ScopeDigest); scopeDigest != "" {
		return scopeDigest
	}
	return approvalScopeDigest(prior.Summary.BoundScope)
}

func fallbackApprovalArtifactSetDigest(prior approvalRecord) string {
	if artifactSetDigest := strings.TrimSpace(prior.Summary.ArtifactSetDigest); artifactSetDigest != "" {
		return artifactSetDigest
	}
	return approvalArtifactSetDigest(prior.RelevantArtifactHashes)
}

func fallbackApprovalDiffDigest(prior approvalRecord) string {
	if diffDigest := strings.TrimSpace(prior.Summary.DiffDigest); diffDigest != "" {
		return diffDigest
	}
	return approvalDiffDigest(prior.SourceDigest)
}

func fallbackApprovalConsumptionLinkDigest(prior approvalRecord, approvalID, decisionDigest, status, consumedActionHash, consumedArtifactDigest string) string {
	if strings.TrimSpace(status) != "consumed" {
		return strings.TrimSpace(prior.Summary.ConsumptionLinkDigest)
	}
	return approvalConsumptionLinkDigest(ApprovalSummary{
		ApprovalID:             approvalID,
		RequestDigest:          strings.TrimSpace(prior.Summary.RequestDigest),
		DecisionDigest:         strings.TrimSpace(decisionDigest),
		ConsumedActionHash:     consumedActionHash,
		ConsumedArtifactDigest: consumedArtifactDigest,
	})
}

func reqSignedApprovalRequestPtr(req ApprovalResolveRequest) *trustpolicy.SignedObjectEnvelope {
	if len(req.SignedApprovalRequest.Payload) == 0 {
		return nil
	}
	env := req.SignedApprovalRequest
	return &env
}

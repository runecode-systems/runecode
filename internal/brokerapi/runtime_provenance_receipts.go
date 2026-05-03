package brokerapi

import (
	"log"
	"strings"

	"github.com/runecode-ai/runecode/internal/secretsd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	runtimeProvenanceVerifierPurpose = "audit_anchor"
	runtimeProvenanceVerifierScope   = "node"
	runtimeSummaryScopeRun           = "run"
	runtimeSummaryBoundaryRoute      = "artifact_io_promotion"

	auditReceiptPayloadSchemaMetaAuditActionV0 = "runecode.protocol.audit.receipt.meta_audit_action.v0"
	auditReceiptKindEvidenceBundleExport       = "evidence_bundle_export"
	auditReceiptKindEvidenceImport             = "evidence_import"
	auditReceiptKindEvidenceRestore            = "evidence_restore"
	auditReceiptKindRetentionPolicyChanged     = "retention_policy_changed"
	auditReceiptKindArchivalOperation          = "archival_operation"
	auditReceiptKindSensitiveEvidenceView      = "sensitive_evidence_view"
)

func (s *Service) persistProviderInvocationReceipt(runID string, decisionOutcome string, decisionReason string, payload gatewayActionPayloadRuntime, match gatewayAllowlistMatch) {
	if s == nil || s.auditLedger == nil {
		return
	}
	kind, outcome := providerReceiptOutcome(decisionOutcome)
	sealDigest, ok := s.currentRuntimeProvenanceSealDigest()
	if !ok {
		return
	}
	receiptPayload, err := providerInvocationReceiptPayloadMap(runID, outcome, decisionReason, payload, match)
	if err != nil {
		return
	}
	if err := s.persistRuntimeProvenanceReceipt(sealDigest, kind, trustpolicy.AuditSegmentAnchoringSubjectSeal, "runecode.protocol.audit.receipt.provider_invocation.v0", receiptPayload); err != nil {
		log.Printf("brokerapi: runtime provenance receipt persistence failed for %s: %v", strings.TrimSpace(kind), err)
	}
}

func (s *Service) persistSecretLeaseReceipt(runID string, kind string, lease secretsd.Lease) {
	if s == nil || s.auditLedger == nil {
		return
	}
	sealDigest, ok := s.currentRuntimeProvenanceSealDigest()
	if !ok {
		return
	}
	receiptPayload := secretLeaseReceiptPayloadMap(runID, kind, lease)
	if err := s.persistRuntimeProvenanceReceipt(sealDigest, kind, trustpolicy.AuditSegmentAnchoringSubjectSeal, "runecode.protocol.audit.receipt.secret_lease.v0", receiptPayload); err != nil {
		log.Printf("brokerapi: runtime provenance receipt persistence failed for %s: %v", strings.TrimSpace(kind), err)
	}
}

package brokerapi

import (
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) recordExternalAnchorAuditArtifacts(requestID string, exportReceiptCopy bool, record artifacts.ExternalAnchorPreparedMutationRecord) *ErrorResponse {
	if s == nil || s.auditLedger == nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit ledger unavailable")
		return &errOut
	}
	if strings.TrimSpace(record.LastExecuteAttemptSealDigest) == "" {
		return nil
	}
	proofDigest, _, _, errResp := s.persistExternalAnchorEvidenceArtifacts(requestID, &record)
	if errResp != nil {
		return errResp
	}
	if errResp := s.persistCompletedExternalAnchorReceiptArtifacts(requestID, exportReceiptCopy, &record, proofDigest); errResp != nil {
		return errResp
	}
	if err := s.ExternalAnchorPreparedUpsert(record); err != nil {
		errOut := s.makeError(requestID, "broker_storage_write_failed", "storage", false, err.Error())
		return &errOut
	}
	return nil
}

func (s *Service) persistExternalAnchorEvidenceArtifacts(requestID string, record *artifacts.ExternalAnchorPreparedMutationRecord) (trustpolicy.Digest, *trustpolicy.Digest, *trustpolicy.Digest, *ErrorResponse) {
	proofDigest, providerReceiptDigest, transcriptDigest, err := s.persistExternalAnchorProofSidecars(*record)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("external anchor sidecar persistence failed: %v", err))
		return trustpolicy.Digest{}, nil, nil, &errOut
	}
	evidenceReq, err := s.buildExternalAnchorEvidenceRequest(*record, proofDigest, providerReceiptDigest, transcriptDigest)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("external anchor evidence request invalid: %v", err))
		return trustpolicy.Digest{}, nil, nil, &errOut
	}
	evidenceDigest, _, err := s.auditLedger.PersistExternalAnchorEvidence(evidenceReq)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("external anchor evidence persistence failed: %v", err))
		return trustpolicy.Digest{}, nil, nil, &errOut
	}
	record.LastAnchorEvidenceDigest, _ = evidenceDigest.Identity()
	record.LastAnchorProofDigest, _ = proofDigest.Identity()
	if providerReceiptDigest != nil {
		record.LastAnchorProviderReceipt, _ = providerReceiptDigest.Identity()
	}
	if transcriptDigest != nil {
		record.LastAnchorTranscriptDigest, _ = transcriptDigest.Identity()
	}
	return proofDigest, providerReceiptDigest, transcriptDigest, nil
}

func (s *Service) persistCompletedExternalAnchorReceiptArtifacts(requestID string, exportReceiptCopy bool, record *artifacts.ExternalAnchorPreparedMutationRecord, proofDigest trustpolicy.Digest) *ErrorResponse {
	if strings.TrimSpace(record.ExecutionState) != gitRemoteMutationExecutionCompleted {
		return nil
	}
	receiptDigest, verificationDigest, err := s.persistExternalAnchorReceiptAndVerify(*record, proofDigest)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("external anchor receipt persistence failed: %v", err))
		return &errOut
	}
	record.LastAnchorReceiptDigest, _ = receiptDigest.Identity()
	record.LastAnchorVerificationDigest, _ = verificationDigest.Identity()
	if exportReceiptCopy {
		logExternalAnchorReceiptCopyExportFailure(requestID, receiptDigest, s.putAnchorReceiptExportArtifact)
	}
	return nil
}

func logExternalAnchorReceiptCopyExportFailure(requestID string, receiptDigest trustpolicy.Digest, putArtifact func(trustpolicy.Digest, string) (artifacts.ArtifactReference, error)) {
	if digestID, err := receiptDigest.Identity(); err == nil {
		if _, err := putArtifact(receiptDigest, digestID); err != nil {
			logAuditAnchorExportCopyFailure(requestID, receiptDigest, "artifact_put_failed", err)
		}
		return
	}
	logAuditAnchorExportCopyFailure(requestID, receiptDigest, "receipt_digest_invalid", fmt.Errorf("receipt digest identity invalid"))
}

func (s *Service) buildExternalAnchorEvidenceRequest(record artifacts.ExternalAnchorPreparedMutationRecord, proofDigest trustpolicy.Digest, providerReceiptDigest, transcriptDigest *trustpolicy.Digest) (auditd.ExternalAnchorEvidenceRequest, error) {
	core, err := externalAnchorEvidenceRequestCore(record)
	if err != nil {
		return auditd.ExternalAnchorEvidenceRequest{}, err
	}
	attestationDigest, projectContextDigest := s.externalAnchorEvidenceOptionalRefs(record.RunID)
	outcome, startedAt, completedAt := externalAnchorEvidenceOutcomeAndTiming(record)
	return auditd.ExternalAnchorEvidenceRequest{
		RunID:                    strings.TrimSpace(record.RunID),
		PreparedMutationID:       strings.TrimSpace(record.PreparedMutationID),
		ExecutionAttemptID:       strings.TrimSpace(record.LastExecuteAttemptID),
		CanonicalTargetKind:      core.TargetKind,
		CanonicalTargetDigest:    core.TargetDigest,
		CanonicalTargetIdentity:  core.TargetIdentity,
		TargetRequirement:        core.TargetRequirement,
		AnchoringSubjectFamily:   trustpolicy.AuditSegmentAnchoringSubjectSeal,
		AnchoringSubjectDigest:   core.SealDigest,
		OutboundPayloadDigest:    &core.OutboundDigest,
		OutboundBytes:            core.OutboundBytes,
		StartedAtRFC3339:         startedAt,
		CompletedAtRFC3339:       completedAt,
		Outcome:                  outcome,
		OutcomeReasonCode:        strings.TrimSpace(record.ExecutionReasonCode),
		TypedRequestHash:         mustDigestPtr(record.TypedRequestHash),
		ActionRequestHash:        mustDigestPtr(record.ActionRequestHash),
		PolicyDecisionHash:       mustDigestPtr(record.PolicyDecisionHash),
		TargetAuthLeaseID:        strings.TrimSpace(record.LastExecuteTargetAuthLeaseID),
		RequiredApprovalID:       strings.TrimSpace(record.RequiredApprovalID),
		ApprovalRequestHash:      mustDigestPtr(record.LastExecuteApprovalReqID),
		ApprovalDecisionHash:     mustDigestPtr(record.LastExecuteApprovalDecID),
		AttestationEvidenceRef:   attestationDigest,
		ProjectContextIdentity:   projectContextDigest,
		ProofDigest:              proofDigest,
		ProofKind:                externalAnchorProofKindForTargetKind(core.TargetKind),
		ProofSchemaID:            externalAnchorProofSchemaForTargetKind(core.TargetKind),
		ProviderReceiptDigest:    providerReceiptDigest,
		VerificationTranscriptID: transcriptDigest,
	}, nil
}

type externalAnchorEvidenceRequestCoreFields struct {
	TargetKind        string
	TargetIdentity    string
	TargetRequirement string
	TargetDigest      trustpolicy.Digest
	SealDigest        trustpolicy.Digest
	OutboundDigest    trustpolicy.Digest
	OutboundBytes     int64
}

func externalAnchorEvidenceRequestCore(record artifacts.ExternalAnchorPreparedMutationRecord) (externalAnchorEvidenceRequestCoreFields, error) {
	targetDigest, targetIdentity, err := externalAnchorCanonicalTargetDigest(record.TypedRequest)
	if err != nil {
		return externalAnchorEvidenceRequestCoreFields{}, err
	}
	sealDigest, _, err := externalAnchorSealDigest(record.TypedRequest)
	if err != nil {
		return externalAnchorEvidenceRequestCoreFields{}, err
	}
	outbound, _, err := externalAnchorPayloadDigest(record.TypedRequest)
	if err != nil {
		return externalAnchorEvidenceRequestCoreFields{}, err
	}
	return externalAnchorEvidenceRequestCoreFields{TargetKind: strings.TrimSpace(stringField(record.TypedRequest, "target_kind")), TargetIdentity: targetIdentity, TargetRequirement: strings.TrimSpace(stringField(record.TypedRequest, "target_requirement")), TargetDigest: targetDigest, SealDigest: sealDigest, OutboundDigest: outbound, OutboundBytes: int64(intField(record.TypedRequest, "outbound_bytes"))}, nil
}

func (s *Service) externalAnchorEvidenceOptionalRefs(runID string) (*trustpolicy.Digest, *trustpolicy.Digest) {
	return externalAnchorRuntimeAttestationDigest(s.RuntimeEvidence(runID)), externalAnchorProjectContextDigest(s.projectSubstrate.Snapshot.ProjectContextIdentityDigest)
}

func externalAnchorRuntimeAttestationDigest(evidence launcherbackend.RuntimeEvidenceSnapshot) *trustpolicy.Digest {
	if evidence.Attestation == nil {
		return nil
	}
	return mustDigestPtr(evidence.Attestation.EvidenceDigest)
}

func externalAnchorProjectContextDigest(identity string) *trustpolicy.Digest {
	return mustDigestPtr(identity)
}

func externalAnchorEvidenceOutcomeAndTiming(record artifacts.ExternalAnchorPreparedMutationRecord) (string, string, string) {
	state := strings.TrimSpace(record.ExecutionState)
	startedAt := record.UpdatedAt.UTC().Format(time.RFC3339)
	completedAt := ""
	if state == gitRemoteMutationExecutionCompleted || state == gitRemoteMutationExecutionFailed || state == gitRemoteMutationExecutionBlocked {
		completedAt = startedAt
	}
	switch state {
	case gitRemoteMutationExecutionCompleted:
		return trustpolicy.ExternalAnchorOutcomeCompleted, startedAt, completedAt
	case gitRemoteMutationExecutionBlocked:
		return trustpolicy.ExternalAnchorOutcomeUnavailable, startedAt, completedAt
	case gitRemoteMutationExecutionFailed:
		return trustpolicy.ExternalAnchorOutcomeInvalid, startedAt, completedAt
	default:
		return trustpolicy.ExternalAnchorOutcomeDeferred, startedAt, completedAt
	}
}

func mustDigestPtr(identity string) *trustpolicy.Digest {
	d, err := digestFromIdentity(strings.TrimSpace(identity))
	if err != nil {
		return nil
	}
	return &d
}

func (s *Service) persistExternalAnchorProofSidecars(record artifacts.ExternalAnchorPreparedMutationRecord) (trustpolicy.Digest, *trustpolicy.Digest, *trustpolicy.Digest, error) {
	proofPayload := map[string]any{
		"schema_id":                "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0",
		"schema_version":           "0.1.0",
		"prepared_mutation_id":     strings.TrimSpace(record.PreparedMutationID),
		"execution_attempt_id":     strings.TrimSpace(record.LastExecuteAttemptID),
		"target_descriptor_digest": record.TypedRequest["target_descriptor_digest"],
		"seal_digest":              record.TypedRequest["seal_digest"],
		"outbound_payload_digest":  record.TypedRequest["outbound_payload_digest"],
	}
	proofDigest, err := s.auditLedger.PersistExternalAnchorSidecar(trustpolicy.ExternalAnchorSidecarKindProofBytes, proofPayload)
	if err != nil {
		return trustpolicy.Digest{}, nil, nil, err
	}
	providerPayload := map[string]any{
		"schema_id":             "runecode.protocol.audit.anchor_provider_receipt.v0",
		"schema_version":        "0.1.0",
		"execution_state":       strings.TrimSpace(record.ExecutionState),
		"execution_reason_code": strings.TrimSpace(record.ExecutionReasonCode),
	}
	providerDigest, err := s.auditLedger.PersistExternalAnchorSidecar(trustpolicy.ExternalAnchorSidecarKindProviderReceipt, providerPayload)
	if err != nil {
		return trustpolicy.Digest{}, nil, nil, err
	}
	transcriptPayload := map[string]any{
		"schema_id":           "runecode.protocol.audit.anchor_verification_transcript.v0",
		"schema_version":      "0.1.0",
		"checked_bindings":    []string{"seal_digest", "target_descriptor_digest", "typed_request_hash", "approval", "lease"},
		"verification_result": strings.TrimSpace(record.ExecutionState),
	}
	transcriptDigest, err := s.auditLedger.PersistExternalAnchorSidecar(trustpolicy.ExternalAnchorSidecarKindVerifyTranscript, transcriptPayload)
	if err != nil {
		return trustpolicy.Digest{}, nil, nil, err
	}
	return proofDigest, &providerDigest, &transcriptDigest, nil
}

package brokerapi

import (
	"context"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) HandleAuditEvidenceSnapshotGet(ctx context.Context, req AuditEvidenceSnapshotGetRequest, meta RequestContext) (AuditEvidenceSnapshotGetResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.prepareAuditEvidenceRequest(ctx, req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditEvidenceSnapshotGetRequestSchemaPath, meta, "audit evidence snapshot service unavailable")
	if errResp != nil {
		return AuditEvidenceSnapshotGetResponse{}, errResp
	}
	defer cleanup()
	if errResp := s.requireAuditEvidenceLedger(requestID); errResp != nil {
		return AuditEvidenceSnapshotGetResponse{}, errResp
	}
	snapshot, errResp := s.loadProjectedAuditEvidenceSnapshot(requestID)
	if errResp != nil {
		return AuditEvidenceSnapshotGetResponse{}, errResp
	}
	resp := AuditEvidenceSnapshotGetResponse{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceSnapshotGetResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Snapshot:      snapshot,
	}
	if err := s.validateResponse(resp, auditEvidenceSnapshotGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditEvidenceSnapshotGetResponse{}, &errOut
	}
	s.persistMetaAuditReceipt(auditReceiptKindSensitiveEvidenceView, "audit_evidence_snapshot", nil, nil, nil, "evidence_snapshot")
	return resp, nil
}

func (s *Service) loadProjectedAuditEvidenceSnapshot(requestID string) (AuditEvidenceSnapshot, *ErrorResponse) {
	trustedSnapshot, err := s.auditLedger.EvidenceSnapshotWithIdentity(s.auditEvidenceIdentityContext())
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence snapshot lookup failed")
		return AuditEvidenceSnapshot{}, &errOut
	}
	snapshot, err := projectAuditEvidenceSnapshot(trustedSnapshot)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence snapshot projection failed")
		return AuditEvidenceSnapshot{}, &errOut
	}
	if err := s.validateResponse(snapshot, auditEvidenceSnapshotSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditEvidenceSnapshot{}, &errOut
	}
	return snapshot, nil
}

func projectAuditEvidenceSnapshot(snapshot auditd.AuditEvidenceSnapshot) (AuditEvidenceSnapshot, error) {
	projected, err := projectAuditSnapshotDigestFamilies(snapshot)
	if err != nil {
		return AuditEvidenceSnapshot{}, err
	}
	repositoryIdentityDigest, err := optionalDigestFromAuditIdentity(snapshot.RepositoryIdentityDigest)
	if err != nil {
		return AuditEvidenceSnapshot{}, err
	}
	return AuditEvidenceSnapshot{
		SchemaID:                      "runecode.protocol.v0.AuditEvidenceSnapshot",
		SchemaVersion:                 "0.1.0",
		CreatedAt:                     snapshot.CreatedAt,
		RepositoryIdentityDigest:      repositoryIdentityDigest,
		ProductInstanceID:             snapshot.ProductInstanceID,
		LedgerIdentity:                snapshot.LedgerIdentity,
		SegmentIDs:                    snapshot.SegmentIDs,
		SegmentSealDigests:            projected.segmentSealDigests,
		AuditReceiptDigests:           projected.auditReceiptDigests,
		VerificationReportDigests:     projected.verificationReportDigests,
		RuntimeEvidenceDigests:        projected.runtimeEvidenceDigests,
		VerifierRecordDigests:         projected.verifierRecordDigests,
		EventContractCatalogDigests:   projected.eventContractCatalogDigests,
		SignerEvidenceDigests:         projected.signerEvidenceDigests,
		StoragePostureDigests:         projected.storagePostureDigests,
		TypedRequestDigests:           projected.typedRequestDigests,
		ActionRequestDigests:          projected.actionRequestDigests,
		ControlPlaneDigests:           projected.controlPlaneDigests,
		AttestationEvidenceDigests:    projected.attestationEvidenceDigests,
		ProjectContextIdentityDigests: projected.projectContextIdentityDigests,
		PolicyEvidenceDigests:         projected.policyEvidenceDigests,
		RequiredApprovalIDs:           snapshot.RequiredApprovalIDs,
		ApprovalEvidenceDigests:       projected.approvalEvidenceDigests,
		AnchorEvidenceDigests:         projected.anchorEvidenceDigests,
		ProviderInvocationDigests:     projected.providerInvocationDigests,
		SecretLeaseDigests:            projected.secretLeaseDigests,
	}, nil
}

type projectedAuditSnapshotDigestFamilies struct {
	segmentSealDigests            []trustpolicy.Digest
	auditReceiptDigests           []trustpolicy.Digest
	verificationReportDigests     []trustpolicy.Digest
	runtimeEvidenceDigests        []trustpolicy.Digest
	verifierRecordDigests         []trustpolicy.Digest
	eventContractCatalogDigests   []trustpolicy.Digest
	signerEvidenceDigests         []trustpolicy.Digest
	storagePostureDigests         []trustpolicy.Digest
	typedRequestDigests           []trustpolicy.Digest
	actionRequestDigests          []trustpolicy.Digest
	controlPlaneDigests           []trustpolicy.Digest
	attestationEvidenceDigests    []trustpolicy.Digest
	projectContextIdentityDigests []trustpolicy.Digest
	policyEvidenceDigests         []trustpolicy.Digest
	approvalEvidenceDigests       []trustpolicy.Digest
	anchorEvidenceDigests         []trustpolicy.Digest
	providerInvocationDigests     []trustpolicy.Digest
	secretLeaseDigests            []trustpolicy.Digest
}

func projectAuditSnapshotDigestFamilies(snapshot auditd.AuditEvidenceSnapshot) (projectedAuditSnapshotDigestFamilies, error) {
	projected := projectedAuditSnapshotDigestFamilies{}
	for _, family := range projectedAuditSnapshotDigestFamilyTargets(&projected, snapshot) {
		digests, err := projectAuditSnapshotDigests(family.identities)
		if err != nil {
			return projectedAuditSnapshotDigestFamilies{}, err
		}
		*family.target = digests
	}
	return projected, nil
}

type projectedAuditSnapshotDigestFamilyTarget struct {
	identities []string
	target     *[]trustpolicy.Digest
}

func projectedAuditSnapshotDigestFamilyTargets(projected *projectedAuditSnapshotDigestFamilies, snapshot auditd.AuditEvidenceSnapshot) []projectedAuditSnapshotDigestFamilyTarget {
	return []projectedAuditSnapshotDigestFamilyTarget{
		{identities: snapshot.SegmentSealDigests, target: &projected.segmentSealDigests},
		{identities: snapshot.AuditReceiptDigests, target: &projected.auditReceiptDigests},
		{identities: snapshot.VerificationReportDigests, target: &projected.verificationReportDigests},
		{identities: snapshot.RuntimeEvidenceDigests, target: &projected.runtimeEvidenceDigests},
		{identities: snapshot.VerifierRecordDigests, target: &projected.verifierRecordDigests},
		{identities: snapshot.EventContractCatalogDigests, target: &projected.eventContractCatalogDigests},
		{identities: snapshot.SignerEvidenceDigests, target: &projected.signerEvidenceDigests},
		{identities: snapshot.StoragePostureDigests, target: &projected.storagePostureDigests},
		{identities: snapshot.TypedRequestDigests, target: &projected.typedRequestDigests},
		{identities: snapshot.ActionRequestDigests, target: &projected.actionRequestDigests},
		{identities: snapshot.ControlPlaneDigests, target: &projected.controlPlaneDigests},
		{identities: snapshot.AttestationEvidenceDigests, target: &projected.attestationEvidenceDigests},
		{identities: snapshot.ProjectContextIdentityDigests, target: &projected.projectContextIdentityDigests},
		{identities: snapshot.PolicyEvidenceDigests, target: &projected.policyEvidenceDigests},
		{identities: snapshot.ApprovalEvidenceDigests, target: &projected.approvalEvidenceDigests},
		{identities: snapshot.AnchorEvidenceDigests, target: &projected.anchorEvidenceDigests},
		{identities: snapshot.ProviderInvocationDigests, target: &projected.providerInvocationDigests},
		{identities: snapshot.SecretLeaseDigests, target: &projected.secretLeaseDigests},
	}
}

func projectAuditSnapshotDigests(identities []string) ([]trustpolicy.Digest, error) {
	if len(identities) == 0 {
		return nil, nil
	}
	out := make([]trustpolicy.Digest, 0, len(identities))
	for i := range identities {
		digest, err := digestFromIdentity(identities[i])
		if err != nil {
			return nil, err
		}
		out = append(out, digest)
	}
	return out, nil
}

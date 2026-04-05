package artifacts

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func verifySignedApprovalDecision(req PromotionRequest, trustedVerifiers []trustpolicy.VerifierRecord) error {
	if req.ApprovalDecision == nil {
		return ErrApprovalArtifactRequired
	}
	verifiers, err := resolveTrustedVerifiersForEnvelope(*req.ApprovalDecision, trustedVerifiers)
	if err != nil {
		return err
	}
	registry, err := trustpolicy.NewVerifierRegistry(verifiers)
	if err != nil {
		return errors.Join(ErrApprovalVerificationFailed, err)
	}
	if err := trustpolicy.VerifySignedEnvelope(*req.ApprovalDecision, registry, trustpolicy.EnvelopeVerificationOptions{
		RequirePayloadSchemaMatch: true,
		ExpectedPayloadSchemaID:   trustpolicy.ApprovalDecisionSchemaID,
		ExpectedPayloadVersion:    trustpolicy.ApprovalDecisionSchemaVersion,
	}); err != nil {
		return errors.Join(ErrApprovalVerificationFailed, err)
	}
	decision := trustpolicy.ApprovalDecision{}
	if err := json.Unmarshal(req.ApprovalDecision.Payload, &decision); err != nil {
		return errors.Join(ErrApprovalVerificationFailed, err)
	}
	if err := trustpolicy.ValidateApprovalDecisionEvidence(decision); err != nil {
		return errors.Join(ErrApprovalVerificationFailed, err)
	}
	if decision.DecisionOutcome != "approve" {
		return errors.Join(ErrApprovalVerificationFailed, errors.New("approval decision outcome is not approve"))
	}
	if decision.Approver.PrincipalID == "" {
		return errors.Join(ErrApprovalVerificationFailed, errors.New("approval decision approver principal is required"))
	}
	if req.Approver != "" && req.Approver != decision.Approver.PrincipalID {
		return errors.Join(ErrApprovalVerificationFailed, errors.New("promotion approver does not match approval decision approver"))
	}
	return nil
}

func resolveTrustedVerifiersForEnvelope(envelope trustpolicy.SignedObjectEnvelope, trustedVerifiers []trustpolicy.VerifierRecord) ([]trustpolicy.VerifierRecord, error) {
	if len(trustedVerifiers) == 0 {
		return nil, errors.Join(ErrApprovalVerificationFailed, ErrVerifierNotFound)
	}
	wantKeyIDValue := envelope.Signature.KeyIDValue
	for _, record := range trustedVerifiers {
		if record.KeyID != envelope.Signature.KeyID {
			continue
		}
		if record.KeyIDValue != wantKeyIDValue {
			continue
		}
		if record.Status != "active" {
			return nil, errors.Join(ErrApprovalVerificationFailed, fmt.Errorf("trusted verifier %s is not active", record.KeyIDValue))
		}
		publicKeyBytes, err := record.PublicKey.DecodedBytes()
		if err != nil {
			return nil, errors.Join(ErrApprovalVerificationFailed, err)
		}
		reencoded := base64.StdEncoding.EncodeToString(publicKeyBytes)
		verified := record
		verified.PublicKey.Value = reencoded
		return []trustpolicy.VerifierRecord{verified}, nil
	}
	return nil, errors.Join(ErrApprovalVerificationFailed, ErrVerifierNotFound)
}

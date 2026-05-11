package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func validateApprovalResolveInput(resp brokerapi.ApprovalGetResponse) error {
	approvalID := strings.TrimSpace(resp.Approval.ApprovalID)
	if approvalID == "" {
		return fmt.Errorf("approval detail missing approval_id")
	}
	if resp.SignedApprovalRequest == nil || resp.SignedApprovalDecision == nil {
		return fmt.Errorf("approval resolve requires signed approval request and decision envelopes")
	}
	boundScope := resp.Approval.BoundScope
	actionKind := strings.TrimSpace(boundScope.ActionKind)
	switch actionKind {
	case "backend_posture_change":
		if err := validateApprovalResolveEnvelopeBinding(resp); err != nil {
			return err
		}
		selection := resp.ApprovalDetail.BackendPostureSelection
		if selection == nil {
			return fmt.Errorf("approval resolve requires typed backend posture selection detail")
		}
		if strings.TrimSpace(selection.TargetInstanceID) == "" || strings.TrimSpace(selection.TargetBackendKind) == "" {
			return fmt.Errorf("approval resolve requires backend posture target instance and backend kind")
		}
	case "promotion":
		return fmt.Errorf("resolve unavailable: promotion approvals must be completed in the promotion flow so exact promotion binding stays intact")
	default:
		return fmt.Errorf("resolve unavailable: this approval action kind is not supported by the current TUI resolve flow")
	}
	return nil
}

func approvalResolveUnavailableReason(err error) string {
	if err == nil {
		return "Resolve unavailable."
	}
	text := safeUIErrorText(err)
	text = strings.TrimSpace(strings.TrimPrefix(text, "resolve unavailable:"))
	if text == "" {
		return "Resolve unavailable."
	}
	return "Resolve unavailable: " + text + "."
}

func validateApprovalResolveEnvelopeBinding(resp brokerapi.ApprovalGetResponse) error {
	// The TUI performs only a consistency guard between broker-projected signed
	// envelopes. Signature authenticity remains broker-owned and is revalidated
	// by ApprovalResolve before any authority-bearing mutation is accepted.
	requestID, err := approvalRequestDigestIdentity(*resp.SignedApprovalRequest)
	if err != nil {
		return fmt.Errorf("approval resolve request envelope invalid: %w", err)
	}
	if err := approvalDecisionMatchesRequest(*resp.SignedApprovalDecision, requestID); err != nil {
		return fmt.Errorf("approval resolve decision envelope invalid: %w", err)
	}
	return nil
}

func approvalRequestDigestIdentity(envelope trustpolicy.SignedObjectEnvelope) (string, error) {
	if envelope.PayloadSchemaID != trustpolicy.ApprovalRequestSchemaID {
		return "", fmt.Errorf("unexpected request payload schema")
	}
	digest, err := digestIdentityFromEnvelopePayload(envelope.Payload)
	if err != nil {
		return "", err
	}
	return digest, nil
}

func approvalDecisionMatchesRequest(envelope trustpolicy.SignedObjectEnvelope, requestID string) error {
	if envelope.PayloadSchemaID != trustpolicy.ApprovalDecisionSchemaID {
		return fmt.Errorf("unexpected decision payload schema")
	}
	if len(envelope.Payload) == 0 {
		return fmt.Errorf("decision payload missing")
	}
	payload := map[string]json.RawMessage{}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return fmt.Errorf("decode decision payload: %w", err)
	}
	rawDigest, ok := payload["approval_request_hash"]
	if !ok {
		return fmt.Errorf("approval_request_hash missing from decision payload")
	}
	var digest trustpolicy.Digest
	if err := json.Unmarshal(rawDigest, &digest); err != nil {
		return fmt.Errorf("decode approval_request_hash: %w", err)
	}
	identity, err := digest.Identity()
	if err != nil {
		return fmt.Errorf("invalid approval_request_hash: %w", err)
	}
	if identity != requestID {
		return fmt.Errorf("decision payload does not match approval request: decision=%s request=%s", shortIdentity(identity), shortIdentity(requestID))
	}
	return nil
}

func digestIdentityFromEnvelopePayload(payload json.RawMessage) (string, error) {
	hexDigest, err := canonicalJSONPayload(payload)
	if err != nil {
		return "", err
	}
	return (trustpolicy.Digest{HashAlg: "sha256", Hash: hexDigest}).Identity()
}

func canonicalJSONPayload(payload json.RawMessage) (string, error) {
	transformed, err := jsoncanonicalizerTransform(payload)
	if err != nil {
		return "", fmt.Errorf("canonicalize approval request payload: %w", err)
	}
	return sha256HexDigest(transformed), nil
}

func jsoncanonicalizerTransform(payload json.RawMessage) ([]byte, error) {
	return jsoncanonicalizer.Transform(payload)
}

func sha256HexDigest(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func approvalResolveRequestFromDetail(resp brokerapi.ApprovalGetResponse) (brokerapi.ApprovalResolveRequest, error) {
	if err := validateApprovalResolveInput(resp); err != nil {
		return brokerapi.ApprovalResolveRequest{}, err
	}
	summary := resp.Approval
	boundScope := summary.BoundScope
	if strings.TrimSpace(boundScope.SchemaID) == "" {
		return brokerapi.ApprovalResolveRequest{}, fmt.Errorf("approval resolve requires broker-provided bound scope schema id")
	}
	if strings.TrimSpace(boundScope.SchemaVersion) == "" {
		return brokerapi.ApprovalResolveRequest{}, fmt.Errorf("approval resolve requires broker-provided bound scope schema version")
	}
	resolveReq := brokerapi.ApprovalResolveRequest{
		SchemaID:               "runecode.protocol.v0.ApprovalResolveRequest",
		SchemaVersion:          localAPISchemaVersion,
		ApprovalID:             strings.TrimSpace(summary.ApprovalID),
		BoundScope:             boundScope,
		ResolutionDetails:      brokerapi.ApprovalResolveDetails{SchemaID: "runecode.protocol.v0.ApprovalResolveDetails", SchemaVersion: "0.1.0"},
		SignedApprovalRequest:  *resp.SignedApprovalRequest,
		SignedApprovalDecision: *resp.SignedApprovalDecision,
	}
	if strings.TrimSpace(boundScope.ActionKind) == "backend_posture_change" && resp.ApprovalDetail.BackendPostureSelection != nil {
		selection := resp.ApprovalDetail.BackendPostureSelection
		resolveReq.ResolutionDetails.BackendPostureSelection = &brokerapi.ApprovalResolveBackendPostureSelectionDetail{
			SchemaID:          "runecode.protocol.v0.ApprovalResolveBackendPostureSelectionDetail",
			SchemaVersion:     "0.1.0",
			TargetInstanceID:  strings.TrimSpace(selection.TargetInstanceID),
			TargetBackendKind: strings.TrimSpace(selection.TargetBackendKind),
		}
	}
	return resolveReq, nil
}

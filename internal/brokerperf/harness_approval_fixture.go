package brokerperf

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func seedBackendPostureApprovalForResolve(service *brokerapi.Service) (brokerapi.ApprovalResolveRequest, error) {
	targetInstanceID, targetBackend, actionHash, err := backendPostureApprovalFixtureInputs(service)
	if err != nil {
		return brokerapi.ApprovalResolveRequest{}, err
	}
	requestEnv, approvalID, verifierRecord, privateKey, err := backendPostureApprovalRequestEnvelope(targetInstanceID, targetBackend, actionHash)
	if err != nil {
		return brokerapi.ApprovalResolveRequest{}, err
	}
	decisionEnv, err := backendPostureApprovalDecisionEnvelope(approvalID, verifierRecord, privateKey)
	if err != nil {
		return brokerapi.ApprovalResolveRequest{}, err
	}
	if err := putTrustedVerifierRecordForService(service, verifierRecord); err != nil {
		return brokerapi.ApprovalResolveRequest{}, err
	}
	policyHash, err := persistBackendPostureApprovalFixture(service, targetInstanceID, actionHash, approvalID, requestEnv)
	if err != nil {
		return brokerapi.ApprovalResolveRequest{}, err
	}
	return brokerapi.ApprovalResolveRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalResolveRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "perf-approval-resolve",
		ApprovalID:    approvalID,
		BoundScope: brokerapi.ApprovalBoundScope{
			SchemaID:           "runecode.protocol.v0.ApprovalBoundScope",
			SchemaVersion:      "0.1.0",
			WorkspaceID:        "workspace-local",
			InstanceID:         targetInstanceID,
			RunID:              "run-backend",
			ActionKind:         policyengine.ActionKindBackendPosture,
			PolicyDecisionHash: policyHash,
		},
		ResolutionDetails:      brokerapi.ApprovalResolveDetails{SchemaID: "runecode.protocol.v0.ApprovalResolveDetails", SchemaVersion: "0.1.0", BackendPostureSelection: &brokerapi.ApprovalResolveBackendPostureSelectionDetail{SchemaID: "runecode.protocol.v0.ApprovalResolveBackendPostureSelectionDetail", SchemaVersion: "0.1.0", TargetInstanceID: targetInstanceID, TargetBackendKind: targetBackend}},
		SignedApprovalRequest:  requestEnv,
		SignedApprovalDecision: decisionEnv,
	}, nil
}

func backendPostureApprovalFixtureInputs(service *brokerapi.Service) (string, string, string, error) {
	postureResp, errResp := service.HandleBackendPostureGet(context.Background(), brokerapi.BackendPostureGetRequest{SchemaID: "runecode.protocol.v0.BackendPostureGetRequest", SchemaVersion: "0.1.0", RequestID: "seed-posture"}, brokerapi.RequestContext{})
	if errResp != nil {
		return "", "", "", fmt.Errorf("backend_posture_get: %s", errResp.Error.Code)
	}
	targetInstanceID := postureResp.Posture.InstanceID
	targetBackend := "container"
	actionHash, err := policyengine.CanonicalActionRequestHash(policyengine.NewBackendPostureChangeAction(policyengine.BackendPostureChangeActionInput{
		ActionEnvelope:               policyengine.ActionEnvelope{CapabilityID: "cap_backend", Actor: policyengine.ActionActor{ActorKind: "daemon", RoleFamily: "workspace", RoleKind: "workspace-edit"}},
		RunID:                        "instance-control:" + targetInstanceID,
		TargetInstanceID:             targetInstanceID,
		TargetBackendKind:            targetBackend,
		SelectionMode:                "explicit_selection",
		ChangeKind:                   "select_backend",
		AssuranceChangeKind:          "reduce_assurance",
		OptInKind:                    "exact_action_approval",
		ReducedAssuranceAcknowledged: true,
		Reason:                       "operator_requested_reduced_assurance_backend_opt_in",
	}))
	if err != nil {
		return "", "", "", err
	}
	return targetInstanceID, targetBackend, actionHash, nil
}

func backendPostureApprovalRequestEnvelope(targetInstanceID, targetBackend, actionHash string) (trustpolicy.SignedObjectEnvelope, string, trustpolicy.VerifierRecord, ed25519.PrivateKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, "", trustpolicy.VerifierRecord{}, nil, err
	}
	keyIDValue := backendPostureKeyIDValue(publicKey)
	requestBytes, err := marshalBackendPostureRequestPayload(targetInstanceID, targetBackend, actionHash, keyIDValue)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, "", trustpolicy.VerifierRecord{}, nil, err
	}
	requestCanonical, err := jsoncanonicalizer.Transform(requestBytes)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, "", trustpolicy.VerifierRecord{}, nil, err
	}
	requestSig := ed25519.Sign(privateKey, requestCanonical)
	approvalID := backendPostureApprovalID(requestCanonical)
	verifier := backendPostureVerifierRecord(publicKey, keyIDValue)
	return backendPostureRequestEnvelope(requestBytes, keyIDValue, requestSig), approvalID, verifier, privateKey, nil
}

func backendPostureKeyIDValue(publicKey ed25519.PublicKey) string {
	keyID := sha256.Sum256(publicKey)
	return hex.EncodeToString(keyID[:])
}

func marshalBackendPostureRequestPayload(targetInstanceID, targetBackend, actionHash, keyIDValue string) ([]byte, error) {
	payload := map[string]any{
		"schema_id":                trustpolicy.ApprovalRequestSchemaID,
		"schema_version":           trustpolicy.ApprovalRequestSchemaVersion,
		"approval_profile":         "moderate",
		"requester":                map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "daemon", "principal_id": "broker", "instance_id": "broker-1"},
		"approval_trigger_code":    "reduced_assurance_backend",
		"manifest_hash":            map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("1", 64)},
		"action_request_hash":      map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(actionHash, "sha256:")},
		"relevant_artifact_hashes": []any{},
		"details_schema_id":        "runecode.protocol.details.policy.required_approval.reduced_assurance_backend.v0",
		"details":                  backendPostureRequestDetails(targetInstanceID, targetBackend),
		"approval_assurance_level": "reauthenticated",
		"presence_mode":            "hardware_touch",
		"requested_at":             time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
		"expires_at":               time.Now().UTC().Add(30 * time.Minute).Format(time.RFC3339),
		"staleness_posture":        "invalidate_on_bound_input_change",
		"changes_if_approved":      "Reduced-assurance backend posture change may be applied.",
		"signatures":               []any{map[string]any{"alg": "ed25519", "key_id": trustpolicy.KeyIDProfile, "key_id_value": keyIDValue, "signature": base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))}},
	}
	return json.Marshal(payload)
}

func backendPostureRequestDetails(targetInstanceID, targetBackend string) map[string]any {
	return map[string]any{
		"target_instance_id":             targetInstanceID,
		"target_backend_kind":            targetBackend,
		"selection_mode":                 "explicit_selection",
		"change_kind":                    "select_backend",
		"requested_posture":              "container_mode_explicit_opt_in",
		"assurance_change_kind":          "reduce_assurance",
		"opt_in_kind":                    "exact_action_approval",
		"reduced_assurance_acknowledged": true,
		"approval_binding_posture":       "exact_action",
	}
}

func backendPostureApprovalID(requestCanonical []byte) string {
	requestHash := sha256.Sum256(requestCanonical)
	return "sha256:" + hex.EncodeToString(requestHash[:])
}

func backendPostureVerifierRecord(publicKey ed25519.PublicKey, keyIDValue string) trustpolicy.VerifierRecord {
	return trustpolicy.VerifierRecord{SchemaID: trustpolicy.VerifierSchemaID, SchemaVersion: trustpolicy.VerifierSchemaVersion, KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Alg: "ed25519", PublicKey: trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)}, LogicalPurpose: "approval_authority", LogicalScope: "user", OwnerPrincipal: trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "user", PrincipalID: "human", InstanceID: "approval-session"}, KeyProtectionPosture: "hardware_backed", IdentityBindingPosture: "attested", PresenceMode: "hardware_touch", CreatedAt: "2026-03-13T12:00:00Z", Status: "active"}
}

func backendPostureRequestEnvelope(requestBytes []byte, keyIDValue string, sig []byte) trustpolicy.SignedObjectEnvelope {
	return trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.ApprovalRequestSchemaID, PayloadSchemaVersion: trustpolicy.ApprovalRequestSchemaVersion, Payload: requestBytes, SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: keyIDValue, Signature: base64.StdEncoding.EncodeToString(sig)}}
}

func backendPostureApprovalDecisionEnvelope(approvalID string, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey) (trustpolicy.SignedObjectEnvelope, error) {
	decisionBytes, err := marshalBackendPostureDecisionPayload(approvalID, verifier)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	decisionCanonical, err := jsoncanonicalizer.Transform(decisionBytes)
	if err != nil {
		return trustpolicy.SignedObjectEnvelope{}, err
	}
	decisionSig := ed25519.Sign(privateKey, decisionCanonical)
	return trustpolicy.SignedObjectEnvelope{SchemaID: trustpolicy.EnvelopeSchemaID, SchemaVersion: trustpolicy.EnvelopeSchemaVersion, PayloadSchemaID: trustpolicy.ApprovalDecisionSchemaID, PayloadSchemaVersion: trustpolicy.ApprovalDecisionSchemaVersion, Payload: decisionBytes, SignatureInput: trustpolicy.SignatureInputProfile, Signature: trustpolicy.SignatureBlock{Alg: "ed25519", KeyID: trustpolicy.KeyIDProfile, KeyIDValue: verifier.KeyIDValue, Signature: base64.StdEncoding.EncodeToString(decisionSig)}}, nil
}

func marshalBackendPostureDecisionPayload(approvalID string, verifier trustpolicy.VerifierRecord) ([]byte, error) {
	decisionPayload := map[string]any{"schema_id": trustpolicy.ApprovalDecisionSchemaID, "schema_version": trustpolicy.ApprovalDecisionSchemaVersion, "approval_request_hash": map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(approvalID, "sha256:")}, "approver": map[string]any{"schema_id": "runecode.protocol.v0.PrincipalIdentity", "schema_version": "0.2.0", "actor_kind": "user", "principal_id": "human", "instance_id": "approval-session"}, "decision_outcome": "approve", "approval_assurance_level": "reauthenticated", "presence_mode": "hardware_touch", "key_protection_posture": "hardware_backed", "identity_binding_posture": "attested", "approval_assertion_hash": map[string]any{"hash_alg": "sha256", "hash": strings.Repeat("f", 64)}, "decided_at": time.Now().UTC().Format(time.RFC3339), "consumption_posture": "single_use", "signatures": []any{map[string]any{"alg": "ed25519", "key_id": trustpolicy.KeyIDProfile, "key_id_value": verifier.KeyIDValue, "signature": "c2ln"}}}
	return json.Marshal(decisionPayload)
}

func persistBackendPostureApprovalFixture(service *brokerapi.Service, targetInstanceID, actionHash, approvalID string, requestEnv trustpolicy.SignedObjectEnvelope) (string, error) {
	policyDecision := policyengine.PolicyDecision{SchemaID: "runecode.protocol.v0.PolicyDecision", SchemaVersion: "0.3.0", DecisionOutcome: policyengine.DecisionRequireHumanApproval, PolicyReasonCode: "approval_required", ManifestHash: "sha256:" + strings.Repeat("1", 64), ActionRequestHash: actionHash, PolicyInputHashes: []string{"sha256:" + strings.Repeat("4", 64)}, DetailsSchemaID: "runecode.protocol.details.policy.evaluation.v0", Details: map[string]any{"precedence": "approval_profile_moderate"}, RequiredApprovalSchemaID: "runecode.protocol.details.policy.required_approval.reduced_assurance_backend.v0", RequiredApproval: map[string]any{"approval_trigger_code": "reduced_assurance_backend", "approval_assurance_level": "reauthenticated", "presence_mode": "hardware_touch", "scope": map[string]any{"schema_id": "runecode.protocol.v0.ApprovalBoundScope", "schema_version": "0.1.0", "workspace_id": "workspace-local", "run_id": "run-backend", "instance_id": targetInstanceID, "action_kind": policyengine.ActionKindBackendPosture}, "changes_if_approved": "Reduced-assurance backend posture change may be applied.", "approval_ttl_seconds": 1800}}
	if err := service.RecordPolicyDecision("run-backend", "", policyDecision); err != nil {
		return "", err
	}
	refs := service.PolicyDecisionRefsForRun("run-backend")
	if len(refs) == 0 {
		return "", fmt.Errorf("missing policy decision refs")
	}
	policyHash := refs[len(refs)-1]
	if err := recordPendingApproval(service, targetInstanceID, actionHash, approvalID, policyHash, requestEnv); err != nil {
		return "", err
	}
	return policyHash, nil
}

func recordPendingApproval(service *brokerapi.Service, targetInstanceID, actionHash, approvalID, policyHash string, requestEnv trustpolicy.SignedObjectEnvelope) error {
	expiresAt := time.Now().UTC().Add(30 * time.Minute)
	requestedAt := time.Now().UTC().Add(-time.Minute)
	record := artifacts.ApprovalRecord{ApprovalID: approvalID, Status: "pending", WorkspaceID: "workspace-local", InstanceID: targetInstanceID, RunID: "run-backend", ActionKind: policyengine.ActionKindBackendPosture, RequestedAt: requestedAt, ExpiresAt: &expiresAt, ApprovalTriggerCode: "reduced_assurance_backend", ChangesIfApproved: "Reduced-assurance backend posture change may be applied.", ApprovalAssuranceLevel: "reauthenticated", PresenceMode: "hardware_touch", ManifestHash: "sha256:" + strings.Repeat("1", 64), ActionRequestHash: actionHash, PolicyDecisionHash: policyHash, RequestDigest: approvalID, RequestEnvelope: &requestEnv}
	return service.RecordApproval(record)
}

func putTrustedVerifierRecordForService(service *brokerapi.Service, record trustpolicy.VerifierRecord) error {
	b, err := json.Marshal(record)
	if err != nil {
		return err
	}
	provenance := "sha256:" + strings.Repeat("1", 64)
	ref, err := service.Put(artifacts.PutRequest{Payload: b, ContentType: "application/json", DataClass: artifacts.DataClassAuditVerificationReport, ProvenanceReceiptHash: provenance, CreatedByRole: "auditd", TrustedSource: true})
	if err != nil {
		return err
	}
	details := map[string]interface{}{artifacts.TrustedContractImportKindDetailKey: artifacts.TrustedContractImportKindVerifierRecord, artifacts.TrustedContractImportArtifactDigestDetailKey: ref.Digest, artifacts.TrustedContractImportProvenanceDetailKey: provenance}
	return service.AppendTrustedAuditEvent(artifacts.TrustedContractImportAuditEventType, "brokerapi", details)
}

func serviceCurrentInstanceID(service *brokerapi.Service) string {
	resp, errResp := service.HandleBackendPostureGet(context.Background(), brokerapi.BackendPostureGetRequest{SchemaID: "runecode.protocol.v0.BackendPostureGetRequest", SchemaVersion: "0.1.0", RequestID: "perf-instance-id"}, brokerapi.RequestContext{})
	if errResp != nil {
		return "launcher-instance-1"
	}
	if strings.TrimSpace(resp.Posture.InstanceID) == "" {
		return "launcher-instance-1"
	}
	return strings.TrimSpace(resp.Posture.InstanceID)
}

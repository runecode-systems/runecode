//go:build runecode_devseed

package brokerapi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func devManualVerifierRecord() trustpolicy.VerifierRecord {
	publicKey := make([]byte, 32)
	keyID := sha256.Sum256(publicKey)
	return trustpolicy.VerifierRecord{
		SchemaID:               trustpolicy.VerifierSchemaID,
		SchemaVersion:          trustpolicy.VerifierSchemaVersion,
		KeyID:                  trustpolicy.KeyIDProfile,
		KeyIDValue:             hex.EncodeToString(keyID[:]),
		Alg:                    "ed25519",
		PublicKey:              trustpolicy.PublicKey{Encoding: "base64", Value: base64.StdEncoding.EncodeToString(publicKey)},
		LogicalPurpose:         "isolate_session_identity",
		LogicalScope:           "session",
		OwnerPrincipal:         trustpolicy.PrincipalIdentity{SchemaID: "runecode.protocol.v0.PrincipalIdentity", SchemaVersion: "0.2.0", ActorKind: "daemon", PrincipalID: "auditd", InstanceID: "auditd-1"},
		KeyProtectionPosture:   "os_keystore",
		IdentityBindingPosture: "attested",
		PresenceMode:           "os_confirmation",
		CreatedAt:              "2026-03-13T12:00:00Z",
		Status:                 "active",
	}
}

func devManualEventContractCatalog() trustpolicy.AuditEventContractCatalog {
	return trustpolicy.AuditEventContractCatalog{SchemaID: trustpolicy.AuditEventContractCatalogSchemaID, SchemaVersion: trustpolicy.AuditEventContractCatalogSchemaVersion, CatalogID: "audit_event_contract_v0", Entries: []trustpolicy.AuditEventContractCatalogEntry{{AuditEventType: "isolate_session_bound", AllowedPayloadSchemaIDs: []string{trustpolicy.IsolateSessionBoundPayloadSchemaID}, AllowedSignerPurposes: []string{"isolate_session_identity"}, AllowedSignerScopes: []string{"session"}, RequiredScopeFields: []string{"workspace_id", "run_id", "stage_id"}, RequiredCorrelationFields: []string{"session_id", "operation_id"}, RequireSubjectRef: true, AllowedSubjectRefRoles: []string{"binding_target"}, AllowedCauseRefRoles: []string{"session_cause"}, AllowedRelatedRefRoles: []string{"binding", "evidence", "receipt"}, RequireSignerEvidenceRefs: true, AllowedSignerEvidenceRefRoles: []string{"admissibility", "binding"}}}}
}

func devManualVerificationReport() trustpolicy.AuditVerificationReportPayload {
	record := devManualVerifierRecord()
	return trustpolicy.AuditVerificationReportPayload{SchemaID: trustpolicy.AuditVerificationReportSchemaID, SchemaVersion: trustpolicy.AuditVerificationReportSchemaVersion, VerifiedAt: "2026-03-13T12:30:00Z", VerificationScope: trustpolicy.AuditVerificationScope{ScopeKind: trustpolicy.AuditVerificationScopeSegment, LastSegmentID: "segment-000001"}, CryptographicallyValid: false, HistoricallyAdmissible: false, CurrentlyDegraded: true, IntegrityStatus: trustpolicy.AuditVerificationStatusFailed, AnchoringStatus: trustpolicy.AuditVerificationStatusDegraded, AnchoringPosture: trustpolicy.AuditVerificationAnchoringPostureAnchorReceiptMissingOrUnbound, StoragePostureStatus: trustpolicy.AuditVerificationStatusOK, SegmentLifecycleStatus: trustpolicy.AuditVerificationStatusOK, VerifierIdentity: trustpolicy.KeyIDProfile + ":" + record.KeyIDValue, TrustRootIdentities: []string{"sha256:" + record.KeyIDValue}, DegradedReasons: []string{trustpolicy.AuditVerificationReasonAnchorReceiptMissing}, HardFailures: []string{trustpolicy.AuditVerificationReasonDetachedSignatureInvalid}, Findings: []trustpolicy.AuditVerificationFinding{{Code: trustpolicy.AuditVerificationReasonDetachedSignatureInvalid, Dimension: trustpolicy.AuditVerificationDimensionIntegrity, Severity: trustpolicy.AuditVerificationSeverityError, Message: "dev manual seed uses synthetic envelopes and does not represent verified production audit state", SegmentID: "segment-000001"}, {Code: trustpolicy.AuditVerificationReasonAnchorReceiptMissing, Dimension: trustpolicy.AuditVerificationDimensionAnchoring, Severity: trustpolicy.AuditVerificationSeverityWarning, Message: "dev manual seed does not include external anchor receipts", SegmentID: "segment-000001"}}, Summary: "synthetic dev seed; not a verified production audit posture"}
}

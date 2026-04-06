package trustpolicy

import (
	"fmt"
)

type AuditAdmissionChecks struct {
	SchemaValidation         bool `json:"schema_validation"`
	EventContractValidation  bool `json:"event_contract_validation"`
	SignerEvidenceValidation bool `json:"signer_evidence_validation"`
	DetachedSignatureVerify  bool `json:"detached_signature_verification"`
}

func (c AuditAdmissionChecks) Validate() error {
	if !c.SchemaValidation {
		return fmt.Errorf("schema_validation must be enabled")
	}
	if !c.EventContractValidation {
		return fmt.Errorf("event_contract_validation must be enabled")
	}
	if !c.SignerEvidenceValidation {
		return fmt.Errorf("signer_evidence_validation must be enabled")
	}
	if !c.DetachedSignatureVerify {
		return fmt.Errorf("detached_signature_verification must be enabled")
	}
	return nil
}

type AuditAdmissionRequest struct {
	Checks               AuditAdmissionChecks           `json:"checks"`
	Envelope             SignedObjectEnvelope           `json:"envelope"`
	VerifierRecords      []VerifierRecord               `json:"verifier_records"`
	EventContractCatalog AuditEventContractCatalog      `json:"event_contract_catalog"`
	SignerEvidence       []AuditSignerEvidenceReference `json:"signer_evidence"`
}

func ValidateAuditAdmissionRequest(req AuditAdmissionRequest) error {
	if err := req.Checks.Validate(); err != nil {
		return err
	}
	if err := validateAuditEventContractCatalog(req.EventContractCatalog); err != nil {
		return err
	}
	registry, err := NewVerifierRegistry(req.VerifierRecords)
	if err != nil {
		return err
	}
	if err := VerifySignedEnvelope(req.Envelope, registry, EnvelopeVerificationOptions{
		RequirePayloadSchemaMatch: true,
		ExpectedPayloadSchemaID:   AuditEventSchemaID,
		ExpectedPayloadVersion:    AuditEventSchemaVersion,
	}); err != nil {
		return err
	}
	event, err := decodeAuditEventPayload(req.Envelope.Payload)
	if err != nil {
		return err
	}
	if err := validateAuditEventPayloadShape(event); err != nil {
		return err
	}
	if err := validateAuditEventPayloadHash(event); err != nil {
		return err
	}
	entry, err := validateAuditEventAgainstCatalog(event, req.EventContractCatalog)
	if err != nil {
		return err
	}
	if err := validateSignerEvidenceRefs(event, req.Envelope.Signature, entry, req.SignerEvidence); err != nil {
		return err
	}
	return nil
}

func validateAuditEventContractCatalog(catalog AuditEventContractCatalog) error {
	if err := validateAuditEventContractCatalogHeader(catalog); err != nil {
		return err
	}
	return validateAuditEventContractCatalogEntries(catalog.Entries)
}

func validateAuditEventContractCatalogHeader(catalog AuditEventContractCatalog) error {
	if catalog.SchemaID != AuditEventContractCatalogSchemaID {
		return fmt.Errorf("unexpected event contract catalog schema_id %q", catalog.SchemaID)
	}
	if catalog.SchemaVersion != AuditEventContractCatalogSchemaVersion {
		return fmt.Errorf("unexpected event contract catalog schema_version %q", catalog.SchemaVersion)
	}
	if !sealReasonPattern.MatchString(catalog.CatalogID) {
		return fmt.Errorf("catalog_id must match %s", sealReasonPattern.String())
	}
	if len(catalog.Entries) == 0 {
		return fmt.Errorf("event contract catalog requires entries")
	}
	return nil
}

func validateAuditEventContractCatalogEntries(entries []AuditEventContractCatalogEntry) error {
	seen := map[string]struct{}{}
	for index := range entries {
		if err := validateAuditEventContractCatalogEntry(entries[index], index, seen); err != nil {
			return err
		}
	}
	return nil
}

func validateAuditEventContractCatalogEntry(entry AuditEventContractCatalogEntry, index int, seen map[string]struct{}) error {
	if entry.AuditEventType == "" {
		return fmt.Errorf("entries[%d].audit_event_type is required", index)
	}
	if _, exists := seen[entry.AuditEventType]; exists {
		return fmt.Errorf("duplicate event-contract entry for audit_event_type %q", entry.AuditEventType)
	}
	seen[entry.AuditEventType] = struct{}{}

	if err := requireCatalogEntryMandatoryLists(entry, index); err != nil {
		return err
	}
	return requireCatalogEntryConditionalLists(entry, index)
}

func requireCatalogEntryMandatoryLists(entry AuditEventContractCatalogEntry, index int) error {
	if len(entry.AllowedPayloadSchemaIDs) == 0 {
		return fmt.Errorf("entries[%d].allowed_payload_schema_ids is required", index)
	}
	if len(entry.AllowedSignerPurposes) == 0 {
		return fmt.Errorf("entries[%d].allowed_signer_purposes is required", index)
	}
	if len(entry.AllowedSignerScopes) == 0 {
		return fmt.Errorf("entries[%d].allowed_signer_scopes is required", index)
	}
	return nil
}

func requireCatalogEntryConditionalLists(entry AuditEventContractCatalogEntry, index int) error {
	if entry.RequireGatewayContext && len(entry.AllowedGatewayEgressCategories) == 0 {
		return fmt.Errorf("entries[%d] requires gateway categories when require_gateway_context=true", index)
	}
	if entry.RequireSignerEvidenceRefs && len(entry.AllowedSignerEvidenceRefRoles) == 0 {
		return fmt.Errorf("entries[%d] requires signer evidence ref roles when require_signer_evidence_refs=true", index)
	}
	if entry.RequireSubjectRef && len(entry.AllowedSubjectRefRoles) == 0 {
		return fmt.Errorf("entries[%d] requires subject_ref roles when require_subject_ref=true", index)
	}
	return nil
}

package trustpolicy

import "fmt"

func validateAuditEventAgainstCatalog(event AuditEventPayload, catalog AuditEventContractCatalog) (AuditEventContractCatalogEntry, error) {
	entry, err := catalogEntryForEventType(catalog, event.AuditEventType)
	if err != nil {
		return AuditEventContractCatalogEntry{}, err
	}
	if err := validateCatalogFieldRequirements(event, entry); err != nil {
		return AuditEventContractCatalogEntry{}, err
	}
	if err := validateCatalogSubjectRef(event, entry); err != nil {
		return AuditEventContractCatalogEntry{}, err
	}
	if err := validateCatalogReferenceRoles(event, entry); err != nil {
		return AuditEventContractCatalogEntry{}, err
	}
	if err := validateCatalogGatewayContext(event, entry); err != nil {
		return AuditEventContractCatalogEntry{}, err
	}
	if err := validateCatalogSignerEvidencePresence(event, entry); err != nil {
		return AuditEventContractCatalogEntry{}, err
	}
	if err := validateReferenceRoles(event.SignerEvidenceRefs, entry.AllowedSignerEvidenceRefRoles, "signer_evidence_refs"); err != nil {
		return AuditEventContractCatalogEntry{}, err
	}
	return entry, nil
}

func validateCatalogFieldRequirements(event AuditEventPayload, entry AuditEventContractCatalogEntry) error {
	if !containsString(entry.AllowedPayloadSchemaIDs, event.EventPayloadSchemaID) {
		return fmt.Errorf("event payload schema %q is not allowed for audit_event_type %q", event.EventPayloadSchemaID, event.AuditEventType)
	}
	if err := requireStringMapFields(event.Scope, entry.RequiredScopeFields, "scope"); err != nil {
		return err
	}
	return requireStringMapFields(event.Correlation, entry.RequiredCorrelationFields, "correlation")
}

func validateCatalogReferenceRoles(event AuditEventPayload, entry AuditEventContractCatalogEntry) error {
	if err := validateReferenceRoles(event.CauseRefs, entry.AllowedCauseRefRoles, "cause_refs"); err != nil {
		return err
	}
	return validateReferenceRoles(event.RelatedRefs, entry.AllowedRelatedRefRoles, "related_refs")
}

func validateCatalogSignerEvidencePresence(event AuditEventPayload, entry AuditEventContractCatalogEntry) error {
	if entry.RequireSignerEvidenceRefs && len(event.SignerEvidenceRefs) == 0 {
		return fmt.Errorf("signer_evidence_refs are required for audit_event_type %q", event.AuditEventType)
	}
	return nil
}

func validateCatalogSubjectRef(event AuditEventPayload, entry AuditEventContractCatalogEntry) error {
	if event.SubjectRef == nil {
		if entry.RequireSubjectRef {
			return fmt.Errorf("subject_ref is required for audit_event_type %q", event.AuditEventType)
		}
		return nil
	}
	if !containsString(entry.AllowedSubjectRefRoles, event.SubjectRef.RefRole) {
		return fmt.Errorf("subject_ref.ref_role %q is not allowed for audit_event_type %q", event.SubjectRef.RefRole, event.AuditEventType)
	}
	return nil
}

func validateCatalogGatewayContext(event AuditEventPayload, entry AuditEventContractCatalogEntry) error {
	if event.GatewayContext == nil {
		if entry.RequireGatewayContext {
			return fmt.Errorf("gateway_context is required for audit_event_type %q", event.AuditEventType)
		}
		return nil
	}
	if !containsString(entry.AllowedGatewayEgressCategories, event.GatewayContext.EgressCategory) {
		return fmt.Errorf("gateway egress category %q is not allowed for audit_event_type %q", event.GatewayContext.EgressCategory, event.AuditEventType)
	}
	return nil
}

func catalogEntryForEventType(catalog AuditEventContractCatalog, auditEventType string) (AuditEventContractCatalogEntry, error) {
	var matched *AuditEventContractCatalogEntry
	for index := range catalog.Entries {
		entry := &catalog.Entries[index]
		if entry.AuditEventType != auditEventType {
			continue
		}
		if matched != nil {
			return AuditEventContractCatalogEntry{}, fmt.Errorf("duplicate event-contract entry for audit_event_type %q", auditEventType)
		}
		matched = entry
	}
	if matched == nil {
		return AuditEventContractCatalogEntry{}, fmt.Errorf("no event-contract entry for audit_event_type %q", auditEventType)
	}
	return *matched, nil
}

func requireStringMapFields(values map[string]string, required []string, blockName string) error {
	for _, key := range required {
		if values == nil || values[key] == "" {
			return fmt.Errorf("%s.%s is required by event contract", blockName, key)
		}
	}
	return nil
}

func validateReferenceRoles(refs []AuditTypedReference, allowed []string, field string) error {
	for index := range refs {
		ref := refs[index]
		if _, err := ref.Digest.Identity(); err != nil {
			return fmt.Errorf("%s[%d].digest: %w", field, index, err)
		}
		if !containsString(allowed, ref.RefRole) {
			return fmt.Errorf("%s[%d].ref_role %q is not allowed", field, index, ref.RefRole)
		}
	}
	return nil
}

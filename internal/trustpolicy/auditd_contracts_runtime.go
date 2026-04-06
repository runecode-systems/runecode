package trustpolicy

// ValidateAuditEventContractCatalogForRuntime validates the event contract catalog
// shape and invariants for runtime loading paths.
func ValidateAuditEventContractCatalogForRuntime(catalog AuditEventContractCatalog) error {
	return validateAuditEventContractCatalog(catalog)
}

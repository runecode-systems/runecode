package protocolschema

import "testing"

func assertSchemaVersions(t *testing.T, manifest manifestFile) {
	t.Helper()
	assertSchemaVersionsCore(t, manifest)
	assertSchemaVersionsLocalBroker(t, manifest)
}

func assertSchemaVersionsCore(t *testing.T, manifest manifestFile) {
	t.Helper()
	for schemaID, version := range coreSchemaVersionsPart1() {
		assertManifestSchemaVersion(t, manifest, schemaID, version)
	}
	for schemaID, version := range coreSchemaVersionsPart2() {
		assertManifestSchemaVersion(t, manifest, schemaID, version)
	}
}

func assertSchemaVersionsLocalBroker(t *testing.T, manifest manifestFile) {
	t.Helper()
	for schemaID, version := range localBrokerSchemaVersions() {
		assertManifestSchemaVersion(t, manifest, schemaID, version)
	}
}

func coreSchemaVersionsPart1() map[string]string {
	return map[string]string{
		"runecode.protocol.v0.ArtifactReference":               "0.4.0",
		"runecode.protocol.v0.ArtifactPolicy":                  "0.1.0",
		"runecode.protocol.v0.AuditRecordDigest":               "0.1.0",
		"runecode.protocol.v0.AuditEvent":                      "0.5.0",
		"runecode.protocol.v0.AuditEventContractCatalog":       "0.1.0",
		"runecode.protocol.v0.AuditReceipt":                    "0.5.0",
		"runecode.protocol.v0.AuditSegmentSeal":                "0.2.0",
		"runecode.protocol.v0.AuditSegmentFile":                "0.1.0",
		"runecode.protocol.v0.AuditVerificationReport":         "0.1.0",
		"runecode.protocol.v0.SignedObjectEnvelope":            "0.2.0",
		"runecode.protocol.v0.ApprovalRequest":                 "0.3.0",
		"runecode.protocol.v0.ApprovalDecision":                "0.3.0",
		"runecode.protocol.v0.PolicyAllowlist":                 "0.1.0",
		"runecode.protocol.v0.ActionPayloadSecretAccess":       "0.1.0",
		"runecode.protocol.v0.SecretLease":                     "0.1.0",
		"runecode.protocol.v0.SecretStoragePosture":            "0.1.0",
		"runecode.protocol.v0.DestinationDescriptor":           "0.1.0",
		"runecode.protocol.v0.DependencyFetchRequest":          "0.1.0",
		"runecode.protocol.v0.DependencyFetchBatchRequest":     "0.1.0",
		"runecode.protocol.v0.DependencyResolvedUnitManifest":  "0.1.0",
		"runecode.protocol.v0.DependencyFetchBatchResult":      "0.1.0",
		"runecode.protocol.v0.DependencyCacheEnsureRequest":    "0.1.0",
		"runecode.protocol.v0.DependencyCacheEnsureResponse":   "0.1.0",
		"runecode.protocol.v0.DependencyFetchRegistryRequest":  "0.1.0",
		"runecode.protocol.v0.DependencyFetchRegistryResponse": "0.1.0",
		"runecode.protocol.v0.GatewayScopeRule":                "0.1.0",
	}
}

func coreSchemaVersionsPart2() map[string]string {
	return map[string]string{
		"runecode.protocol.v0.PolicyRuleSet":                "0.1.0",
		"runecode.protocol.v0.VerifierRecord":               "0.1.0",
		"runecode.protocol.v0.BrokerArtifactListRequest":    "0.1.0",
		"runecode.protocol.v0.BrokerArtifactListResponse":   "0.1.0",
		"runecode.protocol.v0.BrokerArtifactHeadRequest":    "0.1.0",
		"runecode.protocol.v0.BrokerArtifactHeadResponse":   "0.1.0",
		"runecode.protocol.v0.BrokerArtifactPutRequest":     "0.1.0",
		"runecode.protocol.v0.BrokerArtifactPutResponse":    "0.1.0",
		"runecode.protocol.v0.BrokerErrorResponse":          "0.1.0",
		"runecode.protocol.v0.RuntimeImageDescriptor":       "0.2.0",
		"runecode.protocol.v0.IsolateSessionStartedPayload": "0.1.0",
		"runecode.protocol.v0.IsolateSessionBoundPayload":   "0.1.0",
	}
}

func localBrokerSchemaVersions() map[string]string {
	return map[string]string{
		"runecode.protocol.v0.RunSummary":                     "0.2.0",
		"runecode.protocol.v0.RunDetail":                      "0.2.0",
		"runecode.protocol.v0.RunStageSummary":                "0.1.0",
		"runecode.protocol.v0.RunRoleSummary":                 "0.1.0",
		"runecode.protocol.v0.RunCoordinationSummary":         "0.1.0",
		"runecode.protocol.v0.RunnerCheckpointReport":         "0.1.0",
		"runecode.protocol.v0.RunnerResultReport":             "0.1.0",
		"runecode.protocol.v0.RunnerCheckpointReportRequest":  "0.1.0",
		"runecode.protocol.v0.RunnerCheckpointReportResponse": "0.1.0",
		"runecode.protocol.v0.RunnerResultReportRequest":      "0.1.0",
		"runecode.protocol.v0.RunnerResultReportResponse":     "0.1.0",
	}
}

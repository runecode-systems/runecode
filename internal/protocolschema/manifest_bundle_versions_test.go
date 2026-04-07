package protocolschema

import "testing"

func assertSchemaVersionsLocalBroker(t *testing.T, manifest manifestFile) {
	t.Helper()
	assertSchemaVersionPairs(t, manifest, localBrokerSchemaVersionPairs())
}

func assertSchemaVersionPairs(t *testing.T, manifest manifestFile, versions map[string]string) {
	t.Helper()
	for schemaID, version := range versions {
		assertManifestSchemaVersion(t, manifest, schemaID, version)
	}
}

func localBrokerSchemaVersionPairs() map[string]string {
	return map[string]string{
		"runecode.protocol.v0.RunSummary":                   "0.1.0",
		"runecode.protocol.v0.RunDetail":                    "0.1.0",
		"runecode.protocol.v0.RunStageSummary":              "0.1.0",
		"runecode.protocol.v0.RunRoleSummary":               "0.1.0",
		"runecode.protocol.v0.RunCoordinationSummary":       "0.1.0",
		"runecode.protocol.v0.ApprovalSummary":              "0.1.0",
		"runecode.protocol.v0.ApprovalBoundScope":           "0.1.0",
		"runecode.protocol.v0.ArtifactSummary":              "0.1.0",
		"runecode.protocol.v0.BrokerReadiness":              "0.1.0",
		"runecode.protocol.v0.BrokerVersionInfo":            "0.1.0",
		"runecode.protocol.v0.RunListRequest":               "0.1.0",
		"runecode.protocol.v0.RunListResponse":              "0.1.0",
		"runecode.protocol.v0.RunGetRequest":                "0.1.0",
		"runecode.protocol.v0.RunGetResponse":               "0.1.0",
		"runecode.protocol.v0.ApprovalListRequest":          "0.1.0",
		"runecode.protocol.v0.ApprovalListResponse":         "0.1.0",
		"runecode.protocol.v0.ApprovalGetRequest":           "0.1.0",
		"runecode.protocol.v0.ApprovalGetResponse":          "0.1.0",
		"runecode.protocol.v0.ApprovalResolveRequest":       "0.1.0",
		"runecode.protocol.v0.ApprovalResolveResponse":      "0.1.0",
		"runecode.protocol.v0.ArtifactListRequest":          "0.1.0",
		"runecode.protocol.v0.ArtifactListResponse":         "0.1.0",
		"runecode.protocol.v0.ArtifactHeadRequest":          "0.1.0",
		"runecode.protocol.v0.ArtifactHeadResponse":         "0.1.0",
		"runecode.protocol.v0.ArtifactReadRequest":          "0.1.0",
		"runecode.protocol.v0.ArtifactStreamEvent":          "0.1.0",
		"runecode.protocol.v0.AuditTimelineRequest":         "0.1.0",
		"runecode.protocol.v0.AuditTimelineResponse":        "0.1.0",
		"runecode.protocol.v0.AuditVerificationGetRequest":  "0.1.0",
		"runecode.protocol.v0.AuditVerificationGetResponse": "0.1.0",
		"runecode.protocol.v0.LogStreamRequest":             "0.1.0",
		"runecode.protocol.v0.LogStreamEvent":               "0.1.0",
		"runecode.protocol.v0.ReadinessGetRequest":          "0.1.0",
		"runecode.protocol.v0.ReadinessGetResponse":         "0.1.0",
		"runecode.protocol.v0.VersionInfoGetRequest":        "0.1.0",
		"runecode.protocol.v0.VersionInfoGetResponse":       "0.1.0",
	}
}

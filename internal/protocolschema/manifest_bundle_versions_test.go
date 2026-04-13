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
	return mergeSchemaVersionPairs(
		localBrokerGateAndRunSchemaVersionPairs(),
		localBrokerRunnerSchemaVersionPairs(),
		localBrokerApprovalAndArtifactSchemaVersionPairs(),
		localBrokerReadApiSchemaVersionPairs(),
		localBrokerDefinitionSchemaVersionPairs(),
	)
}

func mergeSchemaVersionPairs(pairSets ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, pairSet := range pairSets {
		for schemaID, version := range pairSet {
			out[schemaID] = version
		}
	}
	return out
}

func localBrokerGateAndRunSchemaVersionPairs() map[string]string {
	return map[string]string{
		"runecode.protocol.v0.GateContract":           "0.1.0",
		"runecode.protocol.v0.GateDefinition":         "0.1.0",
		"runecode.protocol.v0.GateCheckpointReport":   "0.1.0",
		"runecode.protocol.v0.GateResultReport":       "0.1.0",
		"runecode.protocol.v0.GateEvidence":           "0.1.0",
		"runecode.protocol.v0.RunSummary":             "0.2.0",
		"runecode.protocol.v0.RunDetail":              "0.2.0",
		"runecode.protocol.v0.RunStageSummary":        "0.1.0",
		"runecode.protocol.v0.RunRoleSummary":         "0.1.0",
		"runecode.protocol.v0.RunCoordinationSummary": "0.1.0",
	}
}

func localBrokerRunnerSchemaVersionPairs() map[string]string {
	return map[string]string{
		"runecode.protocol.v0.RunnerCheckpointReport":         "0.1.0",
		"runecode.protocol.v0.RunnerResultReport":             "0.1.0",
		"runecode.protocol.v0.RunnerCheckpointReportRequest":  "0.1.0",
		"runecode.protocol.v0.RunnerCheckpointReportResponse": "0.1.0",
		"runecode.protocol.v0.RunnerResultReportRequest":      "0.1.0",
		"runecode.protocol.v0.RunnerResultReportResponse":     "0.1.0",
	}
}

func localBrokerApprovalAndArtifactSchemaVersionPairs() map[string]string {
	return map[string]string{
		"runecode.protocol.v0.ApprovalSummary":               "0.1.0",
		"runecode.protocol.v0.ApprovalDetail":                "0.1.0",
		"runecode.protocol.v0.ApprovalLifecycleDetail":       "0.1.0",
		"runecode.protocol.v0.ApprovalWhatChangesIfApproved": "0.1.0",
		"runecode.protocol.v0.ApprovalBlockedWorkScope":      "0.1.0",
		"runecode.protocol.v0.ApprovalBoundIdentity":         "0.1.0",
		"runecode.protocol.v0.ApprovalBoundScope":            "0.1.0",
		"runecode.protocol.v0.ArtifactSummary":               "0.1.0",
		"runecode.protocol.v0.BrokerReadiness":               "0.1.0",
		"runecode.protocol.v0.SecretLease":                   "0.1.0",
		"runecode.protocol.v0.SecretStoragePosture":          "0.1.0",
		"runecode.protocol.v0.BrokerVersionInfo":             "0.1.0",
		"runecode.protocol.v0.ApprovalResolveRequest":        "0.1.0",
		"runecode.protocol.v0.ApprovalResolveResponse":       "0.1.0",
		"runecode.protocol.v0.ArtifactReadRequest":           "0.1.0",
		"runecode.protocol.v0.ArtifactStreamEvent":           "0.1.0",
	}
}

func localBrokerReadApiSchemaVersionPairs() map[string]string {
	return mergeSchemaVersionPairs(
		localBrokerReadApiRunAndSessionSchemaVersionPairs(),
		localBrokerReadApiApprovalAndArtifactSchemaVersionPairs(),
		localBrokerReadApiAuditAndStatusSchemaVersionPairs(),
	)
}

func localBrokerReadApiRunAndSessionSchemaVersionPairs() map[string]string {
	return map[string]string{
		"runecode.protocol.v0.RunListRequest":             "0.1.0",
		"runecode.protocol.v0.RunListResponse":            "0.1.0",
		"runecode.protocol.v0.RunGetRequest":              "0.1.0",
		"runecode.protocol.v0.RunGetResponse":             "0.1.0",
		"runecode.protocol.v0.SessionIdentity":            "0.1.0",
		"runecode.protocol.v0.SessionSummary":             "0.1.0",
		"runecode.protocol.v0.SessionDetail":              "0.1.0",
		"runecode.protocol.v0.SessionTranscriptLinks":     "0.1.0",
		"runecode.protocol.v0.SessionTranscriptMessage":   "0.1.0",
		"runecode.protocol.v0.SessionTranscriptTurn":      "0.1.0",
		"runecode.protocol.v0.SessionListRequest":         "0.1.0",
		"runecode.protocol.v0.SessionListResponse":        "0.1.0",
		"runecode.protocol.v0.SessionGetRequest":          "0.1.0",
		"runecode.protocol.v0.SessionGetResponse":         "0.1.0",
		"runecode.protocol.v0.SessionSendMessageRequest":  "0.1.0",
		"runecode.protocol.v0.SessionSendMessageResponse": "0.1.0",
	}
}

func localBrokerReadApiApprovalAndArtifactSchemaVersionPairs() map[string]string {
	return map[string]string{
		"runecode.protocol.v0.ApprovalListRequest":  "0.1.0",
		"runecode.protocol.v0.ApprovalListResponse": "0.1.0",
		"runecode.protocol.v0.ApprovalGetRequest":   "0.1.0",
		"runecode.protocol.v0.ApprovalGetResponse":  "0.1.0",
		"runecode.protocol.v0.ArtifactListRequest":  "0.1.0",
		"runecode.protocol.v0.ArtifactListResponse": "0.1.0",
		"runecode.protocol.v0.ArtifactHeadRequest":  "0.1.0",
		"runecode.protocol.v0.ArtifactHeadResponse": "0.1.0",
		"runecode.protocol.v0.LogStreamRequest":     "0.1.0",
		"runecode.protocol.v0.LogStreamEvent":       "0.1.0",
	}
}

func localBrokerReadApiAuditAndStatusSchemaVersionPairs() map[string]string {
	return map[string]string{
		"runecode.protocol.v0.AuditRecordDetail":            "0.1.0",
		"runecode.protocol.v0.AuditRecordGetRequest":        "0.1.0",
		"runecode.protocol.v0.AuditRecordGetResponse":       "0.1.0",
		"runecode.protocol.v0.AuditTimelineRequest":         "0.1.0",
		"runecode.protocol.v0.AuditTimelineResponse":        "0.1.0",
		"runecode.protocol.v0.AuditVerificationGetRequest":  "0.1.0",
		"runecode.protocol.v0.AuditVerificationGetResponse": "0.1.0",
		"runecode.protocol.v0.ReadinessGetRequest":          "0.1.0",
		"runecode.protocol.v0.ReadinessGetResponse":         "0.1.0",
		"runecode.protocol.v0.VersionInfoGetRequest":        "0.1.0",
		"runecode.protocol.v0.VersionInfoGetResponse":       "0.1.0",
	}
}

func localBrokerDefinitionSchemaVersionPairs() map[string]string {
	return map[string]string{
		"runecode.protocol.v0.WorkflowDefinition": "0.2.0",
		"runecode.protocol.v0.ProcessDefinition":  "0.2.0",
		"runecode.protocol.v0.RunPlan":            "0.1.0",
	}
}

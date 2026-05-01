package artifacts

func DefaultPolicy() Policy {
	return Policy{
		HandOffReferenceMode:                "hash_only",
		ReservedClassesEnabled:              false,
		DependencyCachePolicy:               defaultDependencyCachePolicy(),
		EncryptedAtRestDefault:              true,
		DevPlaintextOverride:                false,
		ExplicitHumanApprovalRequired:       true,
		PromotionMintsNewArtifactReference:  true,
		MaxPromotionRequestBytes:            1024 * 1024,
		MaxPromotionRequestsPerMinute:       30,
		BulkPromotionRequiresSeparateReview: true,
		FlowMatrix: []FlowRule{
			{ProducerRole: "workspace", ConsumerRole: "model_gateway", AllowedDataClasses: []DataClass{DataClassSpecText, DataClassApprovedFileExcerpts}},
			{ProducerRole: "workspace", ConsumerRole: "auditd", AllowedDataClasses: []DataClass{DataClassAuditEvents, DataClassAuditVerificationReport, DataClassGateEvidence, DataClassBuildLogs, DataClassDiffs, DataClassSpecText, DataClassUnapprovedFileExcerpts, DataClassApprovedFileExcerpts}},
			{ProducerRole: "dependency-fetch", ConsumerRole: "workspace", AllowedDataClasses: []DataClass{DataClassDependencyBatchManifest, DataClassDependencyResolvedUnit, DataClassDependencyPayloadUnit, DataClassDependencyMaterialized}},
			{ProducerRole: "dependency-fetch", ConsumerRole: "workspace-edit", AllowedDataClasses: []DataClass{DataClassDependencyBatchManifest, DataClassDependencyResolvedUnit, DataClassDependencyPayloadUnit, DataClassDependencyMaterialized}},
			{ProducerRole: "dependency-fetch", ConsumerRole: "workspace-test", AllowedDataClasses: []DataClass{DataClassDependencyBatchManifest, DataClassDependencyResolvedUnit, DataClassDependencyPayloadUnit, DataClassDependencyMaterialized}},
		},
		RevokedApprovedExcerptHashes: map[string]bool{},
		PerRoleQuota: map[string]Quota{
			"workspace":     {MaxArtifactCount: 4096, MaxTotalBytes: 512 * 1024 * 1024, MaxSingleArtifactSize: 64 * 1024 * 1024},
			"model_gateway": {MaxArtifactCount: 4096, MaxTotalBytes: 512 * 1024 * 1024, MaxSingleArtifactSize: 64 * 1024 * 1024},
		},
		PerStepQuota:                   map[string]Quota{},
		UnreferencedTTLSeconds:         7 * 24 * 3600,
		DeleteOnQuotaPressure:          true,
		RequireOriginMetadata:          []string{"repo_path", "commit", "extractor_tool_version"},
		RequireFullContentVisibility:   true,
		ApprovedExcerptEgressOptInOnly: true,
		UnapprovedExcerptEgressDenied:  true,
	}
}

func defaultDependencyCachePolicy() DependencyCachePolicy {
	return DependencyCachePolicy{
		ReadOnlyArtifactsRequired:            true,
		BatchManifestImmutable:               true,
		ResolvedUnitManifestImmutable:        true,
		ResolvedPayloadImmutable:             true,
		MaterializedTreesDerivedNonCanonical: true,
		FailClosedOnAmbiguousPartialReuse:    true,
		FailClosedOnIncompleteState:          true,
		RetainCanonicalBeforeDerived:         true,
	}
}

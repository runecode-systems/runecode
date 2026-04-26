package protocolschema

func dependencyCacheHandoffRequestCases() []validationCase {
	return []validationCase{
		{name: "valid dependency cache handoff request", value: validDependencyCacheHandoffRequest()},
		{name: "dependency cache handoff request requires consumer role", value: invalidDependencyCacheHandoffRequestWithoutConsumerRole(), wantErr: true},
	}
}

func dependencyCacheHandoffMetadataCases() []validationCase {
	return []validationCase{
		{name: "valid dependency cache handoff metadata", value: validDependencyCacheHandoffMetadata()},
		{name: "dependency cache handoff metadata enforces handoff mode enum", value: invalidDependencyCacheHandoffMetadataWithUnsupportedMode(), wantErr: true},
	}
}

func dependencyCacheHandoffResponseCases() []validationCase {
	return []validationCase{
		{name: "valid dependency cache handoff response found", value: validDependencyCacheHandoffResponseFound()},
		{name: "valid dependency cache handoff response not found", value: validDependencyCacheHandoffResponseNotFound()},
		{name: "dependency cache handoff response requires handoff when found", value: invalidDependencyCacheHandoffResponseFoundWithoutHandoff(), wantErr: true},
	}
}

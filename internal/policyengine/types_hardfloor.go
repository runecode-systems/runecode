package policyengine

type HardFloorOperationClass string

const (
	HardFloorTrustRootChange                  HardFloorOperationClass = "trust_root_change"
	HardFloorSecurityPostureWeakening         HardFloorOperationClass = "security_posture_weakening"
	HardFloorAuthoritativeStateReconciliation HardFloorOperationClass = "authoritative_state_reconciliation"
	HardFloorDeploymentBootstrapAuthority     HardFloorOperationClass = "deployment_bootstrap_authority_change"
)

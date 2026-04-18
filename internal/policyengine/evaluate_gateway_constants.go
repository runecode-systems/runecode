package policyengine

func requiredGatewayRoleForDestination(destinationKind string) (string, bool) {
	requiredRoleForDestination := map[string]string{
		"model_endpoint":   "model-gateway",
		"auth_provider":    "auth-gateway",
		"git_remote":       "git-gateway",
		"web_origin":       "web-research",
		"package_registry": "dependency-fetch",
	}
	requiredRole, ok := requiredRoleForDestination[destinationKind]
	return requiredRole, ok
}

func isGatewayRequestExecutionOperation(operation string) bool {
	switch operation {
	case "invoke_model", "fetch_dependency", "exchange_auth_code", "refresh_auth_token":
		return true
	default:
		return false
	}
}

func isGatewayScopeChangeOperation(operation string) bool {
	switch operation {
	case "enable_gateway", "expand_scope", "change_allowlist", "enable_dependency_fetch":
		return true
	default:
		return false
	}
}

func isGatewayRemoteMutationOperation(operation string) bool {
	switch operation {
	case "git_ref_update", "git_pull_request_create":
		return true
	default:
		return false
	}
}

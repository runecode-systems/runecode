package policyengine

import "strings"

func validSecretAccessActionRequest(capabilityID string) ActionRequest {
	return newActionRequest(
		ActionKindSecretAccess,
		capabilityID,
		actionPayloadSecretAccessID,
		newSchemaPayload(actionPayloadSecretAccessID, map[string]any{
			"secret_ref":  "secrets/prod/db-password",
			"access_mode": "lease_issue",
		}),
		"workspace",
		"workspace-edit",
	)
}

func validSecretAccessLeaseRenewActionRequest(capabilityID string) ActionRequest {
	return newActionRequest(
		ActionKindSecretAccess,
		capabilityID,
		actionPayloadSecretAccessID,
		newSchemaPayload(actionPayloadSecretAccessID, map[string]any{
			"access_mode":       "lease_renew",
			"lease_id":          "lease-prod-db-001",
			"lease_ttl_seconds": 600,
			"renewal_context": map[string]any{
				"consumer_principal_ref": "principal:run-123:workspace-editor",
				"target_ref":             "db.prod.internal:5432",
				"policy_context_hash":    mustDigestObject("sha256:" + strings.Repeat("5", 64)),
			},
		}),
		"workspace",
		"workspace-edit",
	)
}

func validSecretAccessLeaseRevokeActionRequest(capabilityID string) ActionRequest {
	return newActionRequest(
		ActionKindSecretAccess,
		capabilityID,
		actionPayloadSecretAccessID,
		newSchemaPayload(actionPayloadSecretAccessID, map[string]any{
			"access_mode": "lease_revoke",
			"lease_id":    "lease-prod-db-001",
		}),
		"workspace",
		"workspace-edit",
	)
}

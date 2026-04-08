package policyengine

import (
	"encoding/json"
	"fmt"
)

func decodeRuleSet(input *ManifestInput) (*PolicyRuleSet, string, error) {
	if input == nil {
		return nil, "", nil
	}
	ruleSet := PolicyRuleSet{}
	if err := json.Unmarshal(input.Payload, &ruleSet); err != nil {
		return nil, "", err
	}
	if ruleSet.SchemaID != policyRuleSetSchemaID {
		return nil, "", schemaIDError(ruleSet.SchemaID, policyRuleSetSchemaID)
	}
	if ruleSet.SchemaVersion != policyRuleSetSchemaVersion {
		return nil, "", schemaVersionError(ruleSet.SchemaID, ruleSet.SchemaVersion, policyRuleSetSchemaVersion)
	}
	if err := validateObjectPayloadAgainstSchema(input.Payload, ruleSetSchemaPath); err != nil {
		return nil, "", &EvaluationError{Code: ErrCodeBrokerValidationSchema, Category: "validation", Retryable: false, Message: err.Error()}
	}
	for i := range ruleSet.Rules {
		if err := ensureKnownPolicyReasonCode(ruleSet.Rules[i].ReasonCode); err != nil {
			return nil, "", &EvaluationError{Code: ErrCodeBrokerValidationOperation, Category: "validation", Retryable: false, Message: fmt.Sprintf("rules[%d].reason_code: %v", i, err)}
		}
	}
	digest, err := canonicalHashBytes(input.Payload)
	if err != nil {
		return nil, "", err
	}
	if err := verifyExpectedHash(input.ExpectedHash, digest); err != nil {
		return nil, "", err
	}
	return &ruleSet, digest, nil
}

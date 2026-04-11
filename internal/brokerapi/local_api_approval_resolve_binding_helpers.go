package brokerapi

import (
	"fmt"
	"sort"
	"strings"
)

func validateApprovalRequestBindingToStoredRecord(prior approvalRecord, payload map[string]any) error {
	if err := validateStoredApprovalDigestMatch("manifest_hash", prior.ManifestHash, payload); err != nil {
		return err
	}
	if err := validateStoredApprovalDigestMatch("action_request_hash", prior.ActionRequestHash, payload); err != nil {
		return err
	}
	relevantHashes, err := digestIdentitiesFromPayloadArray(payload, "relevant_artifact_hashes")
	if err != nil {
		return fmt.Errorf("relevant_artifact_hashes: %w", err)
	}
	if err := validateStoredRelevantArtifactHashes(prior.RelevantArtifactHashes, relevantHashes); err != nil {
		return err
	}
	if err := validateStoredApprovalSourceDigest(prior.SourceDigest, relevantHashes); err != nil {
		return err
	}
	return validateStoredApprovalTrigger(prior.Summary.ApprovalTriggerCode, payload)
}

func validateStoredApprovalDigestMatch(label, expected string, payload map[string]any) error {
	actual, err := digestIdentityFromPayloadObject(payload, label)
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if strings.TrimSpace(expected) == "" {
		return fmt.Errorf("%s stored pending approval binding is missing", label)
	}
	if expected != actual {
		return fmt.Errorf("%s %q does not match stored pending approval %s %q", label, actual, label, expected)
	}
	return nil
}

func validateStoredRelevantArtifactHashes(expected, actual []string) error {
	if len(expected) == 0 {
		return nil
	}
	left := append([]string{}, expected...)
	right := append([]string{}, actual...)
	sort.Strings(left)
	sort.Strings(right)
	if equalStringSlices(left, right) {
		return nil
	}
	return fmt.Errorf("relevant_artifact_hashes do not match stored pending approval binding")
}

func validateStoredApprovalSourceDigest(sourceDigest string, relevantHashes []string) error {
	if sourceDigest != "" && !containsStringIdentity(relevantHashes, sourceDigest) {
		return fmt.Errorf("stored source_digest %q missing from request relevant_artifact_hashes", sourceDigest)
	}
	return nil
}

func validateStoredApprovalTrigger(expectedTrigger string, payload map[string]any) error {
	expectedTrigger = strings.TrimSpace(expectedTrigger)
	if expectedTrigger == "" {
		return nil
	}
	if trigger, _ := payload["approval_trigger_code"].(string); strings.TrimSpace(trigger) != expectedTrigger {
		return fmt.Errorf("approval_trigger_code %q does not match stored pending approval trigger %q", strings.TrimSpace(trigger), expectedTrigger)
	}
	return nil
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsStringIdentity(values []string, want string) bool {
	for i := range values {
		if values[i] == want {
			return true
		}
	}
	return false
}

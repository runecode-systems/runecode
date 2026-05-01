package brokerapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestSeedDevManualScenarioSeedsExternalAnchorGatewayContext(t *testing.T) {
	service := newSeededDevManualExternalAnchorServiceForTest(t)
	allowlist, role, capability := seededExternalAnchorContextForSeedTest(t, service)
	assertSeededExternalAnchorCapabilityOptIn(t, "role", role)
	assertSeededExternalAnchorCapabilityOptIn(t, "run capability", capability)
	assertSeededExternalAnchorAllowlistRefs(t, role, capability)
	assertSeededExternalAnchorAllowlistEntry(t, allowlist)
}

func TestSeedDevManualScenarioSupportsExternalAnchorPrepareForSeededRun(t *testing.T) {
	service := newSeededDevManualExternalAnchorServiceForTest(t)
	targetDescriptor, targetDescriptorDigest := seededExternalAnchorTargetDescriptorForSeedTest(t)
	resp, errResp := service.HandleExternalAnchorMutationPrepare(context.Background(), seededExternalAnchorPrepareRequestForSeedTest(targetDescriptor, targetDescriptorDigest), RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleExternalAnchorMutationPrepare returned error response: %+v", errResp)
	}
	gotTargetIdentity, _ := resp.Prepared.PrimaryTarget.TargetDescriptorDigest.Identity()
	if gotTargetIdentity != targetDescriptorDigest {
		t.Fatalf("prepared primary target digest = %q, want %q", gotTargetIdentity, targetDescriptorDigest)
	}
}

type trustedManifestSummaryForSeedTest struct {
	capabilityOptIns []string
	allowlistRefs    []string
}

type trustedContextDigestsForSeedTest struct {
	allowlists []string
	roles      []string
	runs       []string
}

func newSeededDevManualExternalAnchorServiceForTest(t *testing.T) *Service {
	t.Helper()
	if !DevManualSeedBuildEnabled() {
		t.Skip("dev manual seed is disabled in this build")
	}
	service := newDevManualSeedService(t)
	t.Setenv(devManualSeedEnvVar, "1")
	if _, err := service.SeedDevManualScenario(); err != nil {
		t.Fatalf("SeedDevManualScenario returned error: %v", err)
	}
	return service
}

func seededExternalAnchorContextForSeedTest(t *testing.T, service *Service) (map[string]any, trustedManifestSummaryForSeedTest, trustedManifestSummaryForSeedTest) {
	t.Helper()
	allowlist, role, capability, err := trustedPolicyContextSnapshotForRunSeedTest(service, instanceControlRunIDForInstanceID(devManualSeedInstanceID))
	if err != nil {
		t.Fatalf("trustedPolicyContextSnapshotForRunSeedTest returned error: %v", err)
	}
	return allowlist, role, capability
}

func seededExternalAnchorTargetDescriptorForSeedTest(t *testing.T) (map[string]any, string) {
	t.Helper()
	targetDescriptorDigest, err := devManualExternalAnchorTargetDescriptorDigest()
	if err != nil {
		t.Fatalf("devManualExternalAnchorTargetDescriptorDigest returned error: %v", err)
	}
	return map[string]any{
		"descriptor_schema_id":   "runecode.protocol.audit.anchor_target.transparency_log.v0",
		"log_id":                 "manual-seed-transparency-log",
		"log_public_key_digest":  digestObject("sha256:" + strings.Repeat("d", 64)),
		"entry_encoding_profile": "jcs_v1",
	}, targetDescriptorDigest
}

func seededExternalAnchorPrepareRequestForSeedTest(targetDescriptor map[string]any, targetDescriptorDigest string) ExternalAnchorMutationPrepareRequest {
	return ExternalAnchorMutationPrepareRequest{
		SchemaID:      "runecode.protocol.v0.ExternalAnchorMutationPrepareRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-manual-seed-external-anchor-prepare",
		RunID:         devManualSeedRunID,
		TypedRequest: map[string]any{
			"schema_id":                "runecode.protocol.v0.ExternalAnchorSubmitRequest",
			"schema_version":           "0.1.0",
			"request_kind":             "external_anchor_submit_v0",
			"target_kind":              "transparency_log",
			"target_descriptor":        targetDescriptor,
			"target_descriptor_digest": digestObject(targetDescriptorDigest),
			"target_set": []any{map[string]any{
				"target_kind":              "transparency_log",
				"target_requirement":       "required",
				"target_descriptor":        targetDescriptor,
				"target_descriptor_digest": digestObject(targetDescriptorDigest),
			}},
			"seal_digest":             digestObject("sha256:" + strings.Repeat("b", 64)),
			"outbound_payload_digest": digestObject("sha256:" + strings.Repeat("c", 64)),
		},
	}
}

func assertSeededExternalAnchorCapabilityOptIn(t *testing.T, label string, summary trustedManifestSummaryForSeedTest) {
	t.Helper()
	if !containsString(summary.capabilityOptIns, "cap_external_anchor") {
		t.Fatalf("%s capability_opt_ins = %v, want cap_external_anchor", label, summary.capabilityOptIns)
	}
}

func assertSeededExternalAnchorAllowlistRefs(t *testing.T, role, capability trustedManifestSummaryForSeedTest) {
	t.Helper()
	if len(role.allowlistRefs) != 1 || len(capability.allowlistRefs) != 1 {
		t.Fatalf("allowlist refs role=%v capability=%v, want one each", role.allowlistRefs, capability.allowlistRefs)
	}
	if role.allowlistRefs[0] != capability.allowlistRefs[0] {
		t.Fatalf("role/capability allowlist refs mismatch: %q vs %q", role.allowlistRefs[0], capability.allowlistRefs[0])
	}
}

func assertSeededExternalAnchorAllowlistEntry(t *testing.T, allowlist map[string]any) {
	t.Helper()
	entries, ok := allowlist["entries"].([]any)
	if !ok {
		t.Fatalf("allowlist entries has type %T, want []any", allowlist["entries"])
	}
	for _, raw := range entries {
		if seededExternalAnchorAllowlistEntryFound(raw) {
			return
		}
	}
	t.Fatal("seeded allowlist missing git-gateway external anchor entry")
}

func seededExternalAnchorAllowlistEntryFound(raw any) bool {
	entry, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if strings.TrimSpace(stringField(entry, "gateway_role_kind")) != "git-gateway" {
		return false
	}
	targetDigests, ok := entry["external_anchor_target_descriptor_digests"].([]any)
	if !ok || len(targetDigests) == 0 {
		return false
	}
	digestObj, ok := targetDigests[0].(map[string]any)
	if !ok {
		return false
	}
	return digestIdentityFromObjectForSeedTest(digestObj) != ""
}

func trustedPolicyContextSnapshotForRunSeedTest(service *Service, runID string) (map[string]any, trustedManifestSummaryForSeedTest, trustedManifestSummaryForSeedTest, error) {
	events, err := service.ReadAuditEvents()
	if err != nil {
		return nil, trustedManifestSummaryForSeedTest{}, trustedManifestSummaryForSeedTest{}, err
	}
	digests, err := collectTrustedContextDigestsForSeedTest(service, events, runID)
	if err != nil {
		return nil, trustedManifestSummaryForSeedTest{}, trustedManifestSummaryForSeedTest{}, err
	}
	allowlist, err := trustedContextArtifactJSONForSeedTest(service, digests.allowlists[len(digests.allowlists)-1])
	if err != nil {
		return nil, trustedManifestSummaryForSeedTest{}, trustedManifestSummaryForSeedTest{}, err
	}
	role, err := selectTrustedManifestSummaryForSeedTest(service, digests.roles, "cap_external_anchor")
	if err != nil {
		return nil, trustedManifestSummaryForSeedTest{}, trustedManifestSummaryForSeedTest{}, err
	}
	run, err := selectTrustedManifestSummaryForSeedTest(service, digests.runs, "cap_external_anchor")
	if err != nil {
		return nil, trustedManifestSummaryForSeedTest{}, trustedManifestSummaryForSeedTest{}, err
	}
	return allowlist, role, run, nil
}

func collectTrustedContextDigestsForSeedTest(service *Service, events []artifacts.AuditEvent, runID string) (trustedContextDigestsForSeedTest, error) {
	digests := trustedContextDigestsForSeedTest{}
	for _, event := range events {
		appendTrustedContextDigestForSeedTest(service, runID, event, &digests)
	}
	if len(digests.allowlists) == 0 || len(digests.roles) == 0 || len(digests.runs) == 0 {
		return trustedContextDigestsForSeedTest{}, fmt.Errorf("trusted context artifacts missing for run %q", runID)
	}
	return digests, nil
}

func appendTrustedContextDigestForSeedTest(service *Service, runID string, event artifacts.AuditEvent, digests *trustedContextDigestsForSeedTest) {
	kind, digest, ok := trustedContextDigestForRunSeedTest(service, runID, event)
	if !ok {
		return
	}
	switch kind {
	case artifacts.TrustedContractImportKindPolicyAllowlist:
		digests.allowlists = append(digests.allowlists, digest)
	case artifacts.TrustedContractImportKindRoleManifest:
		digests.roles = append(digests.roles, digest)
	case artifacts.TrustedContractImportKindRunCapability:
		digests.runs = append(digests.runs, digest)
	}
}

func trustedContextDigestForRunSeedTest(service *Service, runID string, event artifacts.AuditEvent) (string, string, bool) {
	if event.Type != artifacts.TrustedContractImportAuditEventType {
		return "", "", false
	}
	kind, _ := event.Details[artifacts.TrustedContractImportKindDetailKey].(string)
	digest, _ := event.Details[artifacts.TrustedContractImportArtifactDigestDetailKey].(string)
	if strings.TrimSpace(kind) == "" || strings.TrimSpace(digest) == "" {
		return "", "", false
	}
	rec, err := service.Head(digest)
	if err != nil || rec.RunID != runID {
		return "", "", false
	}
	return kind, digest, true
}

func selectTrustedManifestSummaryForSeedTest(service *Service, digests []string, requiredCapability string) (trustedManifestSummaryForSeedTest, error) {
	for i := len(digests) - 1; i >= 0; i-- {
		payload, err := trustedContextArtifactJSONForSeedTest(service, digests[i])
		if err != nil {
			continue
		}
		summary := trustedManifestSummaryForSeedTest{
			capabilityOptIns: stringSliceFromAnyForSeedTest(payload["capability_opt_ins"]),
			allowlistRefs:    digestIdentityArrayFromAnyForSeedTest(payload["allowlist_refs"]),
		}
		if containsString(summary.capabilityOptIns, requiredCapability) {
			return summary, nil
		}
	}
	return trustedManifestSummaryForSeedTest{}, fmt.Errorf("trusted manifest with capability %q not found", requiredCapability)
}

func trustedContextArtifactJSONForSeedTest(service *Service, digest string) (map[string]any, error) {
	reader, err := service.Get(digest)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func stringSliceFromAnyForSeedTest(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}

func digestIdentityArrayFromAnyForSeedTest(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		identity := digestIdentityFromObjectForSeedTest(obj)
		if identity != "" {
			out = append(out, identity)
		}
	}
	return out
}

func digestIdentityFromObjectForSeedTest(digest map[string]any) string {
	hashAlg, _ := digest["hash_alg"].(string)
	hash, _ := digest["hash"].(string)
	hashAlg = strings.TrimSpace(hashAlg)
	hash = strings.TrimSpace(hash)
	if hashAlg == "" || hash == "" {
		return ""
	}
	return hashAlg + ":" + hash
}

package brokerapi

import (
	"crypto/ed25519"
	"path/filepath"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func newPersistentBrokerServiceForE2ETest(t *testing.T) (string, string, *Service) {
	t.Helper()
	root := t.TempDir()
	storeRoot := filepath.Join(root, "store")
	ledgerRoot := filepath.Join(root, "ledger")
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	s, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t)})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	return storeRoot, ledgerRoot, s
}

func reopenPersistentBrokerServiceForE2ETest(t *testing.T, storeRoot, ledgerRoot string) *Service {
	t.Helper()
	s, err := NewServiceWithConfig(storeRoot, ledgerRoot, APIConfig{RepositoryRoot: repositoryRootForProjectSubstrateTests(t)})
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	return s
}

func putTrustedPolicyContextForE2ERun(t *testing.T, s *Service, in e2eContextInput) {
	t.Helper()
	verifier, privateKey := newSignedContextVerifierFixture(t)
	if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
		t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
	}
	allowlistArray := digestObjectArray(in.allowlistRefs)
	rolePayload := signedRolePayloadForE2E(t, in, verifier, privateKey, allowlistArray)
	putTrustedPolicyArtifact(t, s, in.runID, artifacts.TrustedContractImportKindRoleManifest, rolePayload)
	runPayload := signedRunPayloadForE2E(t, in, verifier, privateKey, allowlistArray)
	putTrustedPolicyArtifact(t, s, in.runID, artifacts.TrustedContractImportKindRunCapability, runPayload)
	putOptionalRuleSetForE2E(t, s, in)
}

func digestObjectArray(refs []string) []any {
	allowlistArray := make([]any, 0, len(refs))
	for _, ref := range refs {
		allowlistArray = append(allowlistArray, digestObject(ref))
	}
	return allowlistArray
}

func signedRolePayloadForE2E(t *testing.T, in e2eContextInput, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey, allowlistArray []any) []byte {
	t.Helper()
	return signedPayloadForTrustedContext(t, map[string]any{
		"schema_id":          "runecode.protocol.v0.RoleManifest",
		"schema_version":     "0.2.0",
		"principal":          signedContextPrincipal(in.roleFamily, in.roleKind, in.runID, ""),
		"role_family":        in.roleFamily,
		"role_kind":          in.roleKind,
		"approval_profile":   "moderate",
		"capability_opt_ins": toAnyStrings(in.capabilities),
		"allowlist_refs":     allowlistArray,
	}, verifier, privateKey)
}

func signedRunPayloadForE2E(t *testing.T, in e2eContextInput, verifier trustpolicy.VerifierRecord, privateKey ed25519.PrivateKey, allowlistArray []any) []byte {
	t.Helper()
	return signedPayloadForTrustedContext(t, map[string]any{
		"schema_id":          "runecode.protocol.v0.CapabilityManifest",
		"schema_version":     "0.2.0",
		"principal":          signedContextPrincipal(in.roleFamily, in.roleKind, in.runID, ""),
		"manifest_scope":     "run",
		"run_id":             in.runID,
		"approval_profile":   "moderate",
		"capability_opt_ins": toAnyStrings(in.capabilities),
		"allowlist_refs":     allowlistArray,
	}, verifier, privateKey)
}

func putOptionalRuleSetForE2E(t *testing.T, s *Service, in e2eContextInput) {
	t.Helper()
	if len(in.ruleSetRules) == 0 {
		return
	}
	ruleSet := map[string]any{
		"schema_id":      "runecode.protocol.v0.PolicyRuleSet",
		"schema_version": "0.1.0",
		"rules":          in.ruleSetRules,
	}
	putTrustedPolicyArtifact(t, s, in.runID, artifacts.TrustedContractImportKindPolicyRuleSet, mustJSONBytes(t, ruleSet))
}

func pendingApprovalForRunAndKind(t *testing.T, s *Service, runID, actionKind string) artifacts.ApprovalRecord {
	t.Helper()
	for _, rec := range s.ApprovalList() {
		if rec.RunID == runID && rec.ActionKind == actionKind && rec.Status == "pending" {
			return rec
		}
	}
	t.Fatalf("missing pending approval for run=%q action_kind=%q", runID, actionKind)
	return artifacts.ApprovalRecord{}
}

func reopenAndRequirePendingPromotionApproval(t *testing.T, storeRoot, ledgerRoot, runID, decisionDigest string) (*Service, artifacts.ApprovalRecord) {
	t.Helper()
	s := reopenPersistentBrokerServiceForE2ETest(t, storeRoot, ledgerRoot)
	pending := pendingApprovalForRunAndKind(t, s, runID, policyengine.ActionKindPromotion)
	if pending.PolicyDecisionHash != decisionDigest {
		t.Fatalf("durable pending policy_decision_hash = %q, want %q", pending.PolicyDecisionHash, decisionDigest)
	}
	return s, pending
}

func signedPromotionResolveRequest(t *testing.T, s *Service, pending artifacts.ApprovalRecord, sourceDigest string) ApprovalResolveRequest {
	t.Helper()
	requestEnvelope, decisionEnvelope, verifier := signedResolveEnvelopesForStoredPendingRequest(t, *pending.RequestEnvelope, "human", "approve")
	if err := putTrustedVerifierRecordForService(s, verifier); err != nil {
		t.Fatalf("putTrustedVerifierRecordForService returned error: %v", err)
	}
	return ApprovalResolveRequest{
		SchemaID:      "runecode.protocol.v0.ApprovalResolveRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-e2e-promotion-resolve",
		ApprovalID:    pending.ApprovalID,
		BoundScope: ApprovalBoundScope{
			SchemaID:           "runecode.protocol.v0.ApprovalBoundScope",
			SchemaVersion:      "0.1.0",
			WorkspaceID:        pending.WorkspaceID,
			RunID:              pending.RunID,
			StageID:            pending.StageID,
			StepID:             pending.StepID,
			ActionKind:         pending.ActionKind,
			PolicyDecisionHash: pending.PolicyDecisionHash,
		},
		UnapprovedDigest:       sourceDigest,
		Approver:               "human",
		RepoPath:               "repo/file.txt",
		Commit:                 "abc123",
		ExtractorToolVersion:   "tool-v1",
		FullContentVisible:     true,
		ExplicitViewFull:       false,
		BulkRequest:            false,
		BulkApprovalConfirmed:  false,
		SignedApprovalRequest:  requestEnvelope,
		SignedApprovalDecision: decisionEnvelope,
	}
}

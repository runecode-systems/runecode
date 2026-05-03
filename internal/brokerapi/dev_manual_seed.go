package brokerapi

import (
	"fmt"
	"os"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/launcherbackend"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

const (
	devManualSeedDefaultProfile    = "tui-rich-v1"
	devManualSeedDegradedProfile   = "tui-rich-degraded-v1"
	devManualSeedEnvVar            = "RUNECODE_DEV_MODE"
	devManualSeedRunID             = "run-manual-001"
	devManualSeedSessionID         = "session-manual-001"
	devManualSeedWorkspaceID       = "workspace-manual"
	devManualSeedStageID           = "stage-apply"
	devManualSeedRoleInstanceID    = "workspace-1"
	devManualSeedRecordedAtRFC3339 = "2026-03-13T12:15:00Z"
)

type DevManualSeedResult struct {
	SchemaID          string   `json:"schema_id"`
	SchemaVersion     string   `json:"schema_version"`
	Profile           string   `json:"profile"`
	RunID             string   `json:"run_id"`
	SessionID         string   `json:"session_id"`
	ApprovalID        string   `json:"approval_id"`
	AuditRecordDigest string   `json:"audit_record_digest"`
	ArtifactDigests   []string `json:"artifact_digests"`
}

// SeedDevManualScenario seeds a deterministic, dev-focused broker state profile
// for manual TUI/broker workflow drills.
func (s *Service) SeedDevManualScenario() (DevManualSeedResult, error) {
	return s.SeedDevManualScenarioWithProfile(devManualSeedDefaultProfile)
}

func SupportedDevManualSeedProfiles() []string {
	return []string{devManualSeedDefaultProfile, devManualSeedDegradedProfile}
}

func (s *Service) SeedDevManualScenarioWithProfile(profile string) (DevManualSeedResult, error) {
	seedProfile, err := normalizeDevManualSeedProfile(profile)
	if err != nil {
		return DevManualSeedResult{}, err
	}
	if s == nil || s.store == nil {
		return DevManualSeedResult{}, fmt.Errorf("broker store unavailable")
	}
	if !DevManualSeedBuildEnabled() {
		return DevManualSeedResult{}, fmt.Errorf("dev manual seeding unavailable in this build")
	}
	if strings.TrimSpace(os.Getenv(devManualSeedEnvVar)) != "1" {
		return DevManualSeedResult{}, fmt.Errorf("dev manual seeding requires %s=1", devManualSeedEnvVar)
	}
	seedState, err := s.seedDevManualScenarioState(seedProfile)
	if err != nil {
		return DevManualSeedResult{}, err
	}
	return DevManualSeedResult{
		SchemaID:          "runecode.protocol.v0.DevManualSeedResult",
		SchemaVersion:     "0.1.0",
		Profile:           seedProfile,
		RunID:             devManualSeedRunID,
		SessionID:         devManualSeedSessionID,
		ApprovalID:        seedState.approvalID,
		AuditRecordDigest: seedState.auditRecordDigest,
		ArtifactDigests:   seedState.artifactDigests,
	}, nil
}

func DevManualSeedBuildEnabled() bool { return devManualSeedBuildEnabled }

type devManualSeedState struct {
	auditRecordDigest string
	artifactDigests   []string
	approvalID        string
}

func normalizeDevManualSeedProfile(profile string) (string, error) {
	value := strings.TrimSpace(profile)
	for _, supported := range SupportedDevManualSeedProfiles() {
		if value == supported {
			return value, nil
		}
	}
	return "", fmt.Errorf("unsupported dev manual seed profile %q", profile)
}

func NormalizeDevManualSeedProfile(profile string) (string, error) {
	return normalizeDevManualSeedProfile(profile)
}

func (s *Service) seedDevManualScenarioState(profile string) (devManualSeedState, error) {
	recordDigest, err := s.seedDevManualAuditLedger(profile)
	if err != nil {
		return devManualSeedState{}, err
	}
	if err := s.seedDevManualPolicy(); err != nil {
		return devManualSeedState{}, err
	}
	artifactDigests, err := s.seedDevManualArtifacts()
	if err != nil {
		return devManualSeedState{}, err
	}
	if err := s.seedDevManualRuntimeFacts(); err != nil {
		return devManualSeedState{}, err
	}
	approvalID, err := s.seedDevManualApproval(profile)
	if err != nil {
		return devManualSeedState{}, err
	}
	if err := s.seedDevManualSession(approvalID, recordDigest, artifactDigests, profile); err != nil {
		return devManualSeedState{}, err
	}
	if err := s.ensureDevManualSessionAuditLink(recordDigest, profile); err != nil {
		return devManualSeedState{}, err
	}
	return devManualSeedState{auditRecordDigest: recordDigest, artifactDigests: artifactDigests, approvalID: approvalID}, nil
}

func (s *Service) seedDevManualAuditLedger(profile string) (string, error) {
	recordDigest, err := seedDevManualAuditLedger(s.auditRoot, profile)
	if err != nil {
		return "", err
	}
	reopenedLedger, err := auditd.Open(s.auditRoot)
	if err != nil {
		return "", err
	}
	s.auditLedger = reopenedLedger
	return recordDigest, nil
}

func (s *Service) seedDevManualPolicy() error {
	policy := s.Policy()
	flowRule := artifacts.FlowRule{
		ProducerRole:       "workspace",
		ConsumerRole:       "model_gateway",
		AllowedDataClasses: []artifacts.DataClass{artifacts.DataClassDiffs, artifacts.DataClassBuildLogs, artifacts.DataClassGateEvidence, artifacts.DataClassAuditVerificationReport},
	}
	policy.FlowMatrix = upsertDevManualFlowRule(policy.FlowMatrix, flowRule)
	if err := s.SetPolicy(policy); err != nil {
		return err
	}
	return s.seedDevManualInstanceControlContext()
}

func (s *Service) seedDevManualArtifacts() ([]string, error) {
	puts := []artifacts.PutRequest{
		{Payload: []byte("spec: manual scenario for broker/TUI route drill-down"), ContentType: "text/plain", DataClass: artifacts.DataClassSpecText, ProvenanceReceiptHash: digestWithByte("1"), CreatedByRole: "workspace", RunID: devManualSeedRunID, StepID: "plan"},
		{Payload: []byte("diff --git a/file b/file\n+manual seeded line\n"), ContentType: "text/plain", DataClass: artifacts.DataClassDiffs, ProvenanceReceiptHash: digestWithByte("2"), CreatedByRole: "workspace", RunID: devManualSeedRunID, StepID: "apply"},
		{Payload: []byte("build: waiting for approval gate"), ContentType: "text/plain", DataClass: artifacts.DataClassBuildLogs, ProvenanceReceiptHash: digestWithByte("3"), CreatedByRole: "workspace", RunID: devManualSeedRunID, StepID: "build"},
	}
	digests := make([]string, 0, len(puts))
	for _, req := range puts {
		ref, err := s.Put(req)
		if err != nil {
			return nil, err
		}
		digests = append(digests, ref.Digest)
	}
	return uniqueSortedStrings(digests), nil
}

func (s *Service) seedDevManualRuntimeFacts() error {
	return s.RecordRuntimeFacts(devManualSeedRunID, launcherbackend.RuntimeFactsSnapshot{LaunchReceipt: launcherbackend.BackendLaunchReceipt{
		RunID:                        devManualSeedRunID,
		StageID:                      devManualSeedStageID,
		RoleInstanceID:               devManualSeedRoleInstanceID,
		BackendKind:                  "microvm",
		IsolationAssuranceLevel:      "isolated",
		ProvisioningPosture:          "tofu",
		IsolateID:                    "isolate-manual-001",
		SessionID:                    devManualSeedSessionID,
		SessionNonce:                 "nonce-manual-001",
		LaunchContextDigest:          digestWithByte("a"),
		HandshakeTranscriptHash:      digestWithByte("b"),
		IsolateSessionKeyIDValue:     strings.Repeat("c", 64),
		RuntimeImageDescriptorDigest: digestWithByte("d"),
	}})
}

func (s *Service) seedDevManualApproval(profile string) (string, error) {
	decision, err := devManualApprovalDecision(profile)
	if err != nil {
		return "", err
	}
	if err := s.RecordPolicyDecision(devManualSeedRunID, "", decision); err != nil {
		return "", err
	}
	decisionDigest, ok, err := s.latestSeedPolicyDecisionDigest(devManualSeedRunID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("seed policy decision not found for run %q", devManualSeedRunID)
	}
	for _, approval := range s.ApprovalList() {
		if approval.PolicyDecisionHash == decisionDigest {
			return approval.ApprovalID, nil
		}
	}
	return "", fmt.Errorf("seed approval not found for policy decision %q", decisionDigest)
}

func (s *Service) latestSeedPolicyDecisionDigest(runID string) (string, bool, error) {
	refs := s.PolicyDecisionRefsForRun(runID)
	for i := len(refs) - 1; i >= 0; i-- {
		ref := refs[i]
		record, ok := s.PolicyDecisionGet(ref)
		if !ok {
			continue
		}
		if isDevManualApprovalDecision(record) {
			return record.Digest, true, nil
		}
	}
	return "", false, nil
}

func isDevManualApprovalDecision(record artifacts.PolicyDecisionRecord) bool {
	if record.RunID != devManualSeedRunID || record.DecisionOutcome != string(policyengine.DecisionRequireHumanApproval) {
		return false
	}
	if record.ManifestHash != digestWithByte("e") || record.ActionRequestHash != digestWithByte("f") {
		return false
	}
	precedence, _ := record.Details["precedence"].(string)
	return strings.HasPrefix(precedence, "manual_seed_profile:")
}

func digestWithByte(ch string) string {
	return "sha256:" + strings.Repeat(ch, 64)
}

func upsertDevManualFlowRule(existing []artifacts.FlowRule, target artifacts.FlowRule) []artifacts.FlowRule {
	for i := range existing {
		if existing[i].ProducerRole != target.ProducerRole || existing[i].ConsumerRole != target.ConsumerRole {
			continue
		}
		existing[i].AllowedDataClasses = mergeAllowedDataClasses(existing[i].AllowedDataClasses, target.AllowedDataClasses)
		return existing
	}
	return append(existing, target)
}

func mergeAllowedDataClasses(existing, added []artifacts.DataClass) []artifacts.DataClass {
	seen := map[artifacts.DataClass]struct{}{}
	out := make([]artifacts.DataClass, 0, len(existing)+len(added))
	for _, class := range existing {
		if _, ok := seen[class]; ok {
			continue
		}
		seen[class] = struct{}{}
		out = append(out, class)
	}
	for _, class := range added {
		if _, ok := seen[class]; ok {
			continue
		}
		seen[class] = struct{}{}
		out = append(out, class)
	}
	return out
}

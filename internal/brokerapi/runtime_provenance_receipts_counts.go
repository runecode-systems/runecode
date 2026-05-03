package brokerapi

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

type runtimeProvenanceCounts struct {
	authorizedProviderCount int64
	deniedProviderCount     int64
	leaseIssueCount         int64
	leaseRevokeCount        int64
}

func runtimeSummaryKindPresent(receipts []trustpolicy.SignedObjectEnvelope, kind string) bool {
	for i := range receipts {
		payload := struct {
			Kind string `json:"audit_receipt_kind"`
		}{}
		if strings.TrimSpace(receipts[i].PayloadSchemaID) != trustpolicy.AuditReceiptSchemaID {
			continue
		}
		if err := json.Unmarshal(receipts[i].Payload, &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.Kind) == strings.TrimSpace(kind) {
			return true
		}
	}
	return false
}

func countRuntimeProvenanceReceipts(receipts []trustpolicy.SignedObjectEnvelope) (runtimeProvenanceCounts, map[string]struct{}) {
	counts := runtimeProvenanceCounts{}
	runDigests := map[string]struct{}{}
	for i := range receipts {
		payload, ok := decodeRuntimeReceiptEnvelope(receipts[i])
		if !ok {
			continue
		}
		observeRuntimeReceiptPayload(&counts, payload, runDigests)
	}
	return counts, runDigests
}

func observeRuntimeReceiptPayload(counts *runtimeProvenanceCounts, payload struct {
	Kind           string          `json:"audit_receipt_kind"`
	ReceiptPayload json.RawMessage `json:"receipt_payload"`
}, runDigests map[string]struct{}) {
	switch strings.TrimSpace(payload.Kind) {
	case "provider_invocation_authorized":
		counts.authorizedProviderCount++
	case "provider_invocation_denied":
		counts.deniedProviderCount++
	case "secret_lease_issued":
		counts.leaseIssueCount++
	case "secret_lease_revoked":
		counts.leaseRevokeCount++
	default:
		return
	}
	collectRuntimeReceiptRunDigest(payload.ReceiptPayload, runDigests)
}

func decodeRuntimeReceiptEnvelope(envelope trustpolicy.SignedObjectEnvelope) (struct {
	Kind           string          `json:"audit_receipt_kind"`
	ReceiptPayload json.RawMessage `json:"receipt_payload"`
}, bool) {
	payload := struct {
		Kind           string          `json:"audit_receipt_kind"`
		ReceiptPayload json.RawMessage `json:"receipt_payload"`
	}{}
	if strings.TrimSpace(envelope.PayloadSchemaID) != trustpolicy.AuditReceiptSchemaID {
		return payload, false
	}
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return payload, false
	}
	return payload, true
}

func collectRuntimeReceiptRunDigest(raw json.RawMessage, runDigests map[string]struct{}) {
	payload := struct {
		RunIDDigest *trustpolicy.Digest `json:"run_id_digest,omitempty"`
	}{}
	if err := json.Unmarshal(raw, &payload); err != nil || payload.RunIDDigest == nil {
		return
	}
	if identity, err := payload.RunIDDigest.Identity(); err == nil {
		runDigests[identity] = struct{}{}
	}
}

func runtimeApprovalConsumptionForRunDigests(runDigests map[string]struct{}, approvals []ApprovalSummary) (int64, string, []string) {
	runIDs := runIDsForReceiptDigests(runDigests, approvals)
	if len(runIDs) == 0 {
		return 0, "limited", nil
	}
	return consumedApprovalCount(runIDs, approvals), "explicit", runIDs
}

func runIDsForReceiptDigests(runDigests map[string]struct{}, approvals []ApprovalSummary) []string {
	if len(runDigests) == 0 {
		return nil
	}
	runIDSet := map[string]struct{}{}
	for _, approval := range approvals {
		runID := strings.TrimSpace(approval.BoundScope.RunID)
		if approvalRunDigestMatches(runID, runDigests) {
			runIDSet[runID] = struct{}{}
		}
	}
	runIDs := make([]string, 0, len(runIDSet))
	for runID := range runIDSet {
		runIDs = append(runIDs, runID)
	}
	sort.Strings(runIDs)
	return runIDs
}

func approvalRunDigestMatches(runID string, runDigests map[string]struct{}) bool {
	if strings.TrimSpace(runID) == "" {
		return false
	}
	runDigest := hashIdentityDigest(runID)
	if runDigest == nil {
		return false
	}
	runIdentity, err := runDigest.Identity()
	if err != nil {
		return false
	}
	_, ok := runDigests[runIdentity]
	return ok
}

func consumedApprovalCount(runIDs []string, approvals []ApprovalSummary) int64 {
	runs := runIDSet(runIDs)
	var consumed int64
	for _, approval := range approvals {
		if !approvalMatchesRuns(approval, runs) {
			continue
		}
		if strings.TrimSpace(approval.Status) == "consumed" {
			consumed++
		}
	}
	return consumed
}

func runtimeBoundaryCrossingsForRuns(runIDs []string, records []artifacts.ArtifactRecord) (int64, string) {
	if len(runIDs) == 0 {
		return 0, "limited"
	}
	runs := runIDSet(runIDs)
	var count int64
	for i := range records {
		if !artifactRecordMatchesRuntimeBoundaryRun(records[i], runs) {
			continue
		}
		count++
	}
	return count, "explicit"
}

func artifactRecordMatchesRuntimeBoundaryRun(record artifacts.ArtifactRecord, runs map[string]struct{}) bool {
	runID := strings.TrimSpace(record.RunID)
	if runID == "" {
		return false
	}
	if _, ok := runs[runID]; !ok {
		return false
	}
	return record.Reference.DataClass == artifacts.DataClassApprovedFileExcerpts
}

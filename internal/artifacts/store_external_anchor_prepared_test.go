package artifacts

import (
	"testing"
	"time"
)

func TestExternalAnchorPreparedIDsSorted(t *testing.T) {
	store := newTestStore(t)
	for _, preparedID := range []string{"prepared-b", "prepared-a", "prepared-c"} {
		rec := externalAnchorPreparedRecordFixture(preparedID)
		if err := store.ExternalAnchorPreparedUpsert(rec); err != nil {
			t.Fatalf("ExternalAnchorPreparedUpsert(%q) returned error: %v", preparedID, err)
		}
	}
	ids := store.ExternalAnchorPreparedIDs()
	if len(ids) != 3 || ids[0] != "prepared-a" || ids[1] != "prepared-b" || ids[2] != "prepared-c" {
		t.Fatalf("ExternalAnchorPreparedIDs() = %v, want [prepared-a prepared-b prepared-c]", ids)
	}
}

func TestExternalAnchorPreparedClaimDeferredExecutionRespectsStaleness(t *testing.T) {
	store := newTestStore(t)
	rec := externalAnchorPreparedRecordFixture("prepared-claim")
	if err := store.ExternalAnchorPreparedUpsert(rec); err != nil {
		t.Fatalf("ExternalAnchorPreparedUpsert returned error: %v", err)
	}

	start := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	claimed, ok, err := store.ExternalAnchorPreparedClaimDeferredExecution(rec.PreparedMutationID, rec.LastExecuteAttemptID, "worker-a", 30*time.Second, start)
	assertExternalAnchorPreparedClaimSucceeded(t, claimed, ok, err, "first", "worker-a")

	blocked, ok, err := store.ExternalAnchorPreparedClaimDeferredExecution(rec.PreparedMutationID, rec.LastExecuteAttemptID, "worker-b", 30*time.Second, start.Add(10*time.Second))
	assertExternalAnchorPreparedClaimBlocked(t, blocked, ok, err, "second")

	takenOver, ok, err := store.ExternalAnchorPreparedClaimDeferredExecution(rec.PreparedMutationID, rec.LastExecuteAttemptID, "worker-b", 30*time.Second, start.Add(31*time.Second))
	assertExternalAnchorPreparedClaimSucceeded(t, takenOver, ok, err, "third", "worker-b")
}

func assertExternalAnchorPreparedClaimSucceeded(t *testing.T, claimed ExternalAnchorPreparedMutationRecord, ok bool, err error, label, wantClaimID string) {
	t.Helper()
	if err != nil {
		t.Fatalf("ExternalAnchorPreparedClaimDeferredExecution(%s) returned error: %v", label, err)
	}
	if !ok {
		t.Fatalf("%s claim rejected, want claimed", label)
	}
	if claimed.LastExecuteDeferredClaimID != wantClaimID || claimed.LastExecuteDeferredClaimedAt == nil {
		t.Fatalf("%s claim not persisted: claim_id=%q claimed_at=%v", label, claimed.LastExecuteDeferredClaimID, claimed.LastExecuteDeferredClaimedAt)
	}
}

func assertExternalAnchorPreparedClaimBlocked(t *testing.T, blocked ExternalAnchorPreparedMutationRecord, ok bool, err error, label string) {
	t.Helper()
	if err != nil {
		t.Fatalf("ExternalAnchorPreparedClaimDeferredExecution(%s) returned error: %v", label, err)
	}
	if ok {
		t.Fatalf("%s claim unexpectedly succeeded: %+v", label, blocked)
	}
}

func externalAnchorPreparedRecordFixture(preparedID string) ExternalAnchorPreparedMutationRecord {
	targetDigest := testDigest("4")
	targetDescriptor := map[string]any{
		"descriptor_schema_id":   "runecode.protocol.audit.anchor_target.transparency_log.v0",
		"log_id":                 "transparency-log-" + preparedID,
		"log_public_key_digest":  map[string]any{"hash_alg": "sha256", "hash": testDigest("5")[7:]},
		"entry_encoding_profile": "jcs_v1",
	}
	return ExternalAnchorPreparedMutationRecord{
		PreparedMutationID:       preparedID,
		RunID:                    "run-" + preparedID,
		DestinationRef:           "sha256:" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		PrimaryTarget:            ExternalAnchorPreparedTargetBinding{TargetKind: "transparency_log", TargetRequirement: "required", TargetDescriptor: targetDescriptor, TargetDescriptorDigest: targetDigest},
		TargetSet:                []ExternalAnchorPreparedTargetBinding{{TargetKind: "transparency_log", TargetRequirement: "required", TargetDescriptor: targetDescriptor, TargetDescriptorDigest: targetDigest}},
		RequestKind:              "external_anchor_submit",
		TypedRequestSchemaID:     "runecode.protocol.v0.ExternalAnchorTypedRequest",
		TypedRequestSchemaVer:    "0.1.0",
		TypedRequest:             map[string]any{"execution_mode": "deferred_poll"},
		TypedRequestHash:         testDigest("1"),
		ActionRequestHash:        testDigest("2"),
		PolicyDecisionHash:       testDigest("3"),
		LifecycleState:           "prepared",
		ExecutionState:           "deferred",
		LastExecuteAttemptID:     "attempt-" + preparedID,
		LastExecuteDeferredPolls: 2,
	}
}

package artifacts

import (
	"strings"
	"testing"
	"time"
)

func TestPutGateEvidenceStoresTypedArtifact(t *testing.T) {
	store := newTestStore(t)
	ref, err := store.PutGateEvidence("run-evidence", GateEvidenceArtifact{
		SchemaID:      "runecode.protocol.v0.GateEvidence",
		SchemaVersion: "0.1.0",
		GateID:        "policy_gate",
		GateKind:      "policy",
		GateVersion:   "1.0.0",
		RunID:         "run-evidence",
		GateAttemptID: "gate-attempt-1",
		StartedAt:     time.Date(2026, 4, 2, 11, 0, 0, 0, time.UTC).Format(time.RFC3339),
		FinishedAt:    time.Date(2026, 4, 2, 11, 2, 0, 0, time.UTC).Format(time.RFC3339),
		Runtime:       map[string]any{"tool": "policyengine"},
		Outcome:       map[string]any{"deterministic_outcome": "failed"},
	})
	if err != nil {
		t.Fatalf("PutGateEvidence returned error: %v", err)
	}
	if !strings.HasPrefix(ref.Digest, "sha256:") {
		t.Fatalf("digest = %q, want sha256 identity", ref.Digest)
	}
	rec, err := store.Head(ref.Digest)
	if err != nil {
		t.Fatalf("Head(gate evidence) returned error: %v", err)
	}
	if rec.Reference.DataClass != DataClassGateEvidence {
		t.Fatalf("data_class = %q, want gate_evidence", rec.Reference.DataClass)
	}
}

func TestPutGateEvidenceRejectsMalformedPayloadFailClosed(t *testing.T) {
	store := newTestStore(t)
	_, err := store.PutGateEvidence("run-evidence", GateEvidenceArtifact{
		SchemaID:      "runecode.protocol.v0.GateEvidence",
		SchemaVersion: "0.1.0",
		GateID:        "policy_gate",
		GateKind:      "policy",
		GateVersion:   "1.0.0",
		RunID:         "run-evidence",
		GateAttemptID: "gate-attempt-1",
		StartedAt:     time.Date(2026, 4, 2, 11, 0, 0, 0, time.UTC).Format(time.RFC3339),
		FinishedAt:    time.Date(2026, 4, 2, 11, 2, 0, 0, time.UTC).Format(time.RFC3339),
		Runtime:       map[string]any{"tool": "policyengine"},
		Outcome:       map[string]any{},
	})
	if err == nil {
		t.Fatal("PutGateEvidence error = nil, want fail-closed validation failure")
	}
}

func TestPutGateEvidenceRejectsFinishedBeforeStarted(t *testing.T) {
	store := newTestStore(t)
	_, err := store.PutGateEvidence("run-evidence", GateEvidenceArtifact{
		SchemaID:      "runecode.protocol.v0.GateEvidence",
		SchemaVersion: "0.1.0",
		GateID:        "policy_gate",
		GateKind:      "policy",
		GateVersion:   "1.0.0",
		RunID:         "run-evidence",
		GateAttemptID: "gate-attempt-2",
		StartedAt:     time.Date(2026, 4, 2, 11, 2, 0, 0, time.UTC).Format(time.RFC3339),
		FinishedAt:    time.Date(2026, 4, 2, 11, 1, 0, 0, time.UTC).Format(time.RFC3339),
		Runtime:       map[string]any{"tool": "policyengine"},
		Outcome:       map[string]any{"deterministic_outcome": "failed"},
	})
	if err == nil {
		t.Fatal("PutGateEvidence error = nil, want finished_at ordering validation failure")
	}
}

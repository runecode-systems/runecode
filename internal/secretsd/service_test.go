package secretsd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLeaseLifecycleAndBinding(t *testing.T) {
	svc, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if _, err := svc.ImportSecret("secrets/prod/db", strings.NewReader("db-secret")); err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
	lease, err := svc.IssueLease(IssueLeaseRequest{SecretRef: "secrets/prod/db", ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", TTLSeconds: 120})
	if err != nil {
		t.Fatalf("IssueLease returned error: %v", err)
	}
	if _, _, err := svc.Retrieve(RetrieveRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:other", RoleKind: "runner", Scope: "stage:alpha"}); err == nil {
		t.Fatal("Retrieve expected deny for consumer mismatch")
	}
	if _, _, err := svc.Retrieve(RetrieveRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha"}); err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if _, err := svc.RenewLease(RenewLeaseRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", TTLSeconds: 180}); err != nil {
		t.Fatalf("RenewLease returned error: %v", err)
	}
	if _, err := svc.RevokeLease(RevokeLeaseRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha", Reason: "operator"}); err != nil {
		t.Fatalf("RevokeLease returned error: %v", err)
	}
	if _, _, err := svc.Retrieve(RetrieveRequest{LeaseID: lease.LeaseID, ConsumerID: "principal:runner:1", RoleKind: "runner", Scope: "stage:alpha"}); err == nil {
		t.Fatal("Retrieve expected deny after revoke")
	}
}

func TestRecoveryFailClosedWithCorruptState(t *testing.T) {
	root := t.TempDir()
	svc, err := Open(root)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if _, err := svc.ImportSecret("secrets/prod/api", strings.NewReader("api-secret")); err != nil {
		t.Fatalf("ImportSecret returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, stateFileName), []byte("{broken"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if _, err := Open(root); err == nil {
		t.Fatal("Open expected fail-closed recovery error")
	} else if !errors.Is(err, ErrStateRecoveryFailed) {
		t.Fatalf("Open error = %v, want ErrStateRecoveryFailed", err)
	}
}

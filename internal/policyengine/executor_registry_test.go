package policyengine

import (
	"reflect"
	"testing"
)

func TestWorkspaceExecutorContractByIDKnownAndUnknown(t *testing.T) {
	contract, ok := workspaceExecutorContractByID("workspace-runner")
	if !ok {
		t.Fatal("workspace-runner contract not found")
	}
	if contract.ID != "workspace-runner" {
		t.Fatalf("contract.ID = %q, want %q", contract.ID, "workspace-runner")
	}
	if contract.AllowedClass != "workspace_ordinary" {
		t.Fatalf("AllowedClass = %q, want %q", contract.AllowedClass, "workspace_ordinary")
	}
	if _, ok := contract.AllowedRoles["workspace-edit"]; !ok {
		t.Fatal("workspace-edit role not allowed for workspace-runner")
	}
	if _, ok := contract.AllowedRoles["workspace-test"]; !ok {
		t.Fatal("workspace-test role not allowed for workspace-runner")
	}

	if _, ok := workspaceExecutorContractByID("unknown-executor"); ok {
		t.Fatal("unknown executor unexpectedly resolved")
	}
}

func TestBuildExecutorRegistryProjectionDeterministicAndReadOnly(t *testing.T) {
	first := BuildExecutorRegistryProjection()
	if first.Version != "trusted-v1" {
		t.Fatalf("Version = %q, want %q", first.Version, "trusted-v1")
	}
	if len(first.Executors) != 2 {
		t.Fatalf("len(Executors) = %d, want 2", len(first.Executors))
	}
	if first.Executors[0].ExecutorID != "python" || first.Executors[1].ExecutorID != "workspace-runner" {
		t.Fatalf("executor ordering = %#v, want python then workspace-runner", first.Executors)
	}

	first.Executors[0].AllowedRoles[0] = "mutated-role"
	first.Executors[0].ExecutorClass = "mutated-class"

	second := BuildExecutorRegistryProjection()
	if second.Executors[0].ExecutorClass == "mutated-class" {
		t.Fatal("projection re-used mutable executor_class data")
	}
	if reflect.DeepEqual(first, second) {
		t.Fatal("projection mutation leaked into authoritative source")
	}
	if !reflect.DeepEqual(second.Executors[0].AllowedRoles, []string{"workspace-edit", "workspace-test"}) {
		t.Fatalf("python roles = %#v, want workspace-edit/workspace-test", second.Executors[0].AllowedRoles)
	}
}

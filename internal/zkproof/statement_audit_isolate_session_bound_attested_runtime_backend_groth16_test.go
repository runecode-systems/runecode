package zkproof

import (
	"errors"
	"testing"
)

func TestTrustedLocalGroth16BackendV0FailsClosedUntilReviewedSetupAssetsExist(t *testing.T) {
	_, _, _, _, err := NewTrustedLocalGroth16BackendV0()
	if err == nil {
		t.Fatal("NewTrustedLocalGroth16BackendV0 expected fail-closed error")
	}
	var feasibility *FeasibilityError
	if !errors.As(err, &feasibility) {
		t.Fatalf("error type = %T, want *FeasibilityError", err)
	}
	if feasibility.Code != feasibilityCodeUnconfiguredProofBackend {
		t.Fatalf("feasibility code = %q, want %q", feasibility.Code, feasibilityCodeUnconfiguredProofBackend)
	}
}

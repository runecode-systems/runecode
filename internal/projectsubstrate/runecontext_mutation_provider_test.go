package projectsubstrate

import (
	"strings"
	"testing"
)

func TestRenderAssuranceBaselineQuotesSourcePosture(t *testing.T) {
	provider := newBundledRuneContextMutationProvider()

	baseline, err := provider.RenderAssuranceBaseline("embedded", "")
	if err != nil {
		t.Fatalf("RenderAssuranceBaseline returned error: %v", err)
	}

	if !strings.Contains(baseline, "source_posture: \"embedded\"") {
		t.Fatalf("baseline source_posture = %q, want quoted YAML scalar", baseline)
	}
}

func TestRenderAssuranceBaselineSafelyEscapesSourcePosture(t *testing.T) {
	provider := newBundledRuneContextMutationProvider()

	baseline, err := provider.RenderAssuranceBaseline("embedded #comment", "")
	if err != nil {
		t.Fatalf("RenderAssuranceBaseline returned error: %v", err)
	}

	if !strings.Contains(baseline, "source_posture: \"embedded #comment\"") {
		t.Fatalf("baseline source_posture = %q, want safely quoted YAML scalar", baseline)
	}
}

package brokerapi

import (
	"testing"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func TestBuildAnchorSegmentResponsePropagatesFailureDetail(t *testing.T) {
	result := auditd.AnchorSegmentResult{
		SealDigest: trustpolicy.Digest{
			HashAlg: "sha256",
			Hash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		AnchorStatus:         "degraded",
		FailureReasonCode:    "external_anchor_deferred_or_unavailable",
		FailureReasonMessage: "external anchor confirmation is deferred",
	}
	resp, errResp := buildAnchorSegmentResponse("req-anchor-propagate", result, "sha256:context")
	if errResp != nil {
		t.Fatalf("buildAnchorSegmentResponse returned error response: %+v", errResp)
	}
	if resp.FailureCode != "external_anchor_deferred_or_unavailable" {
		t.Fatalf("failure_code = %q, want external_anchor_deferred_or_unavailable", resp.FailureCode)
	}
	if resp.FailureMessage != "external anchor confirmation is deferred" {
		t.Fatalf("failure_message = %q, want external anchor confirmation is deferred", resp.FailureMessage)
	}
}

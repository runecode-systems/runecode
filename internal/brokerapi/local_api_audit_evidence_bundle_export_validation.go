package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

func canonicalDigest(v any) (trustpolicy.Digest, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return trustpolicy.Digest{}, err
	}
	sum := sha256.Sum256(canonical)
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(sum[:])}, nil
}

func validateAuditEvidenceBundleExportSemantics(events []AuditEvidenceBundleExportEvent) error {
	if len(events) < 2 {
		return fmt.Errorf("audit evidence bundle export must include start and terminal events")
	}
	if events[0].EventType != "audit_evidence_bundle_export_start" {
		return fmt.Errorf("audit evidence bundle export first event must be start")
	}
	if !events[len(events)-1].Terminal {
		return fmt.Errorf("audit evidence bundle export missing terminal event")
	}
	for i := range events {
		if i == 0 {
			continue
		}
		if events[i].Seq <= events[i-1].Seq {
			return fmt.Errorf("audit evidence bundle export sequence must increase monotonically")
		}
	}
	return nil
}

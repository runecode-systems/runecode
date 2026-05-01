package brokerapi

import (
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	externalAnchorRequestKindSubmitV0                  = "external_anchor_submit_v0"
	externalAnchorTargetKindTransparencyLog            = "transparency_log"
	externalAnchorRuntimeAdapterTransparencyLogV0      = "transparency_log_v0"
	externalAnchorReceiptKindExternalTransparencyLogV0 = "external_transparency_log_v0"
	externalAnchorProofKindTransparencyLogReceiptV0    = "transparency_log_receipt_v0"
	externalAnchorProofSchemaTransparencyLogReceiptV0  = "runecode.protocol.audit.anchor_proof.transparency_log_receipt.v0"
)

type externalAnchorResolvedTarget struct {
	TargetKind               string
	TargetRequirement        string
	TargetDescriptor         map[string]any
	TargetDescriptorDigest   trustpolicy.Digest
	TargetDescriptorIdentity string
	RuntimeAdapter           string
	ReceiptKind              string
	ProofKind                string
	ProofSchemaID            string
}

type externalAnchorImplementedAdapterProfile struct {
	receiptKind    string
	runtimeAdapter string
	proofKind      string
	proofSchemaID  string
}

func externalAnchorImplementedProfile(targetKind string) (externalAnchorImplementedAdapterProfile, error) {
	if strings.TrimSpace(targetKind) != externalAnchorTargetKindTransparencyLog {
		return externalAnchorImplementedAdapterProfile{}, fmt.Errorf("typed_request.target_kind %q is not implemented for broker execution", strings.TrimSpace(targetKind))
	}
	return externalAnchorImplementedAdapterProfile{
		receiptKind:    externalAnchorReceiptKindExternalTransparencyLogV0,
		runtimeAdapter: externalAnchorRuntimeAdapterTransparencyLogV0,
		proofKind:      externalAnchorProofKindTransparencyLogReceiptV0,
		proofSchemaID:  externalAnchorProofSchemaTransparencyLogReceiptV0,
	}, nil
}

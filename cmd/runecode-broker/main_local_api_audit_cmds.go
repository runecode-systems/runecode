package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func handleAuditRecordGet(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("audit-record-get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	recordDigest := fs.String("record-digest", "", "audit record digest")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "audit-record-get usage: runecode-broker audit-record-get --record-digest sha256:..."}
	}
	digest, err := parseDigestFlag(*recordDigest, "--record-digest")
	if err != nil {
		return &usageError{message: "audit-record-get " + err.Error()}
	}

	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.AuditRecordGet(ctx, brokerapi.AuditRecordGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditRecordGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		RecordDigest:  digest,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp.Record)
}

func handleAuditAnchorSegment(args []string, service *brokerapi.Service, stdout io.Writer) error {
	input, err := parseAuditAnchorSegmentCLIInput(args)
	if err != nil {
		return err
	}

	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	presenceResp, errResp := api.AuditAnchorPresenceGet(ctx, brokerapi.AuditAnchorPresenceGetRequest{
		SchemaID:      "runecode.protocol.v0.AuditAnchorPresenceGetRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		SealDigest:    input.sealDigest,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	if err := validateAuditAnchorPresenceResponse(presenceResp); err != nil {
		return err
	}
	resp, errResp := api.AuditAnchorSegment(ctx, brokerapi.AuditAnchorSegmentRequest{
		SchemaID:               "runecode.protocol.v0.AuditAnchorSegmentRequest",
		SchemaVersion:          "0.1.0",
		RequestID:              defaultRequestID(),
		SealDigest:             input.sealDigest,
		ApprovalDecisionDigest: input.approvalDecisionDigest,
		ApprovalAssuranceLevel: input.approvalAssuranceLevel,
		PresenceAttestation:    presenceResp.PresenceAttestation,
		ExportReceiptCopy:      input.exportReceiptCopy,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	if strings.TrimSpace(resp.AnchoringStatus) != "ok" {
		return fmt.Errorf("audit anchor failed: %s", auditAnchorFailureReason(resp))
	}
	return writeJSON(stdout, resp)
}

type auditAnchorSegmentCLIInput struct {
	sealDigest             trustpolicy.Digest
	approvalDecisionDigest *trustpolicy.Digest
	approvalAssuranceLevel string
	exportReceiptCopy      bool
}

func parseAuditAnchorSegmentCLIInput(args []string) (auditAnchorSegmentCLIInput, error) {
	fs := flag.NewFlagSet("audit-anchor-segment", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	sealDigestInput := fs.String("seal-digest", "", "segment seal digest")
	exportReceiptCopy := fs.Bool("export-receipt-copy", false, "optionally export a review receipt copy")
	approvalDecisionDigestInput := fs.String("approval-decision-digest", "", "optional consumed approval decision digest")
	approvalAssuranceLevel := fs.String("approval-assurance-level", "", "optional approval assurance level (requires --approval-decision-digest)")
	if err := fs.Parse(args); err != nil {
		return auditAnchorSegmentCLIInput{}, &usageError{message: "audit-anchor-segment usage: runecode-broker audit-anchor-segment --seal-digest sha256:... [--approval-decision-digest sha256:...] [--approval-assurance-level <level>] [--export-receipt-copy]"}
	}
	sealDigest, err := parseDigestFlag(*sealDigestInput, "--seal-digest")
	if err != nil {
		return auditAnchorSegmentCLIInput{}, &usageError{message: "audit-anchor-segment " + err.Error()}
	}
	approvalDecisionDigest, parseErr := parseOptionalDigestFlag(*approvalDecisionDigestInput, "--approval-decision-digest")
	if parseErr != nil {
		return auditAnchorSegmentCLIInput{}, &usageError{message: "audit-anchor-segment " + parseErr.Error()}
	}
	if strings.TrimSpace(*approvalAssuranceLevel) != "" && approvalDecisionDigest == nil {
		return auditAnchorSegmentCLIInput{}, &usageError{message: "audit-anchor-segment --approval-assurance-level requires --approval-decision-digest"}
	}
	return auditAnchorSegmentCLIInput{
		sealDigest:             sealDigest,
		approvalDecisionDigest: approvalDecisionDigest,
		approvalAssuranceLevel: strings.TrimSpace(*approvalAssuranceLevel),
		exportReceiptCopy:      *exportReceiptCopy,
	}, nil
}

func validateAuditAnchorPresenceResponse(resp brokerapi.AuditAnchorPresenceGetResponse) error {
	mode := strings.TrimSpace(resp.PresenceMode)
	if mode == "" {
		return fmt.Errorf("audit anchor failed: presence mode unavailable")
	}
	if (mode == "os_confirmation" || mode == "hardware_touch") && resp.PresenceAttestation == nil {
		return fmt.Errorf("audit anchor failed: presence attestation unavailable")
	}
	return nil
}

func auditAnchorFailureReason(resp brokerapi.AuditAnchorSegmentResponse) string {
	reason := strings.TrimSpace(resp.FailureCode)
	if reason != "" {
		return reason
	}
	reason = strings.TrimSpace(resp.FailureMessage)
	if reason != "" {
		return reason
	}
	return "anchor action failed"
}

func parseDigestFlag(value string, flagName string) (trustpolicy.Digest, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return trustpolicy.Digest{}, fmt.Errorf("requires %s", flagName)
	}
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		return trustpolicy.Digest{}, fmt.Errorf("%s must use sha256:<64 lowercase hex>", flagName)
	}
	digest := trustpolicy.Digest{HashAlg: parts[0], Hash: parts[1]}
	if _, err := digest.Identity(); err != nil {
		return trustpolicy.Digest{}, fmt.Errorf("%s invalid: %v", flagName, err)
	}
	return digest, nil
}

func parseOptionalDigestFlag(value string, flagName string) (*trustpolicy.Digest, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	digest, err := parseDigestFlag(value, flagName)
	if err != nil {
		return nil, err
	}
	return &digest, nil
}

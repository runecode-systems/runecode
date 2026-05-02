package brokerapi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) HandleAuditEvidenceBundleOfflineVerify(ctx context.Context, req AuditEvidenceBundleOfflineVerifyRequest, meta RequestContext) (AuditEvidenceBundleOfflineVerifyResponse, *ErrorResponse) {
	requestID, _, cleanup, errResp := s.prepareAuditEvidenceRequest(ctx, req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditEvidenceBundleOfflineVerifyRequestSchemaPath, meta, "audit evidence bundle offline verify service unavailable")
	if errResp != nil {
		return AuditEvidenceBundleOfflineVerifyResponse{}, errResp
	}
	defer cleanup()
	if errResp := s.requireAuditEvidenceLedger(requestID); errResp != nil {
		return AuditEvidenceBundleOfflineVerifyResponse{}, errResp
	}
	verification, errResp := s.verifyAuditEvidenceBundleFromRequest(requestID, req)
	if errResp != nil {
		return AuditEvidenceBundleOfflineVerifyResponse{}, errResp
	}
	resp := AuditEvidenceBundleOfflineVerifyResponse{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleOfflineVerifyResponse",
		SchemaVersion: "0.1.0",
		RequestID:     requestID,
		Verification:  verification,
	}
	if err := s.validateResponse(resp, auditEvidenceBundleOfflineVerifyResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditEvidenceBundleOfflineVerifyResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) verifyAuditEvidenceBundleFromRequest(requestID string, req AuditEvidenceBundleOfflineVerifyRequest) (AuditEvidenceBundleOfflineVerification, *ErrorResponse) {
	bundlePath := strings.TrimSpace(req.BundlePath)
	if bundlePath == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "bundle_path is required")
		return AuditEvidenceBundleOfflineVerification{}, &errOut
	}
	cleanBundlePath, err := validatedOfflineBundlePath(bundlePath)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return AuditEvidenceBundleOfflineVerification{}, &errOut
	}
	f, err := os.Open(cleanBundlePath)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("bundle open failed: %v", err))
		return AuditEvidenceBundleOfflineVerification{}, &errOut
	}
	defer f.Close()
	trustedVerification, err := s.auditLedger.OfflineVerifyEvidenceBundle(f, req.ArchiveFormat)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, fmt.Sprintf("audit evidence bundle offline verify failed: %v", err))
		return AuditEvidenceBundleOfflineVerification{}, &errOut
	}
	verification, err := projectAuditEvidenceBundleOfflineVerification(trustedVerification)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence bundle offline verify projection failed")
		return AuditEvidenceBundleOfflineVerification{}, &errOut
	}
	if err := s.validateResponse(verification, auditEvidenceBundleOfflineVerificationSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditEvidenceBundleOfflineVerification{}, &errOut
	}
	return verification, nil
}

func validatedOfflineBundlePath(bundlePath string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(bundlePath))
	if clean == "." || clean == "" {
		return "", fmt.Errorf("bundle_path is required")
	}
	if !filepath.IsAbs(clean) {
		return "", fmt.Errorf("bundle_path must be an absolute path")
	}
	if err := rejectLinkedPathComponents(filepath.Dir(clean)); err != nil {
		if errors.Is(err, errLinkedPathComponent) {
			return "", fmt.Errorf("bundle_path must not contain symlink components")
		}
		return "", err
	}
	info, err := os.Stat(clean)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("bundle_path must reference a file")
	}
	if filepath.Ext(clean) != ".tar" {
		return "", fmt.Errorf("bundle_path must reference a .tar archive")
	}
	return clean, nil
}

func projectAuditEvidenceBundleOfflineVerification(value auditd.AuditEvidenceBundleOfflineVerification) (AuditEvidenceBundleOfflineVerification, error) {
	manifestDigest, err := optionalDigestFromAuditIdentity(value.ManifestDigest)
	if err != nil {
		return AuditEvidenceBundleOfflineVerification{}, err
	}
	trustRoots, err := projectAuditSnapshotDigests(value.TrustRootDigests)
	if err != nil {
		return AuditEvidenceBundleOfflineVerification{}, err
	}
	findings, err := projectAuditEvidenceBundleOfflineFindings(value.Findings)
	if err != nil {
		return AuditEvidenceBundleOfflineVerification{}, err
	}
	reports, err := projectAuditEvidenceBundleOfflineReportPostures(value.VerificationReports)
	if err != nil {
		return AuditEvidenceBundleOfflineVerification{}, err
	}
	scope, err := projectAuditEvidenceBundleScope(value.Scope)
	if err != nil {
		return AuditEvidenceBundleOfflineVerification{}, err
	}
	return AuditEvidenceBundleOfflineVerification{
		SchemaID:            "runecode.protocol.v0.AuditEvidenceBundleOfflineVerification",
		SchemaVersion:       "0.1.0",
		VerifiedAt:          value.VerifiedAt,
		ArchiveFormat:       value.ArchiveFormat,
		ManifestDigest:      manifestDigest,
		BundleID:            value.BundleID,
		ExportProfile:       value.ExportProfile,
		Scope:               scope,
		VerifierIdentity:    projectAuditEvidenceBundleVerifierIdentity(value.VerifierIdentity),
		TrustRootDigests:    trustRoots,
		VerificationStatus:  value.VerificationStatus,
		Findings:            findings,
		VerificationReports: reports,
	}, nil
}

func optionalDigestFromAuditIdentity(identity string) (*trustpolicy.Digest, error) {
	if strings.TrimSpace(identity) == "" {
		return nil, nil
	}
	d, err := digestFromIdentity(identity)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func projectAuditEvidenceBundleOfflineFindings(values []auditd.AuditEvidenceBundleOfflineFinding) ([]AuditEvidenceBundleOfflineFinding, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]AuditEvidenceBundleOfflineFinding, 0, len(values))
	for i := range values {
		d, err := optionalDigestFromAuditIdentity(values[i].Digest)
		if err != nil {
			return nil, err
		}
		out = append(out, AuditEvidenceBundleOfflineFinding{Code: values[i].Code, Severity: values[i].Severity, Message: values[i].Message, ObjectPath: values[i].ObjectPath, Digest: d})
	}
	return out, nil
}

func projectAuditEvidenceBundleOfflineReportPostures(values []auditd.AuditEvidenceBundleOfflineReportPosture) ([]AuditEvidenceBundleOfflineReportPosture, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]AuditEvidenceBundleOfflineReportPosture, 0, len(values))
	for i := range values {
		d, err := digestFromIdentity(values[i].Digest)
		if err != nil {
			return nil, err
		}
		out = append(out, AuditEvidenceBundleOfflineReportPosture{
			Digest:                 d,
			IntegrityStatus:        values[i].IntegrityStatus,
			AnchoringStatus:        values[i].AnchoringStatus,
			StoragePostureStatus:   values[i].StoragePostureStatus,
			SegmentLifecycleStatus: values[i].SegmentLifecycleStatus,
			CurrentlyDegraded:      values[i].CurrentlyDegraded,
			DegradedReasons:        values[i].DegradedReasons,
			HardFailures:           values[i].HardFailures,
		})
	}
	return out, nil
}

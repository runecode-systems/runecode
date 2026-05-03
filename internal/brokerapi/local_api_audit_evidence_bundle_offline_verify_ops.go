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

type offlineBundlePathError struct {
	err error
}

func (e *offlineBundlePathError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *offlineBundlePathError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func newOfflineBundlePathError(format string, args ...any) error {
	return &offlineBundlePathError{err: fmt.Errorf(format, args...)}
}

func isOfflineBundlePathError(err error) bool {
	var target *offlineBundlePathError
	return errors.As(err, &target)
}

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
	s.persistMetaAuditReceipt(auditReceiptKindSensitiveEvidenceView, "audit_evidence_bundle_offline_verify", nil, resp.Verification.ManifestDigest, resp.Verification.ManifestDigest, "offline_verification")
	return resp, nil
}

func (s *Service) verifyAuditEvidenceBundleFromRequest(requestID string, req AuditEvidenceBundleOfflineVerifyRequest) (AuditEvidenceBundleOfflineVerification, *ErrorResponse) {
	bundlePath := strings.TrimSpace(req.BundlePath)
	if bundlePath == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "bundle_path is required")
		return AuditEvidenceBundleOfflineVerification{}, &errOut
	}
	f, err := openValidatedOfflineBundleFile(bundlePath)
	if err != nil {
		code, category := "gateway_failure", "internal"
		if isOfflineBundlePathError(err) {
			code, category = "broker_validation_schema_invalid", "validation"
		}
		errOut := s.makeError(requestID, code, category, false, err.Error())
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

func openValidatedOfflineBundleFile(bundlePath string) (*os.File, error) {
	return openValidatedOfflineBundleFileWithOpener(bundlePath, os.Open)
}

func openValidatedOfflineBundleFileWithOpener(bundlePath string, opener func(string) (*os.File, error)) (*os.File, error) {
	clean := filepath.Clean(strings.TrimSpace(bundlePath))
	if clean == "." || clean == "" {
		return nil, newOfflineBundlePathError("bundle_path is required")
	}
	if !filepath.IsAbs(clean) {
		return nil, newOfflineBundlePathError("bundle_path must be an absolute path")
	}
	if filepath.Ext(clean) != ".tar" {
		return nil, newOfflineBundlePathError("bundle_path must reference a .tar archive")
	}
	if err := validateOfflineBundleParentPath(clean); err != nil {
		return nil, err
	}
	preOpenInfo, err := validatedOfflineBundleLeafInfo(clean)
	if err != nil {
		return nil, err
	}
	f, err := opener(clean)
	if err != nil {
		return nil, err
	}
	openedInfo, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if openedInfo.IsDir() {
		_ = f.Close()
		return nil, newOfflineBundlePathError("bundle_path must reference a file")
	}
	if !os.SameFile(preOpenInfo, openedInfo) {
		_ = f.Close()
		return nil, newOfflineBundlePathError("bundle_path changed while opening")
	}
	return f, nil
}

func validateOfflineBundleParentPath(clean string) error {
	if err := rejectLinkedPathComponents(filepath.Dir(clean)); err != nil {
		if errors.Is(err, errLinkedPathComponent) {
			return newOfflineBundlePathError("bundle_path must not contain symlink components")
		}
		return err
	}
	return nil
}

func validatedOfflineBundleLeafInfo(clean string) (os.FileInfo, error) {
	lstatInfo, err := os.Lstat(clean)
	if err != nil {
		return nil, err
	}
	linked, err := pathEntryIsLinkOrReparse(clean, lstatInfo)
	if err != nil {
		return nil, err
	}
	if linked {
		return nil, newOfflineBundlePathError("bundle_path must not reference a symlink")
	}
	return lstatInfo, nil
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

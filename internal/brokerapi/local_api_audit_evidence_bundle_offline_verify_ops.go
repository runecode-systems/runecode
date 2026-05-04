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

var (
	errOfflineBundlePathRequired           = errors.New("bundle_path is required")
	errOfflineBundlePathAbsolute           = errors.New("bundle_path must be an absolute path")
	errOfflineBundlePathLinkedComponents   = errors.New("bundle_path must not contain symlink components")
	errOfflineBundlePathSymlink            = errors.New("bundle_path must not reference a symlink")
	errOfflineBundlePathFile               = errors.New("bundle_path must reference a file")
	errOfflineBundlePathTar                = errors.New("bundle_path must reference a .tar archive")
	errOfflineBundlePathNotAccessible      = errors.New("bundle_path is not accessible")
	errOfflineBundlePathAccess             = errors.New("bundle path access failed")
	errOfflineBundleVerificationFailedSafe = errors.New("audit evidence bundle offline verify failed")
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
	f, err := openValidatedOfflineBundleFile(bundlePath)
	if err != nil {
		if msg, ok := offlineBundleValidationClientMessage(err); ok {
			errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, msg)
			return AuditEvidenceBundleOfflineVerification{}, &errOut
		}
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "bundle open failed")
		return AuditEvidenceBundleOfflineVerification{}, &errOut
	}
	defer f.Close()
	trustedVerification, err := s.auditLedger.OfflineVerifyEvidenceBundle(f, req.ArchiveFormat)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, errOfflineBundleVerificationFailedSafe.Error())
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
	clean := filepath.Clean(strings.TrimSpace(bundlePath))
	if clean == "." || clean == "" {
		return nil, errOfflineBundlePathRequired
	}
	if !filepath.IsAbs(clean) {
		return nil, errOfflineBundlePathAbsolute
	}
	if filepath.Ext(clean) != ".tar" {
		return nil, errOfflineBundlePathTar
	}
	if err := rejectLinkedPathComponents(filepath.Dir(clean)); err != nil {
		if errors.Is(err, errLinkedPathComponent) {
			return nil, errOfflineBundlePathLinkedComponents
		}
		return nil, fmt.Errorf("%w: %v", errOfflineBundlePathNotAccessible, err)
	}
	info, err := os.Lstat(clean)
	if err != nil {
		return nil, fmt.Errorf("%w", errOfflineBundlePathNotAccessible)
	}
	linked, err := pathEntryIsLinkOrReparse(clean, info)
	if err != nil {
		return nil, fmt.Errorf("%w", errOfflineBundlePathNotAccessible)
	}
	if linked {
		return nil, errOfflineBundlePathSymlink
	}
	if info.IsDir() {
		return nil, errOfflineBundlePathFile
	}
	f, err := openReadOnlyNoFollow(clean)
	if err != nil {
		return nil, fmt.Errorf("%w", errOfflineBundlePathNotAccessible)
	}
	openedInfo, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("%w", errOfflineBundlePathAccess)
	}
	if !openedInfo.Mode().IsRegular() {
		_ = f.Close()
		return nil, errOfflineBundlePathFile
	}
	if !os.SameFile(info, openedInfo) {
		_ = f.Close()
		return nil, fmt.Errorf("%w", errOfflineBundlePathAccess)
	}
	return f, nil
}

func offlineBundleValidationClientMessage(err error) (string, bool) {
	switch {
	case errors.Is(err, errOfflineBundlePathRequired):
		return errOfflineBundlePathRequired.Error(), true
	case errors.Is(err, errOfflineBundlePathAbsolute):
		return errOfflineBundlePathAbsolute.Error(), true
	case errors.Is(err, errOfflineBundlePathLinkedComponents):
		return errOfflineBundlePathLinkedComponents.Error(), true
	case errors.Is(err, errOfflineBundlePathSymlink):
		return errOfflineBundlePathSymlink.Error(), true
	case errors.Is(err, errOfflineBundlePathFile):
		return errOfflineBundlePathFile.Error(), true
	case errors.Is(err, errOfflineBundlePathTar):
		return errOfflineBundlePathTar.Error(), true
	case errors.Is(err, errOfflineBundlePathNotAccessible):
		return errOfflineBundlePathNotAccessible.Error(), true
	default:
		return "", false
	}
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

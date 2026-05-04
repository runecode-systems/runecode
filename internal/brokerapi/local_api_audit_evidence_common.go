package brokerapi

import (
	"context"
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
)

func (s *Service) prepareAuditEvidenceRequest(ctx context.Context, reqID, fallbackReqID string, admissionErr error, req any, schemaPath string, meta RequestContext, unavailableMessage string) (string, context.Context, func(), *ErrorResponse) {
	requestID := resolveRequestID(reqID, fallbackReqID)
	if s == nil {
		errOut := toErrorResponse(requestID, "gateway_failure", "internal", false, unavailableMessage)
		return "", nil, nil, &errOut
	}
	requestID, errResp := s.prepareLocalRequest(reqID, fallbackReqID, admissionErr, req, schemaPath)
	if errResp != nil {
		return "", nil, nil, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return "", nil, nil, &errOut
	}
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	cleanup := func() {
		cancel()
		release()
	}
	if err := requestCtx.Err(); err != nil {
		cleanup()
		errOut := s.errorFromContext(requestID, err)
		return "", nil, nil, &errOut
	}
	return requestID, requestCtx, cleanup, nil
}

func (s *Service) requireAuditEvidenceLedger(requestID string) *ErrorResponse {
	if s.auditLedger != nil {
		return nil
	}
	errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit ledger unavailable")
	return &errOut
}

func (s *Service) auditEvidenceIdentityContext() auditd.AuditEvidenceIdentityContext {
	return auditd.AuditEvidenceIdentityContext{
		// TODO(verification-plane): Keep repository_identity_digest intentionally empty until we
		// have a canonical repository identity source that survives cross-machine relocation.
		// Hashing RepositoryRoot is path-derived and therefore machine-local: the same repository
		// cloned into different absolute paths (or mounted under different workspace roots)
		// produces different digests even when content/history are identical. That breaks
		// evidence identity continuity for external relying parties and creates false identity
		// churn across environments. We need a stable, path-independent canonical identity
		// (for example, a signed repository identity artifact anchored to immutable VCS identity
		// and/or an explicit trust-root-backed repo identity registry) before emitting this field.
		// Until that exists, leaving RepositoryIdentityDigest empty is safer than publishing a
		// misleading digest that cannot be compared reliably across machines.
		RepositoryIdentityDigest: "",
		ProductInstanceID:        strings.TrimSpace(s.productInstanceID),
	}
}

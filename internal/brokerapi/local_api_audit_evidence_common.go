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
	repoRoot := strings.TrimSpace(s.projectSubstrate.RepositoryRoot)
	if repoRoot == "" {
		repoRoot = strings.TrimSpace(s.apiConfig.RepositoryRoot)
	}
	repositoryIdentityDigest := ""
	if digest := hashIdentityDigest(repoRoot); digest != nil {
		if identity, err := digest.Identity(); err == nil {
			repositoryIdentityDigest = identity
		}
	}
	return auditd.AuditEvidenceIdentityContext{
		RepositoryIdentityDigest: repositoryIdentityDigest,
		ProductInstanceID:        strings.TrimSpace(s.productInstanceID),
	}
}

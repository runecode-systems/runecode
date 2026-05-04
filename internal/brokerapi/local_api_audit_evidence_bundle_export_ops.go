package brokerapi

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"github.com/runecode-ai/runecode/internal/auditd"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func (s *Service) HandleAuditEvidenceBundleExport(ctx context.Context, req AuditEvidenceBundleExportRequest, meta RequestContext) ([]AuditEvidenceBundleExportEvent, *ErrorResponse) {
	requestID, requestCtx, cleanup, errResp := s.prepareAuditEvidenceRequest(ctx, req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditEvidenceBundleExportRequestSchemaPath, meta, "audit evidence bundle export service unavailable")
	if errResp != nil {
		return nil, errResp
	}
	defer cleanup()
	if errResp := s.requireAuditEvidenceLedger(requestID); errResp != nil {
		return nil, errResp
	}
	manifest, manifestDigest, reader, errResp := s.buildAuditEvidenceBundleExportArtifacts(requestID, req)
	if errResp != nil {
		return nil, errResp
	}
	defer reader.Close()
	events, errResp := s.collectAuditEvidenceBundleExportEvents(requestCtx, requestID, manifest, manifestDigest, req.ArchiveFormat, reader)
	if errResp != nil {
		return nil, errResp
	}
	if auditEvidenceBundleExportCompleted(events) {
		s.persistMetaAuditReceipt(auditReceiptKindEvidenceBundleExport, "audit_evidence_bundle", manifestDigestRefOrNil(manifestDigest), nil, manifestDigestRefOrNil(manifestDigest), "")
	}
	return s.validateAuditEvidenceBundleExportEvents(requestID, events)
}

func auditEvidenceBundleExportCompleted(events []AuditEvidenceBundleExportEvent) bool {
	if len(events) == 0 {
		return false
	}
	last := events[len(events)-1]
	return last.Terminal && last.TerminalStatus == "completed" && last.EOF
}

func manifestDigestRefOrNil(digest trustpolicy.Digest) *trustpolicy.Digest {
	if identity, err := digest.Identity(); err != nil || strings.TrimSpace(identity) == "" {
		return nil
	}
	copyDigest := digest
	return &copyDigest
}

func (s *Service) buildAuditEvidenceBundleExportArtifacts(requestID string, req AuditEvidenceBundleExportRequest) (AuditEvidenceBundleManifest, trustpolicy.Digest, io.ReadCloser, *ErrorResponse) {
	trustedScope, err := projectAuditEvidenceBundleScopeToTrusted(req.Scope)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return AuditEvidenceBundleManifest{}, trustpolicy.Digest{}, nil, &errOut
	}
	trustedTool, err := projectAuditEvidenceBundleToolIdentityToTrusted(req.CreatedByTool)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return AuditEvidenceBundleManifest{}, trustpolicy.Digest{}, nil, &errOut
	}
	exportResult, err := s.auditLedger.ExportEvidenceBundle(auditd.AuditEvidenceBundleExportRequest{
		ManifestRequest: auditd.AuditEvidenceBundleManifestRequest{
			Scope:             trustedScope,
			ExportProfile:     req.ExportProfile,
			CreatedByTool:     trustedTool,
			IdentityContext:   s.auditEvidenceIdentityContext(),
			DisclosurePosture: projectAuditEvidenceBundleDisclosurePostureToTrusted(req.DisclosurePosture),
			Redactions:        projectAuditEvidenceBundleRedactionsToTrusted(req.Redactions),
		},
		ArchiveFormat: req.ArchiveFormat,
	})
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return AuditEvidenceBundleManifest{}, trustpolicy.Digest{}, nil, &errOut
	}
	manifest, err := projectAuditEvidenceBundleManifest(exportResult.Manifest)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence bundle manifest projection failed")
		_ = exportResult.Reader.Close()
		return AuditEvidenceBundleManifest{}, trustpolicy.Digest{}, nil, &errOut
	}
	manifestDigest, err := canonicalDigest(manifest)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence bundle manifest digest failed")
		_ = exportResult.Reader.Close()
		return AuditEvidenceBundleManifest{}, trustpolicy.Digest{}, nil, &errOut
	}
	return manifest, manifestDigest, exportResult.Reader, nil
}

func (s *Service) validateAuditEvidenceBundleExportEvents(requestID string, events []AuditEvidenceBundleExportEvent) ([]AuditEvidenceBundleExportEvent, *ErrorResponse) {
	for i := range events {
		if err := s.validateResponse(events[i], auditEvidenceBundleExportEventSchemaPath); err != nil {
			errOut := s.errorFromValidation(requestID, err)
			return nil, &errOut
		}
	}
	if err := validateAuditEvidenceBundleExportSemantics(events); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return nil, &errOut
	}
	return events, nil
}

func (s *Service) collectAuditEvidenceBundleExportEvents(requestCtx context.Context, requestID string, manifest AuditEvidenceBundleManifest, manifestDigest trustpolicy.Digest, archiveFormat string, reader io.ReadCloser) ([]AuditEvidenceBundleExportEvent, *ErrorResponse) {
	streamID := "audit-evidence-bundle-export-" + requestID
	format := strings.TrimSpace(archiveFormat)
	if format == "" {
		format = "tar"
	}
	events := []AuditEvidenceBundleExportEvent{auditEvidenceBundleExportStartEvent(streamID, requestID, format, manifest, manifestDigest)}
	chunkSize := s.apiConfig.Limits.MaxStreamChunkBytes
	if chunkSize <= 0 {
		chunkSize = 64 << 10
	}
	buffer := make([]byte, chunkSize)
	seq := int64(2)
	total := 0
	for {
		if terminal, done := auditEvidenceBundleExportContextTerminal(requestCtx, streamID, requestID, seq); done {
			events = append(events, terminal)
			return events, nil
		}
		var readErr error
		var terminal AuditEvidenceBundleExportEvent
		var done bool
		seq, total, readErr, events, terminal, done = s.appendAuditEvidenceBundleExportReadResult(events, requestID, streamID, seq, total, buffer, reader)
		if done {
			events = append(events, terminal)
			return events, nil
		}
		if terminal, done := s.auditEvidenceBundleExportReadTerminal(requestID, streamID, seq, readErr); done {
			events = append(events, terminal)
			return events, nil
		}
	}
}

func (s *Service) appendAuditEvidenceBundleExportReadResult(events []AuditEvidenceBundleExportEvent, requestID string, streamID string, seq int64, total int, buffer []byte, reader io.Reader) (int64, int, error, []AuditEvidenceBundleExportEvent, AuditEvidenceBundleExportEvent, bool) {
	n, readErr := reader.Read(buffer)
	if n == 0 {
		return seq, total, readErr, events, AuditEvidenceBundleExportEvent{}, false
	}
	total += n
	if terminal, done := s.auditEvidenceBundleExportSizeTerminal(requestID, streamID, seq, total); done {
		return seq, total, readErr, events, terminal, true
	}
	events = append(events, auditEvidenceBundleExportChunkEvent(streamID, requestID, seq, buffer[:n]))
	return seq + 1, total, readErr, events, AuditEvidenceBundleExportEvent{}, false
}

func auditEvidenceBundleExportStartEvent(streamID string, requestID string, archiveFormat string, manifest AuditEvidenceBundleManifest, manifestDigest trustpolicy.Digest) AuditEvidenceBundleExportEvent {
	return AuditEvidenceBundleExportEvent{
		SchemaID:       "runecode.protocol.v0.AuditEvidenceBundleExportEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       streamID,
		RequestID:      requestID,
		Seq:            1,
		EventType:      "audit_evidence_bundle_export_start",
		Manifest:       &manifest,
		ManifestDigest: &manifestDigest,
		ArchiveFormat:  archiveFormat,
	}
}

func auditEvidenceBundleExportChunkEvent(streamID string, requestID string, seq int64, chunk []byte) AuditEvidenceBundleExportEvent {
	copyChunk := append([]byte(nil), chunk...)
	return AuditEvidenceBundleExportEvent{
		SchemaID:      "runecode.protocol.v0.AuditEvidenceBundleExportEvent",
		SchemaVersion: "0.1.0",
		StreamID:      streamID,
		RequestID:     requestID,
		Seq:           seq,
		EventType:     "audit_evidence_bundle_export_chunk",
		ChunkBase64:   base64.StdEncoding.EncodeToString(copyChunk),
		ChunkBytes:    len(copyChunk),
	}
}

func auditEvidenceBundleExportCompletedEvent(streamID string, requestID string, seq int64) AuditEvidenceBundleExportEvent {
	return AuditEvidenceBundleExportEvent{
		SchemaID:       "runecode.protocol.v0.AuditEvidenceBundleExportEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       streamID,
		RequestID:      requestID,
		Seq:            seq,
		EventType:      "audit_evidence_bundle_export_terminal",
		Terminal:       true,
		TerminalStatus: "completed",
		EOF:            true,
	}
}

func auditEvidenceBundleExportContextTerminal(requestCtx context.Context, streamID string, requestID string, seq int64) (AuditEvidenceBundleExportEvent, bool) {
	if err := artifactReadContextErr(requestCtx); err != nil {
		return auditEvidenceBundleExportTerminalFromContext(streamID, requestID, seq, err), true
	}
	return AuditEvidenceBundleExportEvent{}, false
}

func (s *Service) auditEvidenceBundleExportSizeTerminal(requestID string, streamID string, seq int64, total int) (AuditEvidenceBundleExportEvent, bool) {
	if total <= s.apiConfig.Limits.MaxResponseStreamBytes {
		return AuditEvidenceBundleExportEvent{}, false
	}
	errOut := s.makeError(requestID, "broker_limit_response_stream_size_exceeded", "transport", false, "audit evidence bundle export exceeded broker max response stream bytes")
	return auditEvidenceBundleExportTerminalError(streamID, requestID, seq, "failed", &errOut.Error), true
}

func (s *Service) auditEvidenceBundleExportReadTerminal(requestID string, streamID string, seq int64, readErr error) (AuditEvidenceBundleExportEvent, bool) {
	if readErr == nil {
		return AuditEvidenceBundleExportEvent{}, false
	}
	if readErr == io.EOF {
		return auditEvidenceBundleExportCompletedEvent(streamID, requestID, seq), true
	}
	errOut := s.makeError(requestID, "gateway_failure", "internal", false, "audit evidence bundle export read failed")
	return auditEvidenceBundleExportTerminalError(streamID, requestID, seq, "failed", &errOut.Error), true
}

func auditEvidenceBundleExportTerminalError(streamID string, requestID string, seq int64, status string, err *ProtocolError) AuditEvidenceBundleExportEvent {
	return AuditEvidenceBundleExportEvent{
		SchemaID:       "runecode.protocol.v0.AuditEvidenceBundleExportEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       streamID,
		RequestID:      requestID,
		Seq:            seq,
		EventType:      "audit_evidence_bundle_export_terminal",
		Terminal:       true,
		TerminalStatus: status,
		Error:          err,
	}
}

func auditEvidenceBundleExportTerminalFromContext(streamID string, requestID string, seq int64, ctxErr error) AuditEvidenceBundleExportEvent {
	terminal := auditEvidenceBundleExportTerminalError(streamID, requestID, seq, "cancelled", nil)
	if errors.Is(ctxErr, context.DeadlineExceeded) {
		terminal.TerminalStatus = "failed"
		terminal.Error = &ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: errorEnvelopeSchemaVersion,
			Code:          "broker_timeout_request_deadline_exceeded",
			Category:      "timeout",
			Retryable:     false,
			Message:       "audit evidence bundle export deadline exceeded",
		}
		return terminal
	}
	if errors.Is(ctxErr, context.Canceled) {
		terminal.TerminalStatus = "cancelled"
		terminal.Error = nil
	}
	return terminal
}

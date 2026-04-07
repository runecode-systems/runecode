package brokerapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
	"github.com/runecode-ai/runecode/third_party/jsoncanonicalizer"
)

type approvalRecord struct {
	Summary          ApprovalSummary
	RequestEnvelope  *trustpolicy.SignedObjectEnvelope
	DecisionEnvelope *trustpolicy.SignedObjectEnvelope
}

type approvalState struct {
	mu      sync.Mutex
	seeded  bool
	records map[string]approvalRecord
}

func (s *Service) HandleRunList(ctx context.Context, req RunListRequest, meta RequestContext) (RunListResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, runListRequestSchemaPath)
	if errResp != nil {
		return RunListResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errResp := s.errorFromLimit(requestID, err)
		return RunListResponse{}, &errResp
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errResp := s.errorFromContext(requestID, err)
		return RunListResponse{}, &errResp
	}
	order := req.Order
	if order == "" {
		order = "updated_at_desc"
	}
	runs, err := s.runSummaries(order)
	if err != nil {
		errResp := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return RunListResponse{}, &errResp
	}
	limit := normalizeLimit(req.Limit, 50, 200)
	page, next, err := paginate(runs, req.Cursor, limit)
	if err != nil {
		errResp := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return RunListResponse{}, &errResp
	}
	resp := RunListResponse{SchemaID: "runecode.protocol.v0.RunListResponse", SchemaVersion: "0.1.0", RequestID: requestID, Order: order, Runs: page, NextCursor: next}
	if err := s.validateResponse(resp, runListResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return RunListResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleRunGet(ctx context.Context, req RunGetRequest, meta RequestContext) (RunGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, runGetRequestSchemaPath)
	if errResp != nil {
		return RunGetResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errResp := s.errorFromLimit(requestID, err)
		return RunGetResponse{}, &errResp
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errResp := s.errorFromContext(requestID, err)
		return RunGetResponse{}, &errResp
	}
	if strings.TrimSpace(req.RunID) == "" {
		errResp := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "run_id is required")
		return RunGetResponse{}, &errResp
	}
	detail, ok, err := s.runDetail(req.RunID)
	if err != nil {
		errResp := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return RunGetResponse{}, &errResp
	}
	if !ok {
		errResp := s.makeError(requestID, "broker_not_found_artifact", "storage", false, fmt.Sprintf("run %q not found", req.RunID))
		return RunGetResponse{}, &errResp
	}
	resp := RunGetResponse{SchemaID: "runecode.protocol.v0.RunGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Run: detail}
	if err := s.validateResponse(resp, runGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return RunGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleApprovalList(ctx context.Context, req ApprovalListRequest, meta RequestContext) (ApprovalListResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, approvalListRequestSchemaPath)
	if errResp != nil {
		return ApprovalListResponse{}, errResp
	}
	if err := s.seedStubApprovals(); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ApprovalListResponse{}, &errOut
	}
	order := req.Order
	if order == "" {
		order = "pending_first_newest_within_status"
	}
	records := s.listApprovals()
	filtered := make([]ApprovalSummary, 0, len(records))
	for _, rec := range records {
		if req.Status != "" && rec.Status != req.Status {
			continue
		}
		if req.RunID != "" && rec.BoundScope.RunID != req.RunID {
			continue
		}
		filtered = append(filtered, rec)
	}
	sortApprovals(filtered)
	limit := normalizeLimit(req.Limit, 50, 200)
	page, next, err := paginate(filtered, req.Cursor, limit)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return ApprovalListResponse{}, &errOut
	}
	resp := ApprovalListResponse{SchemaID: "runecode.protocol.v0.ApprovalListResponse", SchemaVersion: "0.1.0", RequestID: requestID, Order: order, Approvals: page, NextCursor: next}
	if err := s.validateResponse(resp, approvalListResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ApprovalListResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleApprovalGet(ctx context.Context, req ApprovalGetRequest, meta RequestContext) (ApprovalGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, approvalGetRequestSchemaPath)
	if errResp != nil {
		return ApprovalGetResponse{}, errResp
	}
	if err := s.seedStubApprovals(); err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ApprovalGetResponse{}, &errOut
	}
	rec, ok := s.getApproval(req.ApprovalID)
	if !ok {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, fmt.Sprintf("approval %q not found", req.ApprovalID))
		return ApprovalGetResponse{}, &errOut
	}
	resp := ApprovalGetResponse{SchemaID: "runecode.protocol.v0.ApprovalGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Approval: rec.Summary, SignedApprovalRequest: rec.RequestEnvelope, SignedApprovalDecision: rec.DecisionEnvelope}
	if err := s.validateResponse(resp, approvalGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ApprovalGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleApprovalResolve(ctx context.Context, req ApprovalResolveRequest, meta RequestContext) (ApprovalResolveResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, approvalResolveRequestSchemaPath)
	if errResp != nil {
		return ApprovalResolveResponse{}, errResp
	}
	release, err := s.acquireInFlight(meta)
	if err != nil {
		errOut := s.errorFromLimit(requestID, err)
		return ApprovalResolveResponse{}, &errOut
	}
	defer release()
	requestCtx, cancel := withRequestDeadline(ctx, meta, s.apiConfig.Limits.DefaultRequestDeadline)
	defer cancel()
	if err := requestCtx.Err(); err != nil {
		errOut := s.errorFromContext(requestID, err)
		return ApprovalResolveResponse{}, &errOut
	}
	approvalID, decisionDigest, errResp := s.resolveApprovalDigests(requestID, req)
	if errResp != nil {
		return ApprovalResolveResponse{}, errResp
	}
	head, errResp := s.promoteAndHeadResolvedArtifact(requestID, req)
	if errResp != nil {
		return ApprovalResolveResponse{}, errResp
	}
	record := buildResolvedApprovalRecord(req, approvalID, decisionDigest)
	s.putApproval(record)
	resp := buildApprovalResolveResponse(requestID, record, head)
	if err := s.validateResponse(resp, approvalResolveResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ApprovalResolveResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) resolveApprovalDigests(requestID string, req ApprovalResolveRequest) (string, string, *ErrorResponse) {
	approvalID, err := approvalIDFromRequest(req.SignedApprovalRequest)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", "", &errOut
	}
	if req.ApprovalID != "" && req.ApprovalID != approvalID {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, "approval_id does not match signed_approval_request")
		return "", "", &errOut
	}
	decisionDigest, err := signedEnvelopeDigest(req.SignedApprovalDecision)
	if err != nil {
		errOut := s.makeError(requestID, "broker_approval_state_invalid", "auth", false, err.Error())
		return "", "", &errOut
	}
	return approvalID, decisionDigest, nil
}

func (s *Service) promoteAndHeadResolvedArtifact(requestID string, req ApprovalResolveRequest) (artifacts.ArtifactRecord, *ErrorResponse) {
	ref, promoteErr := s.PromoteApprovedExcerpt(artifacts.PromotionRequest{
		UnapprovedDigest:      req.UnapprovedDigest,
		Approver:              req.Approver,
		ApprovalRequest:       &req.SignedApprovalRequest,
		ApprovalDecision:      &req.SignedApprovalDecision,
		RepoPath:              req.RepoPath,
		Commit:                req.Commit,
		ExtractorToolVersion:  req.ExtractorToolVersion,
		FullContentVisible:    req.FullContentVisible,
		ExplicitViewFull:      req.ExplicitViewFull,
		BulkRequest:           req.BulkRequest,
		BulkApprovalConfirmed: req.BulkApprovalConfirmed,
	})
	if promoteErr != nil {
		errOut := s.errorFromStore(requestID, promoteErr)
		return artifacts.ArtifactRecord{}, &errOut
	}
	head, err := s.Head(ref.Digest)
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return artifacts.ArtifactRecord{}, &errOut
	}
	return head, nil
}

func buildResolvedApprovalRecord(req ApprovalResolveRequest, approvalID, decisionDigest string) approvalRecord {
	now := time.Now().UTC().Format(time.RFC3339)
	return approvalRecord{
		Summary: ApprovalSummary{
			SchemaID:               "runecode.protocol.v0.ApprovalSummary",
			SchemaVersion:          "0.1.0",
			ApprovalID:             approvalID,
			Status:                 "approved",
			RequestedAt:            now,
			DecidedAt:              now,
			ApprovalTriggerCode:    "artifact_promotion",
			ChangesIfApproved:      "Promote reviewed file excerpts for downstream use.",
			ApprovalAssuranceLevel: decodeDecisionString(req.SignedApprovalDecision.Payload, "approval_assurance_level", "reauthenticated"),
			PresenceMode:           decodeDecisionString(req.SignedApprovalDecision.Payload, "presence_mode", "hardware_touch"),
			BoundScope:             req.BoundScope,
			RequestDigest:          approvalID,
			DecisionDigest:         decisionDigest,
		},
		RequestEnvelope:  &req.SignedApprovalRequest,
		DecisionEnvelope: &req.SignedApprovalDecision,
	}
}

func buildApprovalResolveResponse(requestID string, record approvalRecord, head artifacts.ArtifactRecord) ApprovalResolveResponse {
	return ApprovalResolveResponse{
		SchemaID:         "runecode.protocol.v0.ApprovalResolveResponse",
		SchemaVersion:    "0.1.0",
		RequestID:        requestID,
		ResolutionStatus: "resolved",
		Approval:         record.Summary,
		ApprovedArtifact: ptrArtifactSummary(toArtifactSummary(head)),
	}
}

func (s *Service) HandleArtifactListV0(ctx context.Context, req LocalArtifactListRequest, meta RequestContext) (LocalArtifactListResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, artifactListRequestSchemaPath)
	if errResp != nil {
		return LocalArtifactListResponse{}, errResp
	}
	order := req.Order
	if order == "" {
		order = "created_at_desc"
	}
	all := s.List()
	summaries := make([]ArtifactSummary, 0, len(all))
	for _, record := range all {
		summary := toArtifactSummary(record)
		if req.RunID != "" && summary.RunID != req.RunID {
			continue
		}
		if req.StepID != "" && summary.StepID != req.StepID {
			continue
		}
		if req.DataClass != "" && string(summary.Reference.DataClass) != req.DataClass {
			continue
		}
		summaries = append(summaries, summary)
	}
	sortArtifactSummariesNewestFirst(summaries)
	limit := normalizeLimit(req.Limit, 100, 500)
	page, next, err := paginate(summaries, req.Cursor, limit)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return LocalArtifactListResponse{}, &errOut
	}
	resp := LocalArtifactListResponse{SchemaID: "runecode.protocol.v0.ArtifactListResponse", SchemaVersion: "0.1.0", RequestID: requestID, Order: order, Artifacts: page, NextCursor: next}
	if err := s.validateResponse(resp, artifactListResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return LocalArtifactListResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleArtifactHeadV0(ctx context.Context, req LocalArtifactHeadRequest, meta RequestContext) (LocalArtifactHeadResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, artifactHeadRequestSchemaPath)
	if errResp != nil {
		return LocalArtifactHeadResponse{}, errResp
	}
	record, err := s.Head(req.Digest)
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return LocalArtifactHeadResponse{}, &errOut
	}
	resp := LocalArtifactHeadResponse{SchemaID: "runecode.protocol.v0.ArtifactHeadResponse", SchemaVersion: "0.1.0", RequestID: requestID, Artifact: toArtifactSummary(record)}
	if err := s.validateResponse(resp, artifactHeadResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return LocalArtifactHeadResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleArtifactRead(ctx context.Context, req ArtifactReadRequest, meta RequestContext) (ArtifactReadHandle, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, artifactReadRequestSchemaPath)
	if errResp != nil {
		return ArtifactReadHandle{}, errResp
	}
	if strings.TrimSpace(req.ProducerRole) == "" || strings.TrimSpace(req.ConsumerRole) == "" {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, "producer_role and consumer_role are required")
		return ArtifactReadHandle{}, &errOut
	}
	if req.RangeStart != nil || req.RangeEnd != nil {
		errOut := s.makeError(requestID, "broker_validation_range_not_supported", "validation", false, "range_start/range_end are not supported for MVP artifact reads")
		return ArtifactReadHandle{}, &errOut
	}
	if req.StreamID == "" {
		req.StreamID = "artifact-read-" + requestID
	}
	if req.ChunkBytes <= 0 || req.ChunkBytes > s.apiConfig.Limits.MaxStreamChunkBytes {
		req.ChunkBytes = s.apiConfig.Limits.MaxStreamChunkBytes
	}
	class := artifacts.DataClass(req.DataClass)
	r, record, err := s.GetForFlow(artifacts.ArtifactReadRequest{
		Digest:        req.Digest,
		ProducerRole:  req.ProducerRole,
		ConsumerRole:  req.ConsumerRole,
		DataClass:     class,
		IsEgress:      true,
		ManifestOptIn: req.ManifestOptIn,
	})
	if err != nil {
		errOut := s.errorFromStore(requestID, err)
		return ArtifactReadHandle{}, &errOut
	}
	return ArtifactReadHandle{RequestID: requestID, Digest: req.Digest, DataClass: record.Reference.DataClass, StreamID: req.StreamID, ChunkBytes: req.ChunkBytes, Reader: r}, nil
}

func (s *Service) StreamArtifactReadEvents(handle ArtifactReadHandle) ([]ArtifactStreamEvent, error) {
	if handle.Reader == nil {
		return nil, fmt.Errorf("artifact read handle reader is required")
	}
	if handle.StreamID == "" {
		return nil, fmt.Errorf("artifact read handle stream_id is required")
	}
	chunkSize := handle.ChunkBytes
	if chunkSize <= 0 || chunkSize > s.apiConfig.Limits.MaxStreamChunkBytes {
		chunkSize = s.apiConfig.Limits.MaxStreamChunkBytes
	}
	events, err := s.collectArtifactReadEvents(handle, chunkSize)
	if err != nil {
		return nil, err
	}
	if err := validateArtifactStreamSemantics(events); err != nil {
		return nil, err
	}
	for i := range events {
		if err := s.validateResponse(events[i], artifactStreamEventSchemaPath); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func (s *Service) collectArtifactReadEvents(handle ArtifactReadHandle, chunkSize int) ([]ArtifactStreamEvent, error) {
	buffer := make([]byte, chunkSize)
	events := []ArtifactStreamEvent{artifactStreamStartEvent(handle, 1)}
	seq := int64(2)
	total := 0
	for {
		n, readErr := handle.Reader.Read(buffer)
		if n > 0 {
			total += n
			if total > s.apiConfig.Limits.MaxResponseStreamBytes {
				events = append(events, artifactStreamTerminalLimitExceeded(handle, seq))
				_ = handle.Reader.Close()
				break
			}
			events = append(events, artifactStreamChunkEvent(handle, seq, buffer[:n]))
			seq++
		}
		if readErr == nil {
			continue
		}
		events = append(events, artifactStreamTerminalFromReadErr(handle, seq, readErr))
		_ = handle.Reader.Close()
		break
	}
	return events, nil
}

func artifactStreamStartEvent(handle ArtifactReadHandle, seq int64) ArtifactStreamEvent {
	return ArtifactStreamEvent{
		SchemaID:      "runecode.protocol.v0.ArtifactStreamEvent",
		SchemaVersion: "0.1.0",
		StreamID:      handle.StreamID,
		RequestID:     handle.RequestID,
		Seq:           seq,
		EventType:     "artifact_stream_start",
		Digest:        handle.Digest,
		DataClass:     string(handle.DataClass),
	}
}

func artifactStreamChunkEvent(handle ArtifactReadHandle, seq int64, chunk []byte) ArtifactStreamEvent {
	return ArtifactStreamEvent{
		SchemaID:      "runecode.protocol.v0.ArtifactStreamEvent",
		SchemaVersion: "0.1.0",
		StreamID:      handle.StreamID,
		RequestID:     handle.RequestID,
		Seq:           seq,
		EventType:     "artifact_stream_chunk",
		Digest:        handle.Digest,
		DataClass:     string(handle.DataClass),
		ChunkBase64:   base64.StdEncoding.EncodeToString(chunk),
		ChunkBytes:    len(chunk),
	}
}

func artifactStreamTerminalLimitExceeded(handle ArtifactReadHandle, seq int64) ArtifactStreamEvent {
	return ArtifactStreamEvent{
		SchemaID:       "runecode.protocol.v0.ArtifactStreamEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       handle.StreamID,
		RequestID:      handle.RequestID,
		Seq:            seq,
		EventType:      "artifact_stream_terminal",
		Digest:         handle.Digest,
		DataClass:      string(handle.DataClass),
		Terminal:       true,
		TerminalStatus: "failed",
		Error: &ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: errorEnvelopeSchemaVersion,
			Code:          "broker_limit_message_size_exceeded",
			Category:      "transport",
			Retryable:     false,
			Message:       "artifact stream exceeded broker max response stream bytes",
		},
	}
}

func artifactStreamTerminalFromReadErr(handle ArtifactReadHandle, seq int64, readErr error) ArtifactStreamEvent {
	if readErr == io.EOF {
		return ArtifactStreamEvent{
			SchemaID:       "runecode.protocol.v0.ArtifactStreamEvent",
			SchemaVersion:  "0.1.0",
			StreamID:       handle.StreamID,
			RequestID:      handle.RequestID,
			Seq:            seq,
			EventType:      "artifact_stream_terminal",
			Digest:         handle.Digest,
			DataClass:      string(handle.DataClass),
			EOF:            true,
			Terminal:       true,
			TerminalStatus: "completed",
		}
	}
	return ArtifactStreamEvent{
		SchemaID:       "runecode.protocol.v0.ArtifactStreamEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       handle.StreamID,
		RequestID:      handle.RequestID,
		Seq:            seq,
		EventType:      "artifact_stream_terminal",
		Digest:         handle.Digest,
		DataClass:      string(handle.DataClass),
		Terminal:       true,
		TerminalStatus: "failed",
		Error: &ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: errorEnvelopeSchemaVersion,
			Code:          "gateway_failure",
			Category:      "internal",
			Retryable:     false,
			Message:       readErr.Error(),
		},
	}
}

func (s *Service) HandleAuditTimeline(ctx context.Context, req AuditTimelineRequest, meta RequestContext) (AuditTimelineResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditTimelineRequestSchemaPath)
	if errResp != nil {
		return AuditTimelineResponse{}, errResp
	}
	order := req.Order
	if order == "" {
		order = "operational_seq_asc"
	}
	surface, err := s.LatestAuditVerificationSurface(1000)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return AuditTimelineResponse{}, &errOut
	}
	views := append([]trustpolicy.AuditOperationalView{}, surface.Views...)
	if order == "operational_seq_desc" {
		reverseViews(views)
	}
	limit := normalizeLimit(req.Limit, 100, 500)
	page, next, err := paginate(views, req.Cursor, limit)
	if err != nil {
		errOut := s.makeError(requestID, "broker_validation_schema_invalid", "validation", false, err.Error())
		return AuditTimelineResponse{}, &errOut
	}
	resp := AuditTimelineResponse{SchemaID: "runecode.protocol.v0.AuditTimelineResponse", SchemaVersion: "0.1.0", RequestID: requestID, Order: order, Views: page, NextCursor: next}
	if err := s.validateResponse(resp, auditTimelineResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditTimelineResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleAuditVerificationGet(ctx context.Context, req AuditVerificationGetRequest, meta RequestContext) (AuditVerificationGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, auditVerificationGetRequestSchemaPath)
	if errResp != nil {
		return AuditVerificationGetResponse{}, errResp
	}
	limit := req.ViewLimit
	if limit <= 0 {
		limit = 50
	}
	surface, err := s.LatestAuditVerificationSurface(limit)
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return AuditVerificationGetResponse{}, &errOut
	}
	resp := AuditVerificationGetResponse{SchemaID: "runecode.protocol.v0.AuditVerificationGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Summary: surface.Summary, Report: surface.Report, Views: surface.Views}
	if err := s.validateResponse(resp, auditVerificationGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return AuditVerificationGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleLogStreamRequest(ctx context.Context, req LogStreamRequest, meta RequestContext) (LogStreamRequest, *ErrorResponse) {
	if strings.TrimSpace(req.StreamID) == "" {
		req.StreamID = "log-" + resolveRequestID(req.RequestID, meta.RequestID)
	}
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, logStreamRequestSchemaPath)
	if errResp != nil {
		return LogStreamRequest{}, errResp
	}
	ack := req
	ack.RequestID = requestID
	if ack.StreamID == "" {
		ack.StreamID = "log-" + requestID
	}
	return ack, nil
}

func (s *Service) StreamLogEvents(req LogStreamRequest) ([]LogStreamEvent, error) {
	events := make([]LogStreamEvent, 0, 8)
	events = append(events, logStreamStartEvent(req, 1))
	seq := int64(2)
	for _, record := range s.logRecordsForRequest(req) {
		events = append(events, logStreamChunkEvent(req, seq, record))
		seq++
	}
	events = append(events, logStreamTerminalEvent(req, seq))

	if err := validateLogStreamSemantics(events); err != nil {
		return nil, err
	}
	for i := range events {
		if err := s.validateResponse(events[i], logStreamEventSchemaPath); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func logStreamStartEvent(req LogStreamRequest, seq int64) LogStreamEvent {
	return LogStreamEvent{
		SchemaID:       "runecode.protocol.v0.LogStreamEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       req.StreamID,
		RequestID:      req.RequestID,
		Seq:            seq,
		EventType:      "log_stream_start",
		RunID:          req.RunID,
		RoleInstanceID: req.RoleInstanceID,
	}
}

func logStreamChunkEvent(req LogStreamRequest, seq int64, record logStreamRecord) LogStreamEvent {
	return LogStreamEvent{
		SchemaID:       "runecode.protocol.v0.LogStreamEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       req.StreamID,
		RequestID:      req.RequestID,
		Seq:            seq,
		EventType:      "log_stream_chunk",
		RunID:          record.RunID,
		RoleInstanceID: record.RoleInstanceID,
		Cursor:         record.Cursor,
		Timestamp:      record.Timestamp,
		Level:          record.Level,
		Message:        record.Message,
	}
}

func logStreamTerminalEvent(req LogStreamRequest, seq int64) LogStreamEvent {
	terminal := LogStreamEvent{
		SchemaID:       "runecode.protocol.v0.LogStreamEvent",
		SchemaVersion:  "0.1.0",
		StreamID:       req.StreamID,
		RequestID:      req.RequestID,
		Seq:            seq,
		EventType:      "log_stream_terminal",
		RunID:          req.RunID,
		RoleInstanceID: req.RoleInstanceID,
		Terminal:       true,
		TerminalStatus: "completed",
	}
	if req.StartCursor == "force_failure" {
		terminal.TerminalStatus = "failed"
		terminal.Error = &ProtocolError{
			SchemaID:      "runecode.protocol.v0.Error",
			SchemaVersion: errorEnvelopeSchemaVersion,
			Code:          "gateway_failure",
			Category:      "internal",
			Retryable:     false,
			Message:       "forced log stream failure",
		}
	}
	return terminal
}

func (s *Service) logRecordsForRequest(req LogStreamRequest) []logStreamRecord {
	now := time.Now().UTC()
	seed := []logStreamRecord{
		{RunID: "run-123", RoleInstanceID: "workspace-1", Cursor: "cursor-1", Timestamp: now.Add(-2 * time.Second).Format(time.RFC3339Nano), Level: "info", Message: "workflow started"},
		{RunID: "run-123", RoleInstanceID: "workspace-1", Cursor: "cursor-2", Timestamp: now.Add(-1 * time.Second).Format(time.RFC3339Nano), Level: "info", Message: "artifact prepared"},
		{RunID: "run-xyz", RoleInstanceID: "gateway-1", Cursor: "cursor-3", Timestamp: now.Format(time.RFC3339Nano), Level: "warn", Message: "waiting for approval"},
	}
	filtered := make([]logStreamRecord, 0, len(seed)+1)
	for _, rec := range seed {
		if req.RunID != "" && rec.RunID != req.RunID {
			continue
		}
		if req.RoleInstanceID != "" && rec.RoleInstanceID != req.RoleInstanceID {
			continue
		}
		if req.StartCursor != "" && rec.Cursor <= req.StartCursor {
			continue
		}
		filtered = append(filtered, rec)
	}
	if req.Follow {
		filtered = append(filtered, logStreamRecord{
			RunID:          coalesce(req.RunID, "run-123"),
			RoleInstanceID: coalesce(req.RoleInstanceID, "workspace-1"),
			Cursor:         "cursor-live",
			Timestamp:      now.Add(1 * time.Second).Format(time.RFC3339Nano),
			Level:          "info",
			Message:        "live log tail event",
		})
	}
	if !req.IncludeBacklog {
		if len(filtered) == 0 {
			return nil
		}
		last := filtered[len(filtered)-1]
		return []logStreamRecord{last}
	}
	return filtered
}

func validateArtifactStreamSemantics(events []ArtifactStreamEvent) error {
	if len(events) == 0 {
		return fmt.Errorf("artifact stream must emit at least one event")
	}
	streamID := events[0].StreamID
	requestID := events[0].RequestID
	if strings.TrimSpace(streamID) == "" {
		return fmt.Errorf("artifact stream_id is required")
	}
	if strings.TrimSpace(requestID) == "" {
		return fmt.Errorf("artifact request_id is required")
	}
	terminalCount := 0
	for i, event := range events {
		if err := validateStableStreamEventIDs("artifact", event.StreamID, streamID, event.RequestID, requestID); err != nil {
			return err
		}
		if err := validateStrictlyMonotonicSeq("artifact", events, i); err != nil {
			return err
		}
		if event.EventType != "artifact_stream_terminal" {
			continue
		}
		terminalCount++
		if err := validateTerminalEvent("artifact", event.Terminal, event.TerminalStatus, event.Error != nil); err != nil {
			return err
		}
	}
	if terminalCount != 1 {
		return fmt.Errorf("artifact stream must include exactly one terminal event")
	}
	if events[len(events)-1].EventType != "artifact_stream_terminal" {
		return fmt.Errorf("artifact terminal event must be last event")
	}
	return nil
}

func validateLogStreamSemantics(events []LogStreamEvent) error {
	if len(events) == 0 {
		return fmt.Errorf("log stream must emit at least one event")
	}
	streamID := events[0].StreamID
	requestID := events[0].RequestID
	if strings.TrimSpace(streamID) == "" {
		return fmt.Errorf("log stream_id is required")
	}
	if strings.TrimSpace(requestID) == "" {
		return fmt.Errorf("log request_id is required")
	}
	terminalCount := 0
	for i, event := range events {
		if err := validateStableStreamEventIDs("log", event.StreamID, streamID, event.RequestID, requestID); err != nil {
			return err
		}
		if err := validateStrictlyMonotonicSeq("log", events, i); err != nil {
			return err
		}
		if event.EventType != "log_stream_terminal" {
			continue
		}
		terminalCount++
		if err := validateTerminalEvent("log", event.Terminal, event.TerminalStatus, event.Error != nil); err != nil {
			return err
		}
	}
	if terminalCount != 1 {
		return fmt.Errorf("log stream must include exactly one terminal event")
	}
	if events[len(events)-1].EventType != "log_stream_terminal" {
		return fmt.Errorf("log terminal event must be last event")
	}
	return nil
}

func validateStableStreamEventIDs(kind, streamID, expectedStreamID, requestID, expectedRequestID string) error {
	if streamID != expectedStreamID {
		return fmt.Errorf("%s stream_id must remain stable", kind)
	}
	if requestID != expectedRequestID {
		return fmt.Errorf("%s request_id must remain stable", kind)
	}
	return nil
}

func validateStrictlyMonotonicSeq[T interface{ GetSeq() int64 }](kind string, events []T, index int) error {
	if index == 0 {
		if events[0].GetSeq() < 1 {
			return fmt.Errorf("%s seq must start at >=1", kind)
		}
		return nil
	}
	if events[index].GetSeq() <= events[index-1].GetSeq() {
		return fmt.Errorf("%s seq must be strictly monotonic", kind)
	}
	return nil
}

func (e ArtifactStreamEvent) GetSeq() int64 { return e.Seq }
func (e LogStreamEvent) GetSeq() int64      { return e.Seq }

func validateTerminalEvent(kind string, terminal bool, terminalStatus string, hasError bool) error {
	if !terminal {
		return fmt.Errorf("%s terminal event must set terminal=true", kind)
	}
	if strings.TrimSpace(terminalStatus) == "" {
		return fmt.Errorf("%s terminal event must set terminal_status", kind)
	}
	if terminalStatus != "completed" && terminalStatus != "failed" && terminalStatus != "cancelled" {
		return fmt.Errorf("%s terminal_status %q unsupported", kind, terminalStatus)
	}
	if terminalStatus == "failed" && !hasError {
		return fmt.Errorf("failed %s terminal event must include error envelope", kind)
	}
	if terminalStatus == "completed" && hasError {
		return fmt.Errorf("completed %s terminal event must not include error envelope", kind)
	}
	return nil
}

func coalesce(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func (s *Service) HandleReadinessGet(ctx context.Context, req ReadinessGetRequest, meta RequestContext) (ReadinessGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, readinessGetRequestSchemaPath)
	if errResp != nil {
		return ReadinessGetResponse{}, errResp
	}
	readiness, err := s.AuditReadiness()
	if err != nil {
		errOut := s.makeError(requestID, "gateway_failure", "internal", false, err.Error())
		return ReadinessGetResponse{}, &errOut
	}
	model := BrokerReadiness{
		SchemaID:                  "runecode.protocol.v0.BrokerReadiness",
		SchemaVersion:             "0.1.0",
		Ready:                     readiness.Ready,
		LocalOnly:                 readiness.LocalOnly,
		ConsumptionChannel:        readiness.ConsumptionChannel,
		RecoveryComplete:          readiness.RecoveryComplete,
		AppendPositionStable:      readiness.AppendPositionStable,
		CurrentSegmentWritable:    readiness.CurrentSegmentWritable,
		VerifierMaterialAvailable: readiness.VerifierMaterialAvailable,
		DerivedIndexCaughtUp:      readiness.DerivedIndexCaughtUp,
	}
	resp := ReadinessGetResponse{SchemaID: "runecode.protocol.v0.ReadinessGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, Readiness: model}
	if err := s.validateResponse(resp, readinessGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return ReadinessGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) HandleVersionInfoGet(ctx context.Context, req VersionInfoGetRequest, meta RequestContext) (VersionInfoGetResponse, *ErrorResponse) {
	requestID, errResp := s.prepareLocalRequest(req.RequestID, meta.RequestID, meta.AdmissionErr, req, versionInfoGetRequestSchemaPath)
	if errResp != nil {
		return VersionInfoGetResponse{}, errResp
	}
	resp := VersionInfoGetResponse{SchemaID: "runecode.protocol.v0.VersionInfoGetResponse", SchemaVersion: "0.1.0", RequestID: requestID, VersionInfo: s.versionInfo}
	if err := s.validateResponse(resp, versionInfoGetResponseSchemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return VersionInfoGetResponse{}, &errOut
	}
	return resp, nil
}

func (s *Service) prepareLocalRequest(reqID, fallbackReqID string, admissionErr error, req any, schemaPath string) (string, *ErrorResponse) {
	requestID := strings.TrimSpace(resolveRequestID(reqID, fallbackReqID))
	if admissionErr != nil {
		errID := requestID
		if errID == "" {
			errID = defaultRequestIDFallback
		}
		err := s.makeError(errID, "broker_api_auth_admission_denied", "auth", false, admissionErr.Error())
		return "", &err
	}
	if requestID == "" {
		err := s.makeError(defaultRequestIDFallback, "broker_validation_request_id_missing", "validation", false, "request_id is required")
		return "", &err
	}
	if err := s.validateRequest(req, schemaPath); err != nil {
		errOut := s.errorFromValidation(requestID, err)
		return "", &errOut
	}
	return requestID, nil
}

func (s *Service) runSummaries(order string) ([]RunSummary, error) {
	runStatus := s.RunStatuses()
	verification := s.runAuditVerificationOrFallback()
	byRun := buildRunRecordIndex(s.List(), runStatus)
	summaries := make([]RunSummary, 0, len(byRun))
	for runID, records := range byRun {
		summaries = append(summaries, buildRunSummary(runID, records, runStatus[runID], verification))
	}
	sortRunSummaries(summaries, order)
	return summaries, nil
}

func (s *Service) runAuditVerificationOrFallback() AuditVerificationSurface {
	verification, err := s.LatestAuditVerificationSurface(20)
	if err == nil {
		return verification
	}
	return AuditVerificationSurface{
		Summary: trustpolicy.DerivedRunAuditVerificationSummary{
			CryptographicallyValid: false,
			HistoricallyAdmissible: false,
			CurrentlyDegraded:      true,
			IntegrityStatus:        "failed",
			AnchoringStatus:        "failed",
			StoragePostureStatus:   "failed",
			SegmentLifecycleStatus: "failed",
			HardFailures:           []string{"audit_surface_unavailable"},
		},
	}
}

func buildRunRecordIndex(all []artifacts.ArtifactRecord, runStatus map[string]string) map[string][]artifacts.ArtifactRecord {
	byRun := map[string][]artifacts.ArtifactRecord{}
	for _, rec := range all {
		if rec.RunID == "" {
			continue
		}
		byRun[rec.RunID] = append(byRun[rec.RunID], rec)
	}
	for runID := range runStatus {
		if _, ok := byRun[runID]; !ok {
			byRun[runID] = nil
		}
	}
	return byRun
}

func buildRunSummary(runID string, records []artifacts.ArtifactRecord, status string, verification AuditVerificationSurface) RunSummary {
	created, updated, pending := runRecordTimingAndPending(records)
	state := runLifecycleFromStore(status, pending)
	summary := RunSummary{
		SchemaID:               "runecode.protocol.v0.RunSummary",
		SchemaVersion:          "0.1.0",
		RunID:                  runID,
		WorkspaceID:            "local-workspace",
		WorkflowKind:           "broker_local_mvp",
		WorkflowDefinitionHash: "sha256:" + strings.Repeat("0", 64),
		CreatedAt:              created.UTC().Format(time.RFC3339),
		StartedAt:              created.UTC().Format(time.RFC3339),
		UpdatedAt:              updated.UTC().Format(time.RFC3339),
		LifecycleState:         state,
		CurrentStageID:         "artifact_flow",
		PendingApprovalCount:   pending,
		ApprovalProfile:        "moderate",
		BackendKind:            "local",
		AssuranceLevel:         "session_authenticated",
		AuditIntegrityStatus:   verification.Summary.IntegrityStatus,
		AuditAnchoringStatus:   verification.Summary.AnchoringStatus,
		AuditCurrentlyDegraded: verification.Summary.CurrentlyDegraded,
	}
	if state == "blocked" {
		summary.BlockingReasonCode = "pending_approval"
	}
	if state == "completed" || state == "failed" || state == "cancelled" {
		summary.FinishedAt = updated.UTC().Format(time.RFC3339)
	}
	return summary
}

func runRecordTimingAndPending(records []artifacts.ArtifactRecord) (time.Time, time.Time, int) {
	created := time.Now().UTC()
	updated := created
	pending := 0
	for _, rec := range records {
		if rec.CreatedAt.Before(created) {
			created = rec.CreatedAt
		}
		if rec.CreatedAt.After(updated) {
			updated = rec.CreatedAt
		}
		if rec.Reference.DataClass == artifacts.DataClassUnapprovedFileExcerpts {
			pending++
		}
	}
	return created, updated, pending
}

func sortRunSummaries(summaries []RunSummary, order string) {
	sort.Slice(summaries, func(i, j int) bool {
		if order == "updated_at_asc" {
			return summaries[i].UpdatedAt < summaries[j].UpdatedAt
		}
		if summaries[i].UpdatedAt == summaries[j].UpdatedAt {
			return summaries[i].RunID < summaries[j].RunID
		}
		return summaries[i].UpdatedAt > summaries[j].UpdatedAt
	})
}

func runLifecycleFromStore(status string, pendingApprovals int) string {
	if pendingApprovals > 0 {
		return "blocked"
	}
	switch status {
	case "active":
		return "active"
	case "retained", "closed":
		return "completed"
	default:
		if status == "" {
			return "active"
		}
		return "active"
	}
}

func (s *Service) runDetail(runID string) (RunDetail, bool, error) {
	summaries, err := s.runSummaries("updated_at_desc")
	if err != nil {
		return RunDetail{}, false, err
	}
	summary, found := findRunSummary(summaries, runID)
	if !found {
		return RunDetail{}, false, nil
	}
	artifactsForRun, classCount := runArtifactsAndClassCount(s.List(), runID)
	pendingIDs := runPendingApprovalIDs(s.listApprovals(), runID)
	verification, _ := s.LatestAuditVerificationSurface(20)
	return buildRunDetail(summary, verification, artifactsForRun, classCount, pendingIDs), true, nil
}

func findRunSummary(summaries []RunSummary, runID string) (RunSummary, bool) {
	for _, item := range summaries {
		if item.RunID == runID {
			return item, true
		}
	}
	return RunSummary{}, false
}

func runArtifactsAndClassCount(all []artifacts.ArtifactRecord, runID string) ([]artifacts.ArtifactRecord, map[string]int) {
	artifactsForRun := make([]artifacts.ArtifactRecord, 0)
	classCount := map[string]int{}
	for _, rec := range all {
		if rec.RunID != runID {
			continue
		}
		artifactsForRun = append(artifactsForRun, rec)
		classCount[string(rec.Reference.DataClass)]++
	}
	return artifactsForRun, classCount
}

func runPendingApprovalIDs(approvals []ApprovalSummary, runID string) []string {
	pendingIDs := make([]string, 0)
	for _, approval := range approvals {
		if approval.Status == "pending" && approval.BoundScope.RunID == runID {
			pendingIDs = append(pendingIDs, approval.ApprovalID)
		}
	}
	sort.Strings(pendingIDs)
	return pendingIDs
}

func buildRunDetail(summary RunSummary, verification AuditVerificationSurface, artifactsForRun []artifacts.ArtifactRecord, classCount map[string]int, pendingIDs []string) RunDetail {
	stages := []RunStageSummary{{
		SchemaID:             "runecode.protocol.v0.RunStageSummary",
		SchemaVersion:        "0.1.0",
		StageID:              "artifact_flow",
		LifecycleState:       summary.LifecycleState,
		StartedAt:            summary.StartedAt,
		FinishedAt:           summary.FinishedAt,
		PendingApprovalCount: len(pendingIDs),
		ArtifactCount:        len(artifactsForRun),
	}}
	roles := []RunRoleSummary{{
		SchemaID:        "runecode.protocol.v0.RunRoleSummary",
		SchemaVersion:   "0.1.0",
		RoleInstanceID:  "workspace-1",
		RoleKind:        "workspace",
		LifecycleState:  summary.LifecycleState,
		ActiveItemCount: len(artifactsForRun),
	}}
	coord := RunCoordinationSummary{
		SchemaID:         "runecode.protocol.v0.RunCoordinationSummary",
		SchemaVersion:    "0.1.0",
		Blocked:          summary.LifecycleState == "blocked",
		WaitReasonCode:   summary.BlockingReasonCode,
		LockCount:        0,
		ConflictCount:    0,
		CoordinationMode: "single_broker_queue",
	}
	return RunDetail{
		SchemaID:                 "runecode.protocol.v0.RunDetail",
		SchemaVersion:            "0.1.0",
		Summary:                  summary,
		StageSummaries:           stages,
		RoleSummaries:            roles,
		Coordination:             coord,
		AuditSummary:             verification.Summary,
		ArtifactCountsByClass:    classCount,
		PendingApprovalIDs:       pendingIDs,
		ActiveManifestHashes:     []string{"sha256:" + strings.Repeat("0", 64)},
		LatestPolicyDecisionRefs: []string{},
		AuthoritativeState:       map[string]any{"source": "broker_store", "status": summary.LifecycleState},
		AdvisoryState:            map[string]any{"source": "runner_advisory", "available": false},
	}
}

func (s *Service) seedStubApprovals() error {
	s.approvals.mu.Lock()
	defer s.approvals.mu.Unlock()
	if s.approvals.seeded {
		return nil
	}
	if s.approvals.records == nil {
		s.approvals.records = map[string]approvalRecord{}
	}
	runs, err := s.runSummaries("updated_at_desc")
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for i, run := range runs {
		if i >= 2 {
			break
		}
		id := shaDigestIdentity("stub-approval:" + run.RunID)
		s.approvals.records[id] = approvalRecord{Summary: ApprovalSummary{
			SchemaID:               "runecode.protocol.v0.ApprovalSummary",
			SchemaVersion:          "0.1.0",
			ApprovalID:             id,
			Status:                 "pending",
			RequestedAt:            now.Add(-time.Duration(i+1) * time.Minute).Format(time.RFC3339),
			ExpiresAt:              now.Add(20 * time.Minute).Format(time.RFC3339),
			ApprovalTriggerCode:    "stage_sign_off",
			ChangesIfApproved:      "Unblock stage progression for local workflow.",
			ApprovalAssuranceLevel: "session_authenticated",
			PresenceMode:           "os_confirmation",
			BoundScope: ApprovalBoundScope{
				SchemaID:      "runecode.protocol.v0.ApprovalBoundScope",
				SchemaVersion: "0.1.0",
				WorkspaceID:   run.WorkspaceID,
				RunID:         run.RunID,
				StageID:       run.CurrentStageID,
				ActionKind:    "stage_transition",
			},
			PolicyDecisionHash: "sha256:" + strings.Repeat("1", 64),
			RequestDigest:      id,
		}}
	}
	s.approvals.seeded = true
	return nil
}

func (s *Service) listApprovals() []ApprovalSummary {
	s.approvals.mu.Lock()
	defer s.approvals.mu.Unlock()
	out := make([]ApprovalSummary, 0, len(s.approvals.records))
	for _, record := range s.approvals.records {
		out = append(out, record.Summary)
	}
	return out
}

func sortApprovals(items []ApprovalSummary) {
	statusRank := map[string]int{"pending": 0, "approved": 1, "denied": 2, "expired": 3, "cancelled": 4, "superseded": 5, "consumed": 6}
	sort.SliceStable(items, func(i, j int) bool {
		ri := statusRank[items[i].Status]
		rj := statusRank[items[j].Status]
		if ri != rj {
			return ri < rj
		}
		if items[i].RequestedAt == items[j].RequestedAt {
			return items[i].ApprovalID < items[j].ApprovalID
		}
		return items[i].RequestedAt > items[j].RequestedAt
	})
}

func (s *Service) getApproval(id string) (approvalRecord, bool) {
	s.approvals.mu.Lock()
	defer s.approvals.mu.Unlock()
	rec, ok := s.approvals.records[id]
	return rec, ok
}

func (s *Service) putApproval(rec approvalRecord) {
	s.approvals.mu.Lock()
	defer s.approvals.mu.Unlock()
	if s.approvals.records == nil {
		s.approvals.records = map[string]approvalRecord{}
	}
	s.approvals.records[rec.Summary.ApprovalID] = rec
}

func approvalIDFromRequest(envelope trustpolicy.SignedObjectEnvelope) (string, error) {
	canonical, err := jsoncanonicalizer.Transform(envelope.Payload)
	if err != nil {
		return "", fmt.Errorf("canonicalize approval request payload: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func signedEnvelopeDigest(envelope trustpolicy.SignedObjectEnvelope) (string, error) {
	b, err := json.Marshal(envelope)
	if err != nil {
		return "", err
	}
	canonical, err := jsoncanonicalizer.Transform(b)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func decodeDecisionString(payload []byte, field string, fallback string) string {
	value := map[string]any{}
	if err := json.Unmarshal(payload, &value); err != nil {
		return fallback
	}
	v, ok := value[field].(string)
	if !ok || strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func reverseViews(views []trustpolicy.AuditOperationalView) {
	for i, j := 0, len(views)-1; i < j; i, j = i+1, j-1 {
		views[i], views[j] = views[j], views[i]
	}
}

func ptrArtifactSummary(value ArtifactSummary) *ArtifactSummary {
	v := value
	return &v
}

func shaDigestIdentity(input string) string {
	sum := sha256.Sum256([]byte(input))
	return "sha256:" + hex.EncodeToString(sum[:])
}

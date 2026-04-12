package brokerapi

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestHandleArtifactListRejectsMissingRequestID(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := service.HandleArtifactList(context.Background(), ArtifactListRequest{
		SchemaID:      "runecode.protocol.v0.BrokerArtifactListRequest",
		SchemaVersion: "0.1.0",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleArtifactList error = nil, want typed error response")
	}
	if errResp.Error.Code != "broker_validation_request_id_missing" {
		t.Fatalf("error code = %q, want broker_validation_request_id_missing", errResp.Error.Code)
	}
	if errResp.RequestID == "" {
		t.Fatal("error request_id empty, want stable fallback request_id")
	}
	if err := validateJSONEnvelope(errResp, brokerErrorResponseSchemaPath); err != nil {
		t.Fatalf("error envelope schema validation failed: %v", err)
	}
}

func TestHandleArtifactHeadRejectsSchemaValidationFailure(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := service.HandleArtifactHead(context.Background(), ArtifactHeadRequest{
		SchemaID:      "runecode.protocol.v0.BrokerArtifactHeadRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-head-1",
		Digest:        "not-a-digest",
	}, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleArtifactHead error = nil, want typed error response")
	}
	if errResp.Error.Code != "broker_validation_schema_invalid" {
		t.Fatalf("error code = %q, want broker_validation_schema_invalid", errResp.Error.Code)
	}
	if errResp.RequestID != "req-head-1" {
		t.Fatalf("error request_id = %q, want req-head-1", errResp.RequestID)
	}
}

func TestHandleArtifactPutRejectsMessageSizeLimit(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxMessageBytes: 2048}})
	request := DefaultArtifactPutRequest(
		"req-put-1",
		[]byte(strings.Repeat("a", 3000)),
		"text/plain",
		"spec_text",
		"sha256:"+strings.Repeat("1", 64),
		"workspace",
		"run-1",
		"step-1",
	)
	_, errResp := service.HandleArtifactPut(context.Background(), request, RequestContext{})
	if errResp == nil {
		t.Fatal("HandleArtifactPut error = nil, want typed error response")
	}
	if errResp.Error.Code != "broker_limit_message_size_exceeded" {
		t.Fatalf("error code = %q, want broker_limit_message_size_exceeded", errResp.Error.Code)
	}
}

func TestHandleArtifactListRejectsInFlightLimit(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxInFlightPerClient: 1, MaxInFlightPerLane: 1}})
	release, err := service.apiInflight.acquire("client-a", "lane-a")
	if err != nil {
		t.Fatalf("acquire precondition returned error: %v", err)
	}
	defer release()
	_, errResp := service.HandleArtifactList(
		context.Background(),
		DefaultArtifactListRequest("req-list-1"),
		RequestContext{ClientID: "client-a", LaneID: "lane-a"},
	)
	if errResp == nil {
		t.Fatal("HandleArtifactList error = nil, want typed error response")
	}
	if errResp.Error.Code != "broker_limit_in_flight_exceeded" {
		t.Fatalf("error code = %q, want broker_limit_in_flight_exceeded", errResp.Error.Code)
	}
}

func TestHandleArtifactListRejectsRateLimitWithTypedCode(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxRequestsPerClientPS: 1}})
	fixed := time.Date(2026, time.April, 7, 12, 0, 0, 0, time.UTC)
	service.SetNowFuncForTests(func() time.Time { return fixed })
	meta := RequestContext{ClientID: "client-rate", LaneID: "lane-rate"}

	if _, errResp := service.HandleArtifactList(context.Background(), DefaultArtifactListRequest("req-rate-1"), meta); errResp != nil {
		t.Fatalf("first request error response: %+v", errResp)
	}
	_, errResp := service.HandleArtifactList(context.Background(), DefaultArtifactListRequest("req-rate-2"), meta)
	if errResp == nil {
		t.Fatal("second request expected typed rate-limit error")
	}
	if errResp.Error.Code != "broker_limit_rate_exceeded" {
		t.Fatalf("error code = %q, want broker_limit_rate_exceeded", errResp.Error.Code)
	}
	if !errResp.Error.Retryable {
		t.Fatal("rate limit rejection should be retryable")
	}

	service.SetNowFuncForTests(func() time.Time { return fixed.Add(1 * time.Second) })
	if _, errResp := service.HandleArtifactList(context.Background(), DefaultArtifactListRequest("req-rate-3"), meta); errResp != nil {
		t.Fatalf("request after next window error response: %+v", errResp)
	}
}

func TestHandleArtifactListRejectsDeadlineExceeded(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	deadline := time.Now().Add(-time.Second)
	_, errResp := service.HandleArtifactList(
		context.Background(),
		DefaultArtifactListRequest("req-list-timeout"),
		RequestContext{Deadline: &deadline},
	)
	if errResp == nil {
		t.Fatal("HandleArtifactList error = nil, want typed error response")
	}
	if errResp.Error.Code != "broker_timeout_request_deadline_exceeded" {
		t.Fatalf("error code = %q, want broker_timeout_request_deadline_exceeded", errResp.Error.Code)
	}
}

func TestHandleArtifactHeadEchoesStableRequestID(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	requestID := "req-head-notfound"
	_, errResp := service.HandleArtifactHead(
		context.Background(),
		DefaultArtifactHeadRequest(requestID, "sha256:"+strings.Repeat("a", 64)),
		RequestContext{},
	)
	if errResp == nil {
		t.Fatal("HandleArtifactHead error = nil, want typed error response")
	}
	if errResp.RequestID != requestID {
		t.Fatalf("error request_id = %q, want %q", errResp.RequestID, requestID)
	}
	if errResp.Error.Code != "broker_not_found_artifact" {
		t.Fatalf("error code = %q, want broker_not_found_artifact", errResp.Error.Code)
	}
}

func TestHandleArtifactListRejectsAdmissionFailureWithTypedError(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	admissionErr := errors.New("peer uid does not match broker uid")
	_, errResp := service.HandleArtifactList(
		context.Background(),
		DefaultArtifactListRequest("req-list-admission"),
		RequestContext{AdmissionErr: admissionErr},
	)
	if errResp == nil {
		t.Fatal("HandleArtifactList error = nil, want typed auth admission error")
	}
	if errResp.Error.Code != "broker_api_auth_admission_denied" {
		t.Fatalf("error code = %q, want broker_api_auth_admission_denied", errResp.Error.Code)
	}
	if errResp.Error.Category != "auth" {
		t.Fatalf("error category = %q, want auth", errResp.Error.Category)
	}
}

func newBrokerAPIServiceForTests(t *testing.T, cfg APIConfig) *Service {
	t.Helper()
	root := t.TempDir()
	service, err := NewServiceWithConfig(root, root+"/audit-ledger", cfg)
	if err != nil {
		t.Fatalf("NewServiceWithConfig returned error: %v", err)
	}
	return service
}

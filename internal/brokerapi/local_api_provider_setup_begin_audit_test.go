package brokerapi

import (
	"context"
	"sync"
	"testing"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestProviderSetupBeginCanonicalEquivalentDestinationEmitsUpdatedChangeKind(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	firstReqID := "req-provider-canonical-first"
	secondReqID := "req-provider-canonical-second"

	first := mustBeginProviderSetupWithRequest(t, service, providerSetupBeginRequest(firstReqID, "API.OpenAI.COM", "/v1/"))
	second := mustBeginProviderSetupWithRequest(t, service, providerSetupBeginRequest(secondReqID, "api.openai.com", "/v1"))

	if first.Profile.ProviderProfileID != second.Profile.ProviderProfileID {
		t.Fatalf("provider_profile_id changed across canonical-equivalent destination: first=%q second=%q", first.Profile.ProviderProfileID, second.Profile.ProviderProfileID)
	}

	events := mustReadAuditEvents(t, service)
	assertProviderProfileAuditChangeKind(t, events, firstReqID, first.Profile.ProviderProfileID, "created")
	assertProviderProfileAuditChangeKind(t, events, secondReqID, second.Profile.ProviderProfileID, "updated")
}

func TestProviderSetupBeginConcurrentCanonicalEquivalentRequestsEmitSingleCreatedChangeKind(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	requests := []ProviderSetupSessionBeginRequest{
		providerSetupBeginRequest("req-provider-canonical-concurrent-1", "API.OpenAI.COM", "/v1/"),
		providerSetupBeginRequest("req-provider-canonical-concurrent-2", "api.openai.com", "/v1"),
	}
	responses := make([]ProviderSetupSessionBeginResponse, len(requests))
	errs := make([]*ErrorResponse, len(requests))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range requests {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			responses[i], errs[i] = service.HandleProviderSetupSessionBegin(context.Background(), requests[i], RequestContext{})
		}(i)
	}
	close(start)
	wg.Wait()
	assertConcurrentBeginResponses(t, responses, errs)

	events := mustReadAuditEvents(t, service)
	created, updated := providerProfileAuditChangeCounts(events, responses[0].Profile.ProviderProfileID)
	if created != 1 || updated != len(requests)-1 {
		t.Fatalf("provider profile change counts = created:%d updated:%d, want created:1 updated:%d", created, updated, len(requests)-1)
	}
}

func providerSetupBeginRequest(requestID, canonicalHost, canonicalPathPrefix string) ProviderSetupSessionBeginRequest {
	return ProviderSetupSessionBeginRequest{
		SchemaID:            "runecode.protocol.v0.ProviderSetupSessionBeginRequest",
		SchemaVersion:       "0.1.0",
		RequestID:           requestID,
		DisplayLabel:        "OpenAI Prod",
		ProviderFamily:      "openai_compatible",
		AdapterKind:         "chat_completions_v0",
		CanonicalHost:       canonicalHost,
		CanonicalPathPrefix: canonicalPathPrefix,
	}
}

func mustBeginProviderSetupWithRequest(t *testing.T, service *Service, req ProviderSetupSessionBeginRequest) ProviderSetupSessionBeginResponse {
	t.Helper()
	resp, errResp := service.HandleProviderSetupSessionBegin(context.Background(), req, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleProviderSetupSessionBegin error response: %+v", errResp)
	}
	return resp
}

func assertConcurrentBeginResponses(t *testing.T, responses []ProviderSetupSessionBeginResponse, errs []*ErrorResponse) {
	t.Helper()
	for i, errResp := range errs {
		if errResp != nil {
			t.Fatalf("HandleProviderSetupSessionBegin[%d] error response: %+v", i, errResp)
		}
	}
	profileID := responses[0].Profile.ProviderProfileID
	for i := 1; i < len(responses); i++ {
		if responses[i].Profile.ProviderProfileID != profileID {
			t.Fatalf("provider_profile_id mismatch across concurrent canonical-equivalent requests: first=%q request[%d]=%q", profileID, i, responses[i].Profile.ProviderProfileID)
		}
	}
}

func mustReadAuditEvents(t *testing.T, service *Service) []artifacts.AuditEvent {
	t.Helper()
	events, err := service.ReadAuditEvents()
	if err != nil {
		t.Fatalf("ReadAuditEvents returned error: %v", err)
	}
	return events
}

func assertProviderProfileAuditChangeKind(t *testing.T, events []artifacts.AuditEvent, requestID, profileID, wantChangeKind string) {
	t.Helper()
	for _, event := range events {
		if event.Type != brokerAuditEventTypeProviderProfile {
			continue
		}
		if event.Details["request_id"] != requestID {
			continue
		}
		if got := event.Details["provider_profile_id"]; got != profileID {
			t.Fatalf("provider_profile_id = %v, want %q", got, profileID)
		}
		if got := event.Details["change_kind"]; got != wantChangeKind {
			t.Fatalf("change_kind = %v, want %q", got, wantChangeKind)
		}
		return
	}
	t.Fatalf("missing provider profile audit event for request_id=%q", requestID)
}

func providerProfileAuditChangeCounts(events []artifacts.AuditEvent, profileID string) (int, int) {
	created := 0
	updated := 0
	for _, event := range events {
		if event.Type != brokerAuditEventTypeProviderProfile {
			continue
		}
		if event.Details["provider_profile_id"] != profileID {
			continue
		}
		switch event.Details["change_kind"] {
		case "created":
			created++
		case "updated":
			updated++
		}
	}
	return created, updated
}

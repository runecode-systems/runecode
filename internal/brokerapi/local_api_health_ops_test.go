package brokerapi

import (
	"context"
	"path/filepath"
	"testing"
)

func TestHandleAuditTimelineProjectsSchemaAlignedViews(t *testing.T) {
	storeRoot := filepath.Join(t.TempDir(), "store")
	ledgerRoot := filepath.Join(t.TempDir(), "ledger")
	if err := seedLedgerForBrokerSurfaceTest(ledgerRoot); err != nil {
		t.Fatalf("seedLedgerForBrokerSurfaceTest returned error: %v", err)
	}
	service, err := NewService(storeRoot, ledgerRoot)
	if err != nil {
		t.Fatalf("NewService returned error: %v", err)
	}
	resp, errResp := service.HandleAuditTimeline(context.Background(), AuditTimelineRequest{
		SchemaID:      "runecode.protocol.v0.AuditTimelineRequest",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-timeline",
		Limit:         10,
		Order:         "operational_seq_desc",
	}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleAuditTimeline returned error: %+v", errResp)
	}
	if len(resp.Views) == 0 {
		t.Fatal("timeline views empty")
	}
	if resp.Views[0].Summary == "" {
		t.Fatal("timeline summary empty")
	}
	if len(resp.Views[0].LinkedReferences) == 0 {
		t.Fatal("timeline linked_references empty")
	}
	if err := service.validateResponse(resp, auditTimelineResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(auditTimelineResponse) returned error: %v", err)
	}
}

func TestHandleAuditTimelineCursorRoundTripSupportsShortEncodedValues(t *testing.T) {
	encoded, err := encodeCursor(pageCursor{Offset: 1})
	if err != nil {
		t.Fatalf("encodeCursor returned error: %v", err)
	}
	if len(encoded) >= 32 {
		t.Fatalf("encoded cursor length = %d, expected short cursor for regression coverage", len(encoded))
	}
	resp := AuditTimelineResponse{
		SchemaID:      "runecode.protocol.v0.AuditTimelineResponse",
		SchemaVersion: "0.1.0",
		RequestID:     "req-audit-cursor",
		Order:         "operational_seq_asc",
		Views: []AuditTimelineViewEntry{{
			RecordDigest: digestChar("a"),
			Summary:      "Audit record projected for timeline.",
		}},
		NextCursor: encoded,
	}
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	if err := s.validateResponse(resp, auditTimelineResponseSchemaPath); err != nil {
		t.Fatalf("validateResponse(short next_cursor) returned error: %v", err)
	}
}

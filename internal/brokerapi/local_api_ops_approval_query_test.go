package brokerapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

func TestHandleApprovalListRejectsInFlightLimit(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxInFlightPerClient: 1, MaxInFlightPerLane: 1}})
	release, err := s.apiInflight.acquire("client-a", "lane-a")
	if err != nil {
		t.Fatalf("acquire precondition returned error: %v", err)
	}
	defer release()
	_, errResp := s.HandleApprovalList(context.Background(), ApprovalListRequest{SchemaID: "runecode.protocol.v0.ApprovalListRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-list-limit"}, RequestContext{ClientID: "client-a", LaneID: "lane-a"})
	if errResp == nil || errResp.Error.Code != "broker_limit_in_flight_exceeded" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestHandleApprovalListRejectsDeadlineExceeded(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	deadline := time.Now().Add(-time.Second)
	_, errResp := s.HandleApprovalList(context.Background(), ApprovalListRequest{SchemaID: "runecode.protocol.v0.ApprovalListRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-list-timeout"}, RequestContext{Deadline: &deadline})
	if errResp == nil || errResp.Error.Code != "broker_timeout_request_deadline_exceeded" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestApprovalListDerivesPendingFromUnapprovedArtifacts(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	ref := putUnapprovedExcerptArtifactForApprovalTest(t, s, "run-approval-derived", "step-1", "a")
	approvalID := createPendingApprovalFromPolicyDecision(t, s, "run-approval-derived", "step-1", ref.Digest)
	resp, errResp := s.HandleApprovalList(context.Background(), ApprovalListRequest{SchemaID: "runecode.protocol.v0.ApprovalListRequest", SchemaVersion: "0.1.0", RequestID: "req-derived-approval-list", RunID: "run-approval-derived"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalList error response: %+v", errResp)
	}
	assertDerivedPendingApproval(t, resp.Approvals, "run-approval-derived", "step-1", approvalID)
}

func putUnapprovedExcerptArtifactForApprovalTest(t *testing.T, s *Service, runID, stepID, hashFill string) artifacts.ArtifactReference {
	t.Helper()
	ref, err := s.Put(artifacts.PutRequest{Payload: []byte("private excerpt"), ContentType: "text/plain", DataClass: artifacts.DataClassUnapprovedFileExcerpts, ProvenanceReceiptHash: "sha256:" + strings.Repeat(hashFill, 64), CreatedByRole: "workspace", RunID: runID, StepID: stepID})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	return ref
}

func assertDerivedPendingApproval(t *testing.T, approvals []ApprovalSummary, runID, stepID, approvalID string) {
	t.Helper()
	if len(approvals) != 1 {
		t.Fatalf("approval count = %d, want 1", len(approvals))
	}
	approval := approvals[0]
	if approval.Status != "pending" || approval.ApprovalTriggerCode != "excerpt_promotion" || approval.BoundScope.RunID != runID || approval.BoundScope.StepID != stepID || approval.ApprovalID != approvalID {
		t.Fatalf("unexpected approval summary: %+v", approval)
	}
}

func TestApprovalGetReturnsDerivedPendingApproval(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	approvalID := createPendingApprovalForGetTest(t, s)
	resp, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-derived-approval-get", ApprovalID: approvalID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalGet error response: %+v", errResp)
	}
	if resp.Approval.ApprovalID != approvalID || resp.SignedApprovalRequest == nil || resp.SignedApprovalDecision != nil {
		t.Fatalf("unexpected approval get response: %+v", resp)
	}
	derivedID, deriveErr := approvalIDFromRequest(*resp.SignedApprovalRequest)
	if deriveErr != nil || derivedID != approvalID {
		t.Fatalf("unexpected approvalIDFromRequest output: id=%q err=%v", derivedID, deriveErr)
	}
	if resp.ApprovalDetail.ApprovalID != approvalID {
		t.Fatalf("approval_detail.approval_id = %q, want %q", resp.ApprovalDetail.ApprovalID, approvalID)
	}
	if resp.ApprovalDetail.PolicyReasonCode != "approval_required" {
		t.Fatalf("approval_detail.policy_reason_code = %q, want approval_required", resp.ApprovalDetail.PolicyReasonCode)
	}
	if resp.ApprovalDetail.LifecycleDetail.LifecycleState != "pending" {
		t.Fatalf("approval_detail.lifecycle_detail.lifecycle_state = %q, want pending", resp.ApprovalDetail.LifecycleDetail.LifecycleState)
	}
	if resp.ApprovalDetail.LifecycleDetail.LifecycleReasonCode != "approval_pending" {
		t.Fatalf("approval_detail.lifecycle_detail.lifecycle_reason_code = %q, want approval_pending", resp.ApprovalDetail.LifecycleDetail.LifecycleReasonCode)
	}
	if resp.ApprovalDetail.LifecycleDetail.Stale {
		t.Fatal("approval_detail.lifecycle_detail.stale should be false for pending approval")
	}
	if resp.ApprovalDetail.BindingKind != "exact_action" {
		t.Fatalf("approval_detail.binding_kind = %q, want exact_action", resp.ApprovalDetail.BindingKind)
	}
	if resp.ApprovalDetail.BoundActionHash == "" {
		t.Fatal("approval_detail.bound_action_hash should be populated for exact_action")
	}
	if resp.ApprovalDetail.WhatChangesIfApproved.Summary != approvalChangesIfApprovedDefault {
		t.Fatalf("approval_detail.what_changes_if_approved.summary = %q", resp.ApprovalDetail.WhatChangesIfApproved.Summary)
	}
	if resp.ApprovalDetail.WhatChangesIfApproved.EffectKind != "promotion" {
		t.Fatalf("approval_detail.what_changes_if_approved.effect_kind = %q, want promotion", resp.ApprovalDetail.WhatChangesIfApproved.EffectKind)
	}
	if resp.ApprovalDetail.BlockedWorkScope.ScopeKind != "stage" {
		t.Fatalf("approval_detail.blocked_work_scope.scope_kind = %q, want stage", resp.ApprovalDetail.BlockedWorkScope.ScopeKind)
	}
	if resp.ApprovalDetail.BoundIdentity.BindingKind != "exact_action" {
		t.Fatalf("approval_detail.bound_identity.binding_kind = %q, want exact_action", resp.ApprovalDetail.BoundIdentity.BindingKind)
	}
	if resp.ApprovalDetail.BoundIdentity.BoundActionHash == "" {
		t.Fatal("approval_detail.bound_identity.bound_action_hash should be populated for exact_action")
	}
	if resp.ApprovalDetail.BoundIdentity.ApprovalRequestDigest != approvalID {
		t.Fatalf("approval_detail.bound_identity.approval_request_digest = %q, want %q", resp.ApprovalDetail.BoundIdentity.ApprovalRequestDigest, approvalID)
	}
}

func TestApprovalGetStageSignOffDetailIncludesStageBindingKindAndHash(t *testing.T) {
	s, requestEnv, _ := setupServiceWithStageSignOffApprovalFixture(t)
	approvalID, err := approvalIDFromRequest(*requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	resp, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-stage-approval-get", ApprovalID: approvalID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalGet error response: %+v", errResp)
	}
	if resp.ApprovalDetail.BindingKind != "stage_sign_off" {
		t.Fatalf("approval_detail.binding_kind = %q, want stage_sign_off", resp.ApprovalDetail.BindingKind)
	}
	if resp.ApprovalDetail.LifecycleDetail.LifecycleState != "pending" || resp.ApprovalDetail.LifecycleDetail.LifecycleReasonCode != "approval_pending" || resp.ApprovalDetail.LifecycleDetail.Stale {
		t.Fatalf("unexpected lifecycle detail for stage-sign-off pending approval: %+v", resp.ApprovalDetail.LifecycleDetail)
	}
	if resp.ApprovalDetail.BoundStageSummaryHash != "sha256:"+strings.Repeat("6", 64) {
		t.Fatalf("approval_detail.bound_stage_summary_hash = %q", resp.ApprovalDetail.BoundStageSummaryHash)
	}
	if resp.ApprovalDetail.WhatChangesIfApproved.EffectKind != "stage_sign_off" {
		t.Fatalf("approval_detail.what_changes_if_approved.effect_kind = %q, want stage_sign_off", resp.ApprovalDetail.WhatChangesIfApproved.EffectKind)
	}
	if resp.ApprovalDetail.BlockedWorkScope.ScopeKind != "stage" {
		t.Fatalf("approval_detail.blocked_work_scope.scope_kind = %q, want stage", resp.ApprovalDetail.BlockedWorkScope.ScopeKind)
	}
	if resp.ApprovalDetail.BoundIdentity.BindingKind != "stage_sign_off" {
		t.Fatalf("approval_detail.bound_identity.binding_kind = %q, want stage_sign_off", resp.ApprovalDetail.BoundIdentity.BindingKind)
	}
	if resp.ApprovalDetail.BoundIdentity.BoundStageSummaryHash != "sha256:"+strings.Repeat("6", 64) {
		t.Fatalf("approval_detail.bound_identity.bound_stage_summary_hash = %q", resp.ApprovalDetail.BoundIdentity.BoundStageSummaryHash)
	}
}

func createPendingApprovalForGetTest(t *testing.T, s *Service) string {
	t.Helper()
	ref, err := s.Put(artifacts.PutRequest{Payload: []byte("private excerpt"), ContentType: "text/plain", DataClass: artifacts.DataClassUnapprovedFileExcerpts, ProvenanceReceiptHash: "sha256:" + strings.Repeat("b", 64), CreatedByRole: "workspace", RunID: "run-approval-get"})
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	return createPendingApprovalFromPolicyDecision(t, s, "run-approval-get", "", ref.Digest)
}

func TestHandleApprovalGetRejectsInFlightLimit(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{Limits: Limits{MaxInFlightPerClient: 1, MaxInFlightPerLane: 1}})
	release, err := s.apiInflight.acquire("client-a", "lane-a")
	if err != nil {
		t.Fatalf("acquire precondition returned error: %v", err)
	}
	defer release()
	_, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get-limit", ApprovalID: "sha256:" + strings.Repeat("a", 64)}, RequestContext{ClientID: "client-a", LaneID: "lane-a"})
	if errResp == nil || errResp.Error.Code != "broker_limit_in_flight_exceeded" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestHandleApprovalGetRejectsDeadlineExceeded(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	deadline := time.Now().Add(-time.Second)
	_, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get-timeout", ApprovalID: "sha256:" + strings.Repeat("a", 64)}, RequestContext{Deadline: &deadline})
	if errResp == nil || errResp.Error.Code != "broker_timeout_request_deadline_exceeded" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestHandleApprovalGetUsesNotFoundApprovalCode(t *testing.T) {
	s := newBrokerAPIServiceForTests(t, APIConfig{})
	_, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get-missing", ApprovalID: "sha256:" + strings.Repeat("f", 64)}, RequestContext{})
	if errResp == nil || errResp.Error.Code != "broker_not_found_approval" {
		t.Fatalf("unexpected error response: %+v", errResp)
	}
}

func TestApprovalGetLifecycleDetailForTerminalStates(t *testing.T) {
	tests := []struct {
		name             string
		status           string
		wantState        string
		wantReason       string
		wantStale        bool
		wantStaleReason  string
		supersededBy     string
		wantSupersededBy string
	}{
		{name: "approved", status: "approved", wantState: "approved", wantReason: "approval_approved", wantStale: false},
		{name: "denied", status: "denied", wantState: "denied", wantReason: "approval_denied", wantStale: false},
		{name: "consumed", status: "consumed", wantState: "consumed", wantReason: "approval_consumed", wantStale: false},
		{name: "expired", status: "expired", wantState: "expired", wantReason: "approval_expired", wantStale: true, wantStaleReason: "approval_expired"},
		{name: "superseded", status: "superseded", wantState: "superseded", wantReason: "approval_superseded", wantStale: true, wantStaleReason: "approval_superseded", supersededBy: "sha256:" + strings.Repeat("d", 64), wantSupersededBy: "sha256:" + strings.Repeat("d", 64)},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			s, requestEnv, _ := setupServiceWithStageSignOffApprovalFixture(t)
			approvalID, err := approvalIDFromRequest(*requestEnv)
			if err != nil {
				t.Fatalf("approvalIDFromRequest returned error: %v", err)
			}
			record, ok := s.ApprovalGet(approvalID)
			if !ok {
				t.Fatalf("ApprovalGet(%q) missing", approvalID)
			}
			now := time.Now().UTC()
			record.Status = tc.status
			if tc.status != "pending" {
				record.DecidedAt = &now
			}
			if tc.status == "consumed" {
				record.ConsumedAt = &now
			}
			record.SupersededByApprovalID = tc.supersededBy
			if err := s.RecordApproval(record); err != nil {
				t.Fatalf("RecordApproval returned error: %v", err)
			}

			resp, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get-lifecycle-" + tc.name, ApprovalID: approvalID}, RequestContext{})
			if errResp != nil {
				t.Fatalf("HandleApprovalGet error response: %+v", errResp)
			}
			got := resp.ApprovalDetail.LifecycleDetail
			if got.LifecycleState != tc.wantState || got.LifecycleReasonCode != tc.wantReason || got.Stale != tc.wantStale || got.StaleReasonCode != tc.wantStaleReason || got.SupersededByApprovalID != tc.wantSupersededBy {
				t.Fatalf("unexpected lifecycle detail: got=%+v want_state=%q want_reason=%q want_stale=%t want_stale_reason=%q want_superseded_by=%q", got, tc.wantState, tc.wantReason, tc.wantStale, tc.wantStaleReason, tc.wantSupersededBy)
			}
		})
	}
}

func TestApprovalGetLifecycleDetailMarksPendingExpiredAsStale(t *testing.T) {
	s, requestEnv, _ := setupServiceWithStageSignOffApprovalFixture(t)
	approvalID, err := approvalIDFromRequest(*requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	record, ok := s.ApprovalGet(approvalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing", approvalID)
	}
	past := time.Now().UTC().Add(-time.Minute)
	record.ExpiresAt = &past
	if err := s.RecordApproval(record); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}

	resp, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get-pending-expired", ApprovalID: approvalID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalGet error response: %+v", errResp)
	}
	got := resp.ApprovalDetail.LifecycleDetail
	if got.LifecycleState != "pending" || got.LifecycleReasonCode != "approval_pending" || !got.Stale || got.StaleReasonCode != "approval_expired" {
		t.Fatalf("unexpected stale pending lifecycle detail: %+v", got)
	}
}

func TestApprovalGetLifecycleDetailMarksBoundaryExpiryAsStale(t *testing.T) {
	s, requestEnv, _ := setupServiceWithStageSignOffApprovalFixture(t)
	approvalID, err := approvalIDFromRequest(*requestEnv)
	if err != nil {
		t.Fatalf("approvalIDFromRequest returned error: %v", err)
	}
	record, ok := s.ApprovalGet(approvalID)
	if !ok {
		t.Fatalf("ApprovalGet(%q) missing", approvalID)
	}
	fixed := time.Now().UTC().Round(0)
	record.ExpiresAt = &fixed
	if err := s.RecordApproval(record); err != nil {
		t.Fatalf("RecordApproval returned error: %v", err)
	}
	s.now = func() time.Time { return fixed }

	resp, errResp := s.HandleApprovalGet(context.Background(), ApprovalGetRequest{SchemaID: "runecode.protocol.v0.ApprovalGetRequest", SchemaVersion: "0.1.0", RequestID: "req-approval-get-boundary-expired", ApprovalID: approvalID}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleApprovalGet error response: %+v", errResp)
	}
	got := resp.ApprovalDetail.LifecycleDetail
	if !got.Stale || got.StaleReasonCode != "approval_expired" {
		t.Fatalf("unexpected boundary stale lifecycle detail: %+v", got)
	}
}

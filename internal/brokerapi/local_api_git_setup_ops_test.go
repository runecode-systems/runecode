package brokerapi

import (
	"context"
	"testing"
)

func TestHandleGitSetupGetReturnsBrokerOwnedTypedState(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	resp, errResp := service.HandleGitSetupGet(context.Background(), GitSetupGetRequest{SchemaID: "runecode.protocol.v0.GitSetupGetRequest", SchemaVersion: "0.1.0", RequestID: "req-git-setup-get", Provider: "github"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleGitSetupGet error response: %+v", errResp)
	}
	if resp.PolicySurface.ArtifactManagedOnly != true || resp.PolicySurface.DirectMutationSupport {
		t.Fatalf("policy surface = %+v, want artifact-managed-only and direct mutation disabled", resp.PolicySurface)
	}
	if got := resp.AuthPosture.BootstrapMode; got != "browser" {
		t.Fatalf("auth posture bootstrap mode = %q, want browser", got)
	}
	if len(resp.IdentityProfiles) == 0 {
		t.Fatal("identity profiles empty, want broker-owned default profile")
	}
}

func TestHandleGitSetupAuthBootstrapDeviceCodeReturnsPendingState(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	resp, errResp := service.HandleGitSetupAuthBootstrap(context.Background(), GitSetupAuthBootstrapRequest{SchemaID: "runecode.protocol.v0.GitSetupAuthBootstrapRequest", SchemaVersion: "0.1.0", RequestID: "req-git-auth-device", Provider: "github", Mode: "device_code"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleGitSetupAuthBootstrap error response: %+v", errResp)
	}
	if resp.Status != "pending" {
		t.Fatalf("status = %q, want pending", resp.Status)
	}
	if resp.DeviceVerificationURI == "" || resp.DeviceUserCode == "" {
		t.Fatalf("device code response missing verification instructions: %+v", resp)
	}
	if resp.AccountState.Linked {
		t.Fatalf("account state linked = true, want false for pending device code bootstrap")
	}
}

func TestHandleGitSetupAuthBootstrapBrowserReturnsPendingUnlinkedState(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	resp, errResp := service.HandleGitSetupAuthBootstrap(context.Background(), GitSetupAuthBootstrapRequest{SchemaID: "runecode.protocol.v0.GitSetupAuthBootstrapRequest", SchemaVersion: "0.1.0", RequestID: "req-git-auth-browser", Provider: "github", Mode: "browser"}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleGitSetupAuthBootstrap error response: %+v", errResp)
	}
	if resp.Status != "pending" {
		t.Fatalf("status = %q, want pending", resp.Status)
	}
	if resp.AccountState.Linked {
		t.Fatalf("account state linked = true, want false for stubbed browser bootstrap")
	}
	if got := resp.AuthPosture.AuthStatus; got != "not_linked" {
		t.Fatalf("auth posture status = %q, want not_linked", got)
	}
}

func TestHandleGitSetupIdentityUpsertStoresBrokerManagedProfile(t *testing.T) {
	service := newBrokerAPIServiceForTests(t, APIConfig{})
	upsertResp, errResp := service.HandleGitSetupIdentityUpsert(context.Background(), GitSetupIdentityUpsertRequest{SchemaID: "runecode.protocol.v0.GitSetupIdentityUpsertRequest", SchemaVersion: "0.1.0", RequestID: "req-git-upsert", Provider: "github", Profile: GitCommitIdentityProfile{SchemaID: "runecode.protocol.v0.GitCommitIdentityProfile", SchemaVersion: "0.1.0", ProfileID: "work", DisplayName: "Work", AuthorName: "Work Author", AuthorEmail: "work.author@example.invalid", CommitterName: "Work Committer", CommitterEmail: "work.committer@example.invalid", SignoffName: "Work Signoff", SignoffEmail: "work.signoff@example.invalid", DefaultProfile: true}}, RequestContext{})
	if errResp != nil {
		t.Fatalf("HandleGitSetupIdentityUpsert error response: %+v", errResp)
	}
	if upsertResp.ControlPlaneState.DefaultIdentityProfileID != "work" {
		t.Fatalf("default identity profile id = %q, want work", upsertResp.ControlPlaneState.DefaultIdentityProfileID)
	}
	getResp, getErr := service.HandleGitSetupGet(context.Background(), GitSetupGetRequest{SchemaID: "runecode.protocol.v0.GitSetupGetRequest", SchemaVersion: "0.1.0", RequestID: "req-git-after-upsert", Provider: "github"}, RequestContext{})
	if getErr != nil {
		t.Fatalf("HandleGitSetupGet(after upsert) error response: %+v", getErr)
	}
	found := false
	for _, profile := range getResp.IdentityProfiles {
		if profile.ProfileID == "work" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("identity profiles missing upserted profile: %+v", getResp.IdentityProfiles)
	}
}

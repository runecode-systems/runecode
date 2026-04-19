package brokerapi

import (
	"testing"
	"time"
)

func TestProviderSetupLookupActiveIngressRejectsUsedTokenWithDistinctMessage(t *testing.T) {
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	state := newProviderSetupState(func() time.Time { return now })
	state.session["session-used"] = ProviderSetupSession{SetupSessionID: "session-used"}
	state.ingress["token-used"] = providerSetupIngressRecord{Token: "token-used", SetupSessionID: "session-used", ExpiresAt: now.Add(5 * time.Minute), Used: true}

	if _, _, err := state.lookupActiveIngress("token-used", now); err == nil {
		t.Fatal("lookupActiveIngress succeeded for used token, want error")
	} else if got := err.Error(); got != "secret ingress token already used" {
		t.Fatalf("lookupActiveIngress used token message = %q, want secret ingress token already used", got)
	}
}

func TestProviderSetupLookupActiveIngressRejectsExpiredTokenWithDistinctMessage(t *testing.T) {
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	state := newProviderSetupState(func() time.Time { return now })
	state.session["session-expired"] = ProviderSetupSession{SetupSessionID: "session-expired"}
	state.ingress["token-expired"] = providerSetupIngressRecord{Token: "token-expired", SetupSessionID: "session-expired", ExpiresAt: now.Add(-time.Second)}

	if _, _, err := state.lookupActiveIngress("token-expired", now); err == nil {
		t.Fatal("lookupActiveIngress succeeded for expired token, want error")
	} else if got := err.Error(); got != "secret ingress token expired" {
		t.Fatalf("lookupActiveIngress expired token message = %q, want secret ingress token expired", got)
	}
}

func TestNormalizeProviderIngressRequestAllowsSupportedChannels(t *testing.T) {
	channel, field, err := normalizeProviderIngressRequest("", "")
	if err != nil {
		t.Fatalf("normalizeProviderIngressRequest default returned error: %v", err)
	}
	if channel != "cli_stdin" {
		t.Fatalf("default channel = %q, want cli_stdin", channel)
	}
	if field != "api_key" {
		t.Fatalf("default credential field = %q, want api_key", field)
	}

	channel, field, err = normalizeProviderIngressRequest("tui_masked_input", "token")
	if err != nil {
		t.Fatalf("normalizeProviderIngressRequest(tui_masked_input) returned error: %v", err)
	}
	if channel != "tui_masked_input" || field != "token" {
		t.Fatalf("normalized channel/field = %q/%q, want tui_masked_input/token", channel, field)
	}
}

func TestNormalizeProviderIngressRequestRejectsUnsupportedChannel(t *testing.T) {
	if _, _, err := normalizeProviderIngressRequest("environment_variable", "api_key"); err == nil {
		t.Fatal("normalizeProviderIngressRequest accepted forbidden channel")
	} else if got := err.Error(); got != "ingress_channel \"environment_variable\" is unsupported" {
		t.Fatalf("unsupported channel message = %q, want ingress_channel \"environment_variable\" is unsupported", got)
	}
}

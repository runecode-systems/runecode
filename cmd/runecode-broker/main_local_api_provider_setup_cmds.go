package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
	"golang.org/x/term"
)

func handleProviderSetupDirect(args []string, service *brokerapi.Service, stdout io.Writer) error {
	beginReq, err := parseProviderSetupDirectArgs(args)
	if err != nil {
		return err
	}
	if strings.TrimSpace(os.Getenv("RUNE_PROVIDER_API_KEY")) != "" {
		return &usageError{message: "provider-setup-direct forbids secret environment-variable injection; use trusted prompt input"}
	}
	secret, err := readDirectCredentialSecret(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}
	if len(secret) == 0 {
		return &usageError{message: "provider-setup-direct requires non-empty secret input"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	beginReq.RequestID = defaultRequestID()
	beginResp, errResp := api.ProviderSetupSessionBegin(ctx, beginReq)
	if errResp != nil {
		return localAPIError(errResp)
	}
	prepareResp, errResp := api.ProviderSetupSecretIngressPrepare(ctx, brokerapi.ProviderSetupSecretIngressPrepareRequest{
		SchemaID:        "runecode.protocol.v0.ProviderSetupSecretIngressPrepareRequest",
		SchemaVersion:   "0.1.0",
		RequestID:       defaultRequestID(),
		SetupSessionID:  beginResp.SetupSession.SetupSessionID,
		IngressChannel:  "cli_stdin",
		CredentialField: "api_key",
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	resp, errResp := api.ProviderSetupSecretIngressSubmit(ctx, brokerapi.ProviderSetupSecretIngressSubmitRequest{SchemaID: "runecode.protocol.v0.ProviderSetupSecretIngressSubmitRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID(), SecretIngressToken: prepareResp.SecretIngressToken}, secret)
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func parseProviderSetupDirectArgs(args []string) (brokerapi.ProviderSetupSessionBeginRequest, error) {
	fs := flag.NewFlagSet("provider-setup-direct", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	displayLabel := fs.String("display-label", "", "operator-facing provider profile label")
	providerFamily := fs.String("provider-family", "", "provider family identifier")
	adapterKind := fs.String("adapter-kind", "", "adapter kind")
	canonicalHost := fs.String("canonical-host", "", "canonical endpoint host")
	canonicalPathPrefix := fs.String("canonical-path-prefix", "/v1", "canonical endpoint path prefix")
	allowlistedModelIDs := fs.String("allowlisted-model-ids", "", "comma-separated allowlisted model IDs")
	if err := fs.Parse(args); err != nil {
		return brokerapi.ProviderSetupSessionBeginRequest{}, &usageError{message: "provider-setup-direct usage: runecode-broker provider-setup-direct --provider-family family --canonical-host host [--canonical-path-prefix /v1] [--display-label label] [--adapter-kind kind] [--allowlisted-model-ids id1,id2]"}
	}
	if strings.TrimSpace(*providerFamily) == "" {
		return brokerapi.ProviderSetupSessionBeginRequest{}, &usageError{message: "provider-setup-direct requires --provider-family"}
	}
	if strings.TrimSpace(*canonicalHost) == "" {
		return brokerapi.ProviderSetupSessionBeginRequest{}, &usageError{message: "provider-setup-direct requires --canonical-host"}
	}
	return brokerapi.ProviderSetupSessionBeginRequest{
		SchemaID:            "runecode.protocol.v0.ProviderSetupSessionBeginRequest",
		SchemaVersion:       "0.1.0",
		DisplayLabel:        strings.TrimSpace(*displayLabel),
		ProviderFamily:      strings.TrimSpace(*providerFamily),
		AdapterKind:         strings.TrimSpace(*adapterKind),
		CanonicalHost:       strings.TrimSpace(*canonicalHost),
		CanonicalPathPrefix: strings.TrimSpace(*canonicalPathPrefix),
		AllowlistedModelIDs: splitCSVAllowlist(*allowlistedModelIDs),
	}, nil
}

func handleProviderCredentialLeaseIssue(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("provider-credential-lease-issue", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	providerProfileID := fs.String("provider-profile-id", "", "provider profile id")
	runID := fs.String("run-id", "", "run id")
	ttlSeconds := fs.Int("ttl-seconds", 900, "lease ttl seconds")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "provider-credential-lease-issue usage: runecode-broker provider-credential-lease-issue --provider-profile-id id --run-id run-1 [--ttl-seconds 900]"}
	}
	if strings.TrimSpace(*providerProfileID) == "" {
		return &usageError{message: "provider-credential-lease-issue requires --provider-profile-id"}
	}
	if strings.TrimSpace(*runID) == "" {
		return &usageError{message: "provider-credential-lease-issue requires --run-id"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ProviderCredentialLeaseIssue(ctx, brokerapi.ProviderCredentialLeaseIssueRequest{SchemaID: "runecode.protocol.v0.ProviderCredentialLeaseIssueRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID(), ProviderProfileID: strings.TrimSpace(*providerProfileID), RunID: strings.TrimSpace(*runID), TTLSeconds: *ttlSeconds})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleProviderProfileList(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("provider-profile-list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "provider-profile-list usage: runecode-broker provider-profile-list"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ProviderProfileList(ctx, brokerapi.ProviderProfileListRequest{SchemaID: "runecode.protocol.v0.ProviderProfileListRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID()})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleProviderProfileGet(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("provider-profile-get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	providerProfileID := fs.String("provider-profile-id", "", "provider profile id")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "provider-profile-get usage: runecode-broker provider-profile-get --provider-profile-id id"}
	}
	if strings.TrimSpace(*providerProfileID) == "" {
		return &usageError{message: "provider-profile-get requires --provider-profile-id"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ProviderProfileGet(ctx, brokerapi.ProviderProfileGetRequest{SchemaID: "runecode.protocol.v0.ProviderProfileGetRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID(), ProviderProfileID: strings.TrimSpace(*providerProfileID)})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func splitCSVAllowlist(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func readDirectCredentialSecret(stdin io.Reader, stdout io.Writer) ([]byte, error) {
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		_, _ = fmt.Fprint(stdout, "Enter API credential (input hidden): ")
		secret, err := term.ReadPassword(int(f.Fd()))
		_, _ = fmt.Fprintln(stdout)
		if err != nil {
			return nil, err
		}
		return secret, nil
	}
	reader := bufio.NewReader(stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	return []byte(strings.TrimSpace(line)), nil
}

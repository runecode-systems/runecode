package main

import (
	"context"
	"flag"
	"io"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func handleGitSetupGet(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("git-setup-get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	provider := fs.String("provider", "github", "git provider id")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "git-setup-get usage: runecode-broker git-setup-get [--provider github]"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.GitSetupGet(ctx, brokerapi.GitSetupGetRequest{SchemaID: "runecode.protocol.v0.GitSetupGetRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID(), Provider: strings.TrimSpace(*provider)})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleGitSetupAuthBootstrap(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("git-setup-auth-bootstrap", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	provider := fs.String("provider", "github", "git provider id")
	mode := fs.String("mode", "", "auth bootstrap mode (browser|device_code)")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "git-setup-auth-bootstrap usage: runecode-broker git-setup-auth-bootstrap [--provider github] --mode browser|device_code"}
	}
	if strings.TrimSpace(*mode) == "" {
		return &usageError{message: "git-setup-auth-bootstrap requires --mode browser|device_code"}
	}
	if strings.TrimSpace(*mode) == "interactive_token_prompt" {
		return &usageError{message: "manual token fallback must be entered through trusted interactive prompts; use browser or device_code bootstrap mode here"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.GitSetupAuthBootstrap(ctx, brokerapi.GitSetupAuthBootstrapRequest{SchemaID: "runecode.protocol.v0.GitSetupAuthBootstrapRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID(), Provider: strings.TrimSpace(*provider), Mode: strings.TrimSpace(*mode)})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleGitSetupIdentityUpsert(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("git-setup-identity-upsert", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	provider := fs.String("provider", "github", "git provider id")
	profileID := fs.String("profile-id", "", "identity profile id")
	displayName := fs.String("display-name", "", "profile display name")
	authorName := fs.String("author-name", "", "git author name")
	authorEmail := fs.String("author-email", "", "git author email")
	committerName := fs.String("committer-name", "", "git committer name")
	committerEmail := fs.String("committer-email", "", "git committer email")
	signoffName := fs.String("signoff-name", "", "git signoff name")
	signoffEmail := fs.String("signoff-email", "", "git signoff email")
	defaultProfile := fs.Bool("default-profile", false, "mark profile as default")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "git-setup-identity-upsert usage: runecode-broker git-setup-identity-upsert [--provider github] --profile-id id --display-name name --author-name name --author-email mail --committer-name name --committer-email mail --signoff-name name --signoff-email mail [--default-profile]"}
	}
	required := map[string]string{"--profile-id": strings.TrimSpace(*profileID), "--author-name": strings.TrimSpace(*authorName), "--author-email": strings.TrimSpace(*authorEmail), "--committer-name": strings.TrimSpace(*committerName), "--committer-email": strings.TrimSpace(*committerEmail), "--signoff-name": strings.TrimSpace(*signoffName), "--signoff-email": strings.TrimSpace(*signoffEmail)}
	for flagName, value := range required {
		if value == "" {
			return &usageError{message: "git-setup-identity-upsert requires " + flagName}
		}
	}
	resolvedDisplay := strings.TrimSpace(*displayName)
	if resolvedDisplay == "" {
		resolvedDisplay = strings.TrimSpace(*profileID)
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.GitSetupIdentityUpsert(ctx, brokerapi.GitSetupIdentityUpsertRequest{SchemaID: "runecode.protocol.v0.GitSetupIdentityUpsertRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID(), Provider: strings.TrimSpace(*provider), Profile: brokerapi.GitCommitIdentityProfile{SchemaID: "runecode.protocol.v0.GitCommitIdentityProfile", SchemaVersion: "0.1.0", ProfileID: strings.TrimSpace(*profileID), DisplayName: resolvedDisplay, AuthorName: strings.TrimSpace(*authorName), AuthorEmail: strings.TrimSpace(*authorEmail), CommitterName: strings.TrimSpace(*committerName), CommitterEmail: strings.TrimSpace(*committerEmail), SignoffName: strings.TrimSpace(*signoffName), SignoffEmail: strings.TrimSpace(*signoffEmail), DefaultProfile: *defaultProfile}})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

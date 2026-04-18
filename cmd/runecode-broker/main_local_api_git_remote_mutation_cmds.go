package main

import (
	"context"
	"flag"
	"io"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func handleGitRemoteMutationPrepare(args []string, service *brokerapi.Service, stdout io.Writer) error {
	requestFile, err := parseGitRemoteMutationRequestFile("git-remote-mutation-prepare", args)
	if err != nil {
		return err
	}
	request := brokerapi.GitRemoteMutationPrepareRequest{}
	if err := loadStrictJSONFileValue(requestFile, &request); err != nil {
		return err
	}
	request.SchemaID = "runecode.protocol.v0.GitRemoteMutationPrepareRequest"
	request.SchemaVersion = "0.1.0"
	request.RequestID = defaultRequestID()
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.GitRemoteMutationPrepare(ctx, request)
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleGitRemoteMutationGet(args []string, service *brokerapi.Service, stdout io.Writer) error {
	requestFile, err := parseGitRemoteMutationRequestFile("git-remote-mutation-get", args)
	if err != nil {
		return err
	}
	request := brokerapi.GitRemoteMutationGetRequest{}
	if err := loadStrictJSONFileValue(requestFile, &request); err != nil {
		return err
	}
	request.SchemaID = "runecode.protocol.v0.GitRemoteMutationGetRequest"
	request.SchemaVersion = "0.1.0"
	request.RequestID = defaultRequestID()
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.GitRemoteMutationGet(ctx, request)
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleGitRemoteMutationExecute(args []string, service *brokerapi.Service, stdout io.Writer) error {
	requestFile, err := parseGitRemoteMutationRequestFile("git-remote-mutation-execute", args)
	if err != nil {
		return err
	}
	request := brokerapi.GitRemoteMutationExecuteRequest{}
	if err := loadStrictJSONFileValue(requestFile, &request); err != nil {
		return err
	}
	request.SchemaID = "runecode.protocol.v0.GitRemoteMutationExecuteRequest"
	request.SchemaVersion = "0.1.0"
	request.RequestID = defaultRequestID()
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.GitRemoteMutationExecute(ctx, request)
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleGitRemoteMutationIssueExecuteLease(args []string, service *brokerapi.Service, stdout io.Writer) error {
	requestFile, err := parseGitRemoteMutationRequestFile("git-remote-mutation-issue-execute-lease", args)
	if err != nil {
		return err
	}
	request := brokerapi.GitRemoteMutationIssueExecuteLeaseRequest{}
	if err := loadStrictJSONFileValue(requestFile, &request); err != nil {
		return err
	}
	request.SchemaID = "runecode.protocol.v0.GitRemoteMutationIssueExecuteLeaseRequest"
	request.SchemaVersion = "0.1.0"
	request.RequestID = defaultRequestID()
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.GitRemoteMutationIssueExecuteLease(ctx, request)
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func parseGitRemoteMutationRequestFile(command string, args []string) (string, error) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	requestFile := fs.String("request-file", "", "path to typed request JSON")
	if err := fs.Parse(args); err != nil {
		return "", &usageError{message: command + " usage: runecode-broker " + command + " --request-file path"}
	}
	resolved := strings.TrimSpace(*requestFile)
	if resolved == "" {
		return "", &usageError{message: command + " requires --request-file"}
	}
	return resolved, nil
}

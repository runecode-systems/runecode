package main

import (
	"context"
	"io"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func handleExternalAnchorMutationPrepare(args []string, service *brokerapi.Service, stdout io.Writer) error {
	requestFile, err := parseGitRemoteMutationRequestFile("external-anchor-mutation-prepare", args)
	if err != nil {
		return err
	}
	request := brokerapi.ExternalAnchorMutationPrepareRequest{}
	if err := loadStrictJSONFileValue(requestFile, &request); err != nil {
		return err
	}
	request.SchemaID = "runecode.protocol.v0.ExternalAnchorMutationPrepareRequest"
	request.SchemaVersion = "0.1.0"
	request.RequestID = defaultRequestID()
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ExternalAnchorMutationPrepare(ctx, request)
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleExternalAnchorMutationGet(args []string, service *brokerapi.Service, stdout io.Writer) error {
	requestFile, err := parseGitRemoteMutationRequestFile("external-anchor-mutation-get", args)
	if err != nil {
		return err
	}
	request := brokerapi.ExternalAnchorMutationGetRequest{}
	if err := loadStrictJSONFileValue(requestFile, &request); err != nil {
		return err
	}
	request.SchemaID = "runecode.protocol.v0.ExternalAnchorMutationGetRequest"
	request.SchemaVersion = "0.1.0"
	request.RequestID = defaultRequestID()
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ExternalAnchorMutationGet(ctx, request)
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleExternalAnchorMutationExecute(args []string, service *brokerapi.Service, stdout io.Writer) error {
	requestFile, err := parseGitRemoteMutationRequestFile("external-anchor-mutation-execute", args)
	if err != nil {
		return err
	}
	request := brokerapi.ExternalAnchorMutationExecuteRequest{}
	if err := loadStrictJSONFileValue(requestFile, &request); err != nil {
		return err
	}
	request.SchemaID = "runecode.protocol.v0.ExternalAnchorMutationExecuteRequest"
	request.SchemaVersion = "0.1.0"
	request.RequestID = defaultRequestID()
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ExternalAnchorMutationExecute(ctx, request)
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

package main

import (
	"context"
	"flag"
	"io"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func handleDependencyCacheEnsure(args []string, service *brokerapi.Service, stdout io.Writer) error {
	requestFile, err := parseDependencyRequestFile("dependency-cache-ensure", args)
	if err != nil {
		return err
	}
	request := brokerapi.DependencyCacheEnsureRequest{}
	if err := loadStrictJSONFileValue(requestFile, &request); err != nil {
		return err
	}
	request.SchemaID = "runecode.protocol.v0.DependencyCacheEnsureRequest"
	request.SchemaVersion = "0.1.0"
	request.RequestID = defaultRequestID()
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.DependencyCacheEnsure(ctx, request)
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleDependencyFetchRegistry(args []string, service *brokerapi.Service, stdout io.Writer) error {
	requestFile, err := parseDependencyRequestFile("dependency-fetch-registry", args)
	if err != nil {
		return err
	}
	request := brokerapi.DependencyFetchRegistryRequest{}
	if err := loadStrictJSONFileValue(requestFile, &request); err != nil {
		return err
	}
	request.SchemaID = "runecode.protocol.v0.DependencyFetchRegistryRequest"
	request.SchemaVersion = "0.1.0"
	request.RequestID = defaultRequestID()
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.DependencyFetchRegistry(ctx, request)
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleDependencyCacheHandoff(args []string, service *brokerapi.Service, stdout io.Writer) error {
	requestFile, err := parseDependencyRequestFile("dependency-cache-handoff", args)
	if err != nil {
		return err
	}
	request := brokerapi.DependencyCacheHandoffRequest{}
	if err := loadStrictJSONFileValue(requestFile, &request); err != nil {
		return err
	}
	request.SchemaID = "runecode.protocol.v0.DependencyCacheHandoffRequest"
	request.SchemaVersion = "0.1.0"
	request.RequestID = defaultRequestID()
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.DependencyCacheHandoff(ctx, request)
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func parseDependencyRequestFile(command string, args []string) (string, error) {
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

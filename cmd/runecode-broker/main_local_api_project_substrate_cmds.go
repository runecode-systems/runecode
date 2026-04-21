package main

import (
	"context"
	"flag"
	"io"
	"strings"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func handleProjectSubstrateGet(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("project-substrate-get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "project-substrate-get usage: runecode-broker project-substrate-get"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ProjectSubstrateGet(ctx, brokerapi.ProjectSubstrateGetRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateGetRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID()})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleProjectSubstratePostureGet(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("project-substrate-posture-get", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "project-substrate-posture-get usage: runecode-broker project-substrate-posture-get"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ProjectSubstratePostureGet(ctx, brokerapi.ProjectSubstratePostureGetRequest{SchemaID: "runecode.protocol.v0.ProjectSubstratePostureGetRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID()})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleProjectSubstrateAdopt(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("project-substrate-adopt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "project-substrate-adopt usage: runecode-broker project-substrate-adopt"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ProjectSubstrateAdopt(ctx, brokerapi.ProjectSubstrateAdoptRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateAdoptRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID()})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleProjectSubstrateInitPreview(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("project-substrate-init-preview", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "project-substrate-init-preview usage: runecode-broker project-substrate-init-preview"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ProjectSubstrateInitPreview(ctx, brokerapi.ProjectSubstrateInitPreviewRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateInitPreviewRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID()})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleProjectSubstrateInitApply(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("project-substrate-init-apply", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	expectedToken := fs.String("expected-preview-token", "", "expected preview token from init preview")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "project-substrate-init-apply usage: runecode-broker project-substrate-init-apply [--expected-preview-token sha256:...]"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ProjectSubstrateInitApply(ctx, brokerapi.ProjectSubstrateInitApplyRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateInitApplyRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID(), ExpectedPreviewToken: strings.TrimSpace(*expectedToken)})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleProjectSubstrateUpgradePreview(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("project-substrate-upgrade-preview", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "project-substrate-upgrade-preview usage: runecode-broker project-substrate-upgrade-preview"}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ProjectSubstrateUpgradePreview(ctx, brokerapi.ProjectSubstrateUpgradePreviewRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateUpgradePreviewRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID()})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleProjectSubstrateUpgradeApply(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("project-substrate-upgrade-apply", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	expectedDigest := fs.String("expected-preview-digest", "", "expected preview digest from upgrade preview")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "project-substrate-upgrade-apply usage: runecode-broker project-substrate-upgrade-apply --expected-preview-digest sha256:..."}
	}
	if strings.TrimSpace(*expectedDigest) == "" {
		return &usageError{message: "project-substrate-upgrade-apply usage: runecode-broker project-substrate-upgrade-apply --expected-preview-digest sha256:..."}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ProjectSubstrateUpgradeApply(ctx, brokerapi.ProjectSubstrateUpgradeApplyRequest{SchemaID: "runecode.protocol.v0.ProjectSubstrateUpgradeApplyRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID(), ExpectedPreviewDigest: strings.TrimSpace(*expectedDigest)})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

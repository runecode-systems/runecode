package main

import (
	"context"
	"flag"
	"io"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func handleZKProofGenerate(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("zk-proof-generate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	recordDigest := fs.String("record-digest", "", "audit record digest")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "zk-proof-generate usage: runecode-broker zk-proof-generate --record-digest sha256:..."}
	}
	digest, err := parseDigestFlag(*recordDigest, "--record-digest")
	if err != nil {
		return &usageError{message: "zk-proof-generate " + err.Error()}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ZKProofGenerate(ctx, brokerapi.ZKProofGenerateRequest{
		SchemaID:      "runecode.protocol.v0.ZKProofGenerateRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		RecordDigest:  digest,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

func handleZKProofVerify(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("zk-proof-verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	proofDigest := fs.String("proof-digest", "", "zk proof artifact digest")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "zk-proof-verify usage: runecode-broker zk-proof-verify --proof-digest sha256:..."}
	}
	digest, err := parseDigestFlag(*proofDigest, "--proof-digest")
	if err != nil {
		return &usageError{message: "zk-proof-verify " + err.Error()}
	}
	api := localAPIForService(service)
	ctx, cancel := commandRequestContext(context.Background())
	defer cancel()
	resp, errResp := api.ZKProofVerify(ctx, brokerapi.ZKProofVerifyRequest{
		SchemaID:              "runecode.protocol.v0.ZKProofVerifyRequest",
		SchemaVersion:         "0.1.0",
		RequestID:             defaultRequestID(),
		ZKProofArtifactDigest: digest,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp)
}

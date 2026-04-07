package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func handleListArtifacts(_ []string, service *brokerapi.Service, stdout io.Writer) error {
	api := localAPIForService(service)
	resp, errResp := api.ArtifactList(context.Background(), brokerapi.LocalArtifactListRequest{
		SchemaID:      "runecode.protocol.v0.ArtifactListRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	artifactsOut := make([]artifacts.ArtifactReference, 0, len(resp.Artifacts))
	for _, artifact := range resp.Artifacts {
		artifactsOut = append(artifactsOut, artifact.Reference)
	}
	return writeJSON(stdout, artifactsOut)
}

func handleHeadArtifact(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("head-artifact", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	digest := fs.String("digest", "", "artifact digest")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "head-artifact usage: runecode-broker head-artifact --digest sha256:..."}
	}
	if *digest == "" {
		return &usageError{message: "head-artifact requires --digest"}
	}
	api := localAPIForService(service)
	resp, errResp := api.ArtifactHead(context.Background(), brokerapi.LocalArtifactHeadRequest{
		SchemaID:      "runecode.protocol.v0.ArtifactHeadRequest",
		SchemaVersion: "0.1.0",
		RequestID:     defaultRequestID(),
		Digest:        *digest,
	})
	if errResp != nil {
		return localAPIError(errResp)
	}
	return writeJSON(stdout, resp.Artifact.Reference)
}

type getArtifactOptions struct {
	digest        string
	producer      string
	consumer      string
	manifestOptIn bool
	dataClass     string
	out           string
}

func parseGetArtifactArgs(args []string) (getArtifactOptions, error) {
	fs := flag.NewFlagSet("get-artifact", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	digest := fs.String("digest", "", "artifact digest")
	producer := fs.String("producer", "", "producer role for flow check")
	consumer := fs.String("consumer", "", "consumer role for flow check")
	manifestOptIn := fs.Bool("manifest-opt-in", false, "manifest opt-in posture for approved excerpts")
	dataClass := fs.String("data-class", "", "optional expected data class")
	out := fs.String("out", "", "output file path")
	if err := fs.Parse(args); err != nil {
		return getArtifactOptions{}, &usageError{message: "get-artifact usage: runecode-broker get-artifact --digest sha256:... --producer role --consumer role [--manifest-opt-in] [--data-class class] --out path"}
	}
	if *digest == "" || *producer == "" || *consumer == "" || *out == "" {
		return getArtifactOptions{}, &usageError{message: "get-artifact requires --digest --producer --consumer and --out"}
	}
	return getArtifactOptions{digest: *digest, producer: *producer, consumer: *consumer, manifestOptIn: *manifestOptIn, dataClass: *dataClass, out: *out}, nil
}

func (o getArtifactOptions) toRequest() brokerapi.ArtifactReadRequest {
	return brokerapi.ArtifactReadRequest{SchemaID: "runecode.protocol.v0.ArtifactReadRequest", SchemaVersion: "0.1.0", RequestID: defaultRequestID(), Digest: o.digest, ProducerRole: o.producer, ConsumerRole: o.consumer, ManifestOptIn: o.manifestOptIn, DataClass: o.dataClass}
}

func handleGetArtifact(args []string, service *brokerapi.Service, stdout io.Writer) error {
	opts, err := parseGetArtifactArgs(args)
	if err != nil {
		return err
	}
	api := localAPIForService(service)
	events, errResp := api.ArtifactRead(context.Background(), opts.toRequest())
	if errResp != nil {
		return localAPIError(errResp)
	}
	written, err := writeArtifactEventsToFile(events, opts.out)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "wrote %d bytes to %s\n", written, opts.out)
	return err
}

func writeArtifactEventsToFile(events []brokerapi.ArtifactStreamEvent, outPath string) (int64, error) {
	tmpPath := outPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return 0, err
	}
	defer os.Remove(tmpPath)
	var written int64
	sawCompletedTerminal := false
	for _, event := range events {
		n, completedTerminal, processErr := processArtifactFileEvent(f, event)
		if processErr != nil {
			_ = f.Close()
			return 0, processErr
		}
		written += int64(n)
		sawCompletedTerminal = sawCompletedTerminal || completedTerminal
	}
	if !sawCompletedTerminal {
		_ = f.Close()
		return 0, fmt.Errorf("artifact stream did not complete successfully")
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return 0, err
	}
	if err := f.Close(); err != nil {
		return 0, err
	}
	if err := replaceFile(tmpPath, outPath); err != nil {
		return 0, err
	}
	return written, nil
}

func processArtifactFileEvent(f *os.File, event brokerapi.ArtifactStreamEvent) (int, bool, error) {
	switch event.EventType {
	case "artifact_stream_chunk":
		chunk, err := base64.StdEncoding.DecodeString(event.ChunkBase64)
		if err != nil {
			return 0, false, err
		}
		n, err := f.Write(chunk)
		return n, false, err
	case "artifact_stream_terminal":
		if event.Error != nil {
			return 0, false, fmt.Errorf("%s: %s", event.Error.Code, event.Error.Message)
		}
		if event.TerminalStatus != "completed" {
			return 0, false, fmt.Errorf("artifact stream terminal status %q is not completed", event.TerminalStatus)
		}
		return 0, true, nil
	}
	return 0, false, nil
}

func replaceFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if removeErr := os.Remove(dst); removeErr != nil && !os.IsNotExist(removeErr) {
		return removeErr
	}
	return os.Rename(src, dst)
}

func handlePutArtifact(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("put-artifact", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	filePath := fs.String("file", "", "artifact payload file")
	contentType := fs.String("content-type", "application/octet-stream", "artifact content type")
	dataClass := fs.String("data-class", string(artifacts.DataClassSpecText), "artifact data class")
	provenance := fs.String("provenance-hash", "", "provenance receipt hash")
	role := fs.String("role", "workspace", "producer role")
	runID := fs.String("run-id", "", "run id")
	stepID := fs.String("step-id", "", "step id")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "put-artifact usage: runecode-broker put-artifact --file path --content-type text/plain --data-class spec_text --provenance-hash sha256:..."}
	}
	if *filePath == "" || *provenance == "" {
		return &usageError{message: "put-artifact requires --file and --provenance-hash"}
	}
	payload, err := os.ReadFile(*filePath)
	if err != nil {
		return err
	}
	request := brokerapi.DefaultArtifactPutRequest(defaultRequestID(), payload, *contentType, *dataClass, *provenance, *role, *runID, *stepID)
	resp, errResp := service.HandleArtifactPut(context.Background(), request, brokerapi.RequestContext{})
	if errResp != nil {
		return fmt.Errorf("%s: %s", errResp.Error.Code, errResp.Error.Message)
	}
	return writeJSON(stdout, resp.Artifact)
}

func handleCheckFlow(args []string, service *brokerapi.Service, stdout io.Writer) error {
	fs := flag.NewFlagSet("check-flow", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	producer := fs.String("producer", "", "producer role")
	consumer := fs.String("consumer", "", "consumer role")
	dataClass := fs.String("data-class", "", "data class")
	digest := fs.String("digest", "", "digest")
	isEgress := fs.Bool("egress", false, "egress flow")
	manifestOptIn := fs.Bool("manifest-opt-in", false, "manifest opted in for approved excerpts")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: "check-flow usage: runecode-broker check-flow --producer workspace --consumer model_gateway --data-class spec_text --digest sha256:... [--egress] [--manifest-opt-in]"}
	}
	if *producer == "" || *consumer == "" || *dataClass == "" || *digest == "" {
		return &usageError{message: "check-flow requires --producer --consumer --data-class --digest"}
	}
	class, err := brokerapi.ParseDataClass(*dataClass)
	if err != nil {
		return &usageError{message: err.Error()}
	}
	if err := service.CheckFlow(artifacts.FlowCheckRequest{ProducerRole: *producer, ConsumerRole: *consumer, DataClass: class, Digest: *digest, IsEgress: *isEgress, ManifestOptIn: *manifestOptIn}); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "allowed")
	return err
}

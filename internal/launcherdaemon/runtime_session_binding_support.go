package launcherdaemon

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

type runtimeSessionBinding struct {
	IsolateID                string
	SessionID                string
	SessionNonce             string
	LaunchContextDigest      string
	HandshakeTranscriptHash  string
	IsolateSessionKeyIDValue string
}

func deriveRuntimeSessionBinding(spec launcherbackend.BackendLaunchSpec, runtimeImageDescriptorDigest, isolateID, sessionID, nonce string) (runtimeSessionBinding, error) {
	runID := strings.TrimSpace(spec.RunID)
	stageID := strings.TrimSpace(spec.StageID)
	roleInstanceID := strings.TrimSpace(spec.RoleInstanceID)
	descriptorDigest := strings.TrimSpace(runtimeImageDescriptorDigest)
	trimmedIsolateID := strings.TrimSpace(isolateID)
	trimmedSessionID := strings.TrimSpace(sessionID)
	trimmedNonce := strings.TrimSpace(nonce)

	switch {
	case runID == "":
		return runtimeSessionBinding{}, fmt.Errorf("session binding requires run id")
	case stageID == "":
		return runtimeSessionBinding{}, fmt.Errorf("session binding requires stage id")
	case roleInstanceID == "":
		return runtimeSessionBinding{}, fmt.Errorf("session binding requires role instance id")
	case descriptorDigest == "":
		return runtimeSessionBinding{}, fmt.Errorf("session binding requires runtime image descriptor digest")
	case trimmedIsolateID == "":
		return runtimeSessionBinding{}, fmt.Errorf("session binding requires isolate id")
	case trimmedSessionID == "":
		return runtimeSessionBinding{}, fmt.Errorf("session binding requires session id")
	case trimmedNonce == "":
		return runtimeSessionBinding{}, fmt.Errorf("session binding requires session nonce")
	}

	return runtimeSessionBinding{
		IsolateID:                trimmedIsolateID,
		SessionID:                trimmedSessionID,
		SessionNonce:             trimmedNonce,
		LaunchContextDigest:      syntheticDigest("launch-context", runID, stageID, roleInstanceID, trimmedIsolateID, trimmedSessionID, trimmedNonce),
		HandshakeTranscriptHash:  syntheticDigest("handshake", runID, stageID, roleInstanceID, trimmedIsolateID, trimmedSessionID, trimmedNonce, descriptorDigest),
		IsolateSessionKeyIDValue: syntheticHashHex("session-key", runID, stageID, roleInstanceID, trimmedIsolateID, trimmedSessionID, trimmedNonce, descriptorDigest),
	}, nil
}

func syntheticDigest(domain string, parts ...string) string {
	sum := sha256.Sum256(syntheticHashInput(domain, parts...))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func syntheticHashHex(domain string, parts ...string) string {
	sum := sha256.Sum256(syntheticHashInput(domain, parts...))
	return hex.EncodeToString(sum[:])
}

func syntheticHashInput(domain string, parts ...string) []byte {
	var payload bytes.Buffer
	appendSyntheticHashField(&payload, domain)
	for _, part := range parts {
		appendSyntheticHashField(&payload, part)
	}
	return payload.Bytes()
}

func appendSyntheticHashField(payload *bytes.Buffer, value string) {
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(value)))
	payload.Write(size[:])
	payload.WriteString(value)
}

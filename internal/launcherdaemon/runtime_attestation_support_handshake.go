package launcherdaemon

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"

	"github.com/runecode-ai/runecode/internal/launcherbackend"
)

type runtimeSecureSessionHandshakeTuple struct {
	launchContext launcherbackend.LaunchContext
	host          launcherbackend.HostHello
	isolate       launcherbackend.IsolateHello
	ready         launcherbackend.SessionReady
}

func secureSessionHandshakeTuple(spec launcherbackend.BackendLaunchSpec, receipt launcherbackend.BackendLaunchReceipt) (runtimeSecureSessionHandshakeTuple, string, error) {
	isolateID, sessionID, sessionNonce, err := secureSessionBindingTuple(receipt)
	if err != nil {
		return runtimeSecureSessionHandshakeTuple{}, "", err
	}
	launchContext, launchContextDigest, err := secureSessionLaunchContext(spec, sessionID, sessionNonce)
	if err != nil {
		return runtimeSecureSessionHandshakeTuple{}, "", err
	}
	host := secureSessionHostHello(spec, receipt, isolateID, sessionID, sessionNonce, launchContextDigest)
	isolate, keyIDValue, privateKey := secureSessionIsolateHello(spec, receipt, isolateID, sessionID, sessionNonce, launchContextDigest)
	handshakeTranscriptHash, err := signSecureSessionProof(&isolate, host, privateKey)
	if err != nil {
		return runtimeSecureSessionHandshakeTuple{}, "", err
	}
	ready := secureSessionReady(spec, isolateID, sessionID, sessionNonce, keyIDValue, handshakeTranscriptHash)
	return runtimeSecureSessionHandshakeTuple{
		launchContext: launchContext,
		host:          host,
		isolate:       isolate,
		ready:         ready,
	}, launchContextDigest, nil
}

func secureSessionBindingTuple(receipt launcherbackend.BackendLaunchReceipt) (string, string, string, error) {
	isolateID := receipt.IsolateID
	sessionID := receipt.SessionID
	sessionNonce := receipt.SessionNonce
	if isolateID == "" || sessionID == "" || sessionNonce == "" {
		return "", "", "", fmt.Errorf("session binding is required before secure session validation")
	}
	return isolateID, sessionID, sessionNonce, nil
}

func secureSessionLaunchContext(spec launcherbackend.BackendLaunchSpec, sessionID, sessionNonce string) (launcherbackend.LaunchContext, string, error) {
	launchContext := launcherbackend.LaunchContext{
		RunID:          spec.RunID,
		StageID:        spec.StageID,
		RoleInstanceID: spec.RoleInstanceID,
		SessionID:      sessionID,
		SessionNonce:   sessionNonce,
	}
	launchContextDigest, err := launchContext.CanonicalDigest()
	if err != nil {
		return launcherbackend.LaunchContext{}, "", err
	}
	launchContext.LaunchContextDigest = launchContextDigest
	return launchContext, launchContextDigest, nil
}

func secureSessionHostHello(spec launcherbackend.BackendLaunchSpec, receipt launcherbackend.BackendLaunchReceipt, isolateID, sessionID, sessionNonce, launchContextDigest string) launcherbackend.HostHello {
	return launcherbackend.HostHello{
		RunID:               spec.RunID,
		StageID:             spec.StageID,
		RoleInstanceID:      spec.RoleInstanceID,
		IsolateID:           isolateID,
		SessionID:           sessionID,
		SessionNonce:        sessionNonce,
		LaunchContextDigest: launchContextDigest,
		TransportKind:       secureSessionTransportKind(receipt.TransportKind),
		TransportRequirements: launcherbackend.SessionTransportRequirements{
			MutualAuthenticationRequired: true,
			EncryptionRequired:           true,
			ReplayProtectionRequired:     true,
		},
		Framing: launcherbackend.SessionFramingContract{
			FrameFormat:              launcherbackend.SessionFramingLengthPrefixedV1,
			MaxFrameBytes:            launcherbackend.SessionMaxFrameBytesHardLimit,
			MaxHandshakeMessageBytes: launcherbackend.SessionMaxHandshakeMessageBytesHardLimit,
		},
	}
}

func secureSessionIsolateHello(spec launcherbackend.BackendLaunchSpec, receipt launcherbackend.BackendLaunchReceipt, isolateID, sessionID, sessionNonce, launchContextDigest string) (launcherbackend.IsolateHello, string, ed25519.PrivateKey) {
	keyIDValue, publicKey, privateKey := deriveSyntheticSecureSessionKeyPair(spec, receipt)
	return launcherbackend.IsolateHello{
		RunID:               spec.RunID,
		IsolateID:           isolateID,
		SessionID:           sessionID,
		SessionNonce:        sessionNonce,
		LaunchContextDigest: launchContextDigest,
		IsolateSessionKey: launcherbackend.IsolateSessionKey{
			Alg:               "ed25519",
			KeyID:             "runtime-session-key",
			KeyIDValue:        keyIDValue,
			PublicKeyEncoding: "base64",
			PublicKey:         base64.StdEncoding.EncodeToString(publicKey),
			KeyOrigin:         launcherbackend.SessionKeyOriginIsolateBoundaryEphemeral,
		},
		ProofOfPossession: launcherbackend.SessionKeyProof{
			Alg:        "ed25519",
			KeyID:      "runtime-session-key",
			KeyIDValue: keyIDValue,
		},
	}, keyIDValue, privateKey
}

func signSecureSessionProof(isolate *launcherbackend.IsolateHello, host launcherbackend.HostHello, privateKey ed25519.PrivateKey) (string, error) {
	handshakeTranscriptHash, err := launcherbackend.HandshakeTranscriptHash(host, *isolate)
	if err != nil {
		return "", err
	}
	isolate.HandshakeTranscriptHash = handshakeTranscriptHash
	payload, err := secureSessionProofPayload(host, *isolate, handshakeTranscriptHash)
	if err != nil {
		return "", err
	}
	isolate.ProofOfPossession.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	return handshakeTranscriptHash, nil
}

func secureSessionReady(spec launcherbackend.BackendLaunchSpec, isolateID, sessionID, sessionNonce, keyIDValue, handshakeTranscriptHash string) launcherbackend.SessionReady {
	return launcherbackend.SessionReady{
		RunID:                     spec.RunID,
		IsolateID:                 isolateID,
		SessionID:                 sessionID,
		SessionNonce:              sessionNonce,
		ProvisioningMode:          launcherbackend.ProvisioningPostureTOFU,
		IdentityBindingPosture:    launcherbackend.ProvisioningPostureTOFU,
		IsolateKeyIDValue:         keyIDValue,
		HandshakeTranscriptHash:   handshakeTranscriptHash,
		ChannelKeyMode:            launcherbackend.SessionChannelKeyModeDistinct,
		MutuallyAuthenticated:     true,
		Encrypted:                 true,
		ProofOfPossessionVerified: true,
	}
}

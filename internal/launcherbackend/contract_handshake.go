package launcherbackend

import (
	"fmt"
)

func ValidateSessionHandshake(launchContext LaunchContext, host HostHello, isolate IsolateHello, ready SessionReady, previous *SessionBindingRecord) (SessionBindingRecord, error) {
	if err := launchContext.Validate(); err != nil {
		return SessionBindingRecord{}, fmt.Errorf("launch_context: %w", err)
	}
	if err := host.Validate(); err != nil {
		return SessionBindingRecord{}, fmt.Errorf("host_hello: %w", err)
	}
	if err := isolate.Validate(); err != nil {
		return SessionBindingRecord{}, fmt.Errorf("isolate_hello: %w", err)
	}
	if err := ready.Validate(); err != nil {
		return SessionBindingRecord{}, fmt.Errorf("session_ready: %w", err)
	}
	if err := validateHandshakeLaunchContextAlignment(launchContext, host, isolate); err != nil {
		return SessionBindingRecord{}, err
	}
	if err := validateHandshakeTupleAlignment(host, isolate, ready); err != nil {
		return SessionBindingRecord{}, err
	}
	transcriptHash, err := HandshakeTranscriptHash(host, isolate)
	if err != nil {
		return SessionBindingRecord{}, err
	}
	if err := validateHandshakeTranscriptConsistency(isolate, ready, transcriptHash); err != nil {
		return SessionBindingRecord{}, err
	}
	if err := validateHandshakeReplayAndIdentity(previous, host, isolate, transcriptHash); err != nil {
		return SessionBindingRecord{}, err
	}
	return buildSessionBindingRecord(host, isolate, ready, transcriptHash), nil
}

func validateHandshakeLaunchContextAlignment(launchContext LaunchContext, host HostHello, isolate IsolateHello) error {
	if host.RunID != launchContext.RunID || host.StageID != launchContext.StageID || host.RoleInstanceID != launchContext.RoleInstanceID {
		return fmt.Errorf("host_hello run/stage/role identity does not match launch_context")
	}
	if host.SessionID != launchContext.SessionID || host.SessionNonce != launchContext.SessionNonce {
		return fmt.Errorf("host_hello session identity does not match launch_context")
	}
	if host.LaunchContextDigest != launchContext.LaunchContextDigest || isolate.LaunchContextDigest != launchContext.LaunchContextDigest {
		return fmt.Errorf("launch_context_digest mismatch across handshake messages")
	}
	return nil
}

func validateHandshakeTupleAlignment(host HostHello, isolate IsolateHello, ready SessionReady) error {
	if isolate.RunID != host.RunID || isolate.IsolateID != host.IsolateID || isolate.SessionID != host.SessionID || isolate.SessionNonce != host.SessionNonce {
		return fmt.Errorf("isolate_hello must match host session tuple")
	}
	if ready.RunID != host.RunID || ready.IsolateID != host.IsolateID || ready.SessionID != host.SessionID || ready.SessionNonce != host.SessionNonce {
		return fmt.Errorf("session_ready must match host session tuple")
	}
	if ready.IsolateKeyIDValue != isolate.IsolateSessionKey.KeyIDValue {
		return fmt.Errorf("session_ready isolate_key_id_value must match isolate_session_key")
	}
	return nil
}

func validateHandshakeTranscriptConsistency(isolate IsolateHello, ready SessionReady, transcriptHash string) error {
	if isolate.HandshakeTranscriptHash != transcriptHash {
		return fmt.Errorf("isolate_hello handshake_transcript_hash mismatch")
	}
	if ready.HandshakeTranscriptHash != transcriptHash {
		return fmt.Errorf("session_ready handshake_transcript_hash mismatch")
	}
	return nil
}

func validateHandshakeReplayAndIdentity(previous *SessionBindingRecord, host HostHello, isolate IsolateHello, transcriptHash string) error {
	if previous == nil {
		return nil
	}
	if previous.RunID != host.RunID || previous.IsolateID != host.IsolateID || previous.SessionID != host.SessionID {
		return nil
	}
	if previous.SessionNonce == host.SessionNonce || previous.HandshakeTranscriptHash == transcriptHash {
		return fmt.Errorf("replay detected for session tuple")
	}
	if previous.IsolateKeyIDValue != "" && previous.IsolateKeyIDValue != isolate.IsolateSessionKey.KeyIDValue {
		return fmt.Errorf("mid-session identity change detected for pinned isolate key")
	}
	return nil
}

func buildSessionBindingRecord(host HostHello, isolate IsolateHello, ready SessionReady, transcriptHash string) SessionBindingRecord {
	return SessionBindingRecord{
		RunID:                   host.RunID,
		IsolateID:               host.IsolateID,
		SessionID:               host.SessionID,
		SessionNonce:            host.SessionNonce,
		IsolateKeyIDValue:       isolate.IsolateSessionKey.KeyIDValue,
		HandshakeTranscriptHash: transcriptHash,
		ProvisioningMode:        ready.ProvisioningMode,
		IdentityBindingPosture:  ready.IdentityBindingPosture,
	}
}

func HandshakeTranscriptHash(host HostHello, isolate IsolateHello) (string, error) {
	transcript := struct {
		HostHello    HostHello `json:"host_hello"`
		IsolateHello struct {
			RunID               string            `json:"run_id"`
			IsolateID           string            `json:"isolate_id"`
			SessionID           string            `json:"session_id"`
			SessionNonce        string            `json:"session_nonce"`
			LaunchContextDigest string            `json:"launch_context_digest"`
			IsolateSessionKey   IsolateSessionKey `json:"isolate_session_key"`
			ProofOfPossession   SessionKeyProof   `json:"proof_of_possession"`
		} `json:"isolate_hello"`
	}{
		HostHello: host,
		IsolateHello: struct {
			RunID               string            `json:"run_id"`
			IsolateID           string            `json:"isolate_id"`
			SessionID           string            `json:"session_id"`
			SessionNonce        string            `json:"session_nonce"`
			LaunchContextDigest string            `json:"launch_context_digest"`
			IsolateSessionKey   IsolateSessionKey `json:"isolate_session_key"`
			ProofOfPossession   SessionKeyProof   `json:"proof_of_possession"`
		}{
			RunID:               isolate.RunID,
			IsolateID:           isolate.IsolateID,
			SessionID:           isolate.SessionID,
			SessionNonce:        isolate.SessionNonce,
			LaunchContextDigest: isolate.LaunchContextDigest,
			IsolateSessionKey:   isolate.IsolateSessionKey,
			ProofOfPossession:   isolate.ProofOfPossession,
		},
	}
	return canonicalSHA256Digest(transcript, "handshake transcript")
}

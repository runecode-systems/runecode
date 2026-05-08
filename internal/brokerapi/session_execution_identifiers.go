package brokerapi

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

const (
	maxSessionExecutionRunIDLength  = 128
	maxSessionExecutionPlanIDLength = 128 - len(runPlanAuthorityStepPrefix)
	maxSessionExecutionAttemptIDLen = 128
)

func sessionExecutionRunID(sessionID string, executionIndex int) string {
	return sessionExecutionScopedID("run", sessionID, executionIndex, maxSessionExecutionRunIDLength)
}

func sessionExecutionDerivedPlanID(sourceID string, executionIndex int) string {
	return sessionExecutionScopedID("plan", sourceID, executionIndex, maxSessionExecutionPlanIDLength)
}

func sessionExecutionDerivedAttemptID(prefix, sourceID string, executionIndex int) string {
	return sessionExecutionScopedID(prefix, sourceID, executionIndex, maxSessionExecutionAttemptIDLen)
}

func sessionExecutionScopedID(prefix, sourceID string, executionIndex, maxLength int) string {
	if executionIndex < 1 {
		executionIndex = 1
	}
	token := sessionExecutionIdentifierToken(sourceID)
	digest := sessionExecutionIdentifierDigestHex(sourceID)
	indexComponent := strconv.Itoa(executionIndex)
	maxTokenLength := maxLength - len(prefix) - 1 - 2 - len(digest) - len(indexComponent)
	if maxTokenLength < len("session") {
		maxTokenLength = len("session")
	}
	if len(token) > maxTokenLength {
		token = token[:maxTokenLength]
	}
	return fmt.Sprintf("%s_%s_%s_%s", prefix, token, digest, indexComponent)
}

func sessionExecutionIdentifierDigestHex(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

func sessionExecutionIdentifierToken(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "session"
	}
	b := strings.Builder{}
	b.Grow(len(trimmed))
	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			b.WriteByte(ch)
			continue
		}
		b.WriteByte('_')
	}
	normalized := strings.Trim(b.String(), "_-")
	if normalized == "" {
		return "session"
	}
	if normalized[0] < 'a' || normalized[0] > 'z' {
		return "s_" + normalized
	}
	return normalized
}

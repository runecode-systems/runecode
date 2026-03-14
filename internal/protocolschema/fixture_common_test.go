package protocolschema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const maxCanonicalSafeInteger int64 = 9007199254740991

type fixtureManifestFile struct {
	SchemaFixtures         []schemaFixtureEntry         `json:"schema_fixtures"`
	StreamSequenceFixtures []streamSequenceFixtureEntry `json:"stream_sequence_fixtures"`
	RuntimeFixtures        []runtimeFixtureEntry        `json:"runtime_invariant_fixtures"`
	CanonicalFixtures      []canonicalFixtureEntry      `json:"canonicalization_fixtures"`
}

type schemaFixtureEntry struct {
	ID          string `json:"id"`
	SchemaPath  string `json:"schema_path"`
	FixturePath string `json:"fixture_path"`
	ExpectValid bool   `json:"expect_valid"`
}

type streamSequenceFixtureEntry struct {
	ID              string `json:"id"`
	EventSchemaPath string `json:"event_schema_path"`
	FixturePath     string `json:"fixture_path"`
	ExpectValid     bool   `json:"expect_valid"`
}

type runtimeFixtureEntry struct {
	ID          string `json:"id"`
	SchemaPath  string `json:"schema_path"`
	FixturePath string `json:"fixture_path"`
	Rule        string `json:"rule"`
	ExpectValid bool   `json:"expect_valid"`
}

type canonicalFixtureEntry struct {
	ID                string `json:"id"`
	PayloadPath       string `json:"payload_path"`
	CanonicalJSONPath string `json:"canonical_json_path"`
	SHA256            string `json:"sha256"`
	ExpectValid       bool   `json:"expect_valid"`
}

func fixtureRoot() string {
	return "../../protocol/fixtures"
}

func fixturePath(t *testing.T, rel string) string {
	t.Helper()
	return rootedSchemaPath(t, fixtureRoot(), rel, "protocol/fixtures")
}

func loadFixtureManifest(t *testing.T) fixtureManifestFile {
	t.Helper()

	var manifest fixtureManifestFile
	loadJSON(t, fixturePath(t, "manifest.json"), &manifest)
	return manifest
}

func loadJSONArray(t *testing.T, filePath string) []any {
	t.Helper()

	value := loadJSONValue(t, filePath)
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("Decode(%q) produced %T, want []any", filePath, value)
	}
	return items
}

func loadJSONValue(t *testing.T, filePath string) any {
	t.Helper()

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Open(%q) returned error: %v", filePath, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("Decode(%q) returned error: %v", filePath, err)
	}
	return value
}

func canonicalizeJSONValue(value any) (string, error) {
	switch typed := value.(type) {
	case nil:
		return "null", nil
	case bool:
		if typed {
			return "true", nil
		}
		return "false", nil
	case string:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return "", err
		}
		return string(encoded), nil
	case json.Number:
		return canonicalizeNumber(typed)
	case []any:
		return canonicalizeArray(typed)
	case map[string]any:
		return canonicalizeObject(typed)
	default:
		return "", fmt.Errorf("unsupported JSON type %T", value)
	}
}

func canonicalizeNumber(value json.Number) (string, error) {
	parsed, err := canonicalIntegerFromText(value.String(), "number")
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(parsed, 10), nil
}

func canonicalizeArray(values []any) (string, error) {
	parts := make([]string, 0, len(values))
	for _, item := range values {
		canonical, err := canonicalizeJSONValue(item)
		if err != nil {
			return "", err
		}
		parts = append(parts, canonical)
	}
	return "[" + strings.Join(parts, ",") + "]", nil
}

func canonicalizeObject(object map[string]any) (string, error) {
	keys := make([]string, 0, len(object))
	for key := range object {
		if !isASCIIString(key) {
			return "", fmt.Errorf("object key %q is outside the MVP ASCII-only canonicalization profile", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		keyJSON, err := canonicalizeJSONValue(key)
		if err != nil {
			return "", err
		}
		valueJSON, err := canonicalizeJSONValue(object[key])
		if err != nil {
			return "", err
		}
		parts = append(parts, keyJSON+":"+valueJSON)
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

func canonicalSHA256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func validateLLMRuntimeInvariant(rule string, value map[string]any) error {
	switch rule {
	case "llm_request_unique_artifact_digests":
		return requireUniqueArtifactDigests(value, "input_artifacts")
	case "llm_request_unique_tool_identities":
		return requireUniqueToolIdentities(value, "tool_allowlist")
	case "llm_response_unique_output_artifact_digests":
		return requireUniqueArtifactDigests(value, "output_artifacts")
	case "llm_response_unique_tool_call_ids":
		return requireUniqueToolCallIDs(value, "proposed_tool_calls")
	default:
		return fmt.Errorf("unknown runtime invariant rule %q", rule)
	}
}

func requireUniqueArtifactDigests(value map[string]any, key string) error {
	items, err := requiredArrayValue(value, key)
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for index, item := range items {
		artifact, err := objectFromFixtureValue(item, fmt.Sprintf("%s[%d]", key, index))
		if err != nil {
			return err
		}
		identity, err := digestIdentityField(artifact, "digest")
		if err != nil {
			return err
		}
		if _, ok := seen[identity]; ok {
			return fmt.Errorf("duplicate artifact digest %s", identity)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func requireUniqueToolIdentities(value map[string]any, key string) error {
	items, err := requiredArrayValue(value, key)
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for index, item := range items {
		tool, err := objectFromFixtureValue(item, fmt.Sprintf("%s[%d]", key, index))
		if err != nil {
			return err
		}
		identity, err := toolIdentity(tool)
		if err != nil {
			return err
		}
		if _, ok := seen[identity]; ok {
			return fmt.Errorf("duplicate tool identity %s", identity)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func requireUniqueToolCallIDs(value map[string]any, key string) error {
	items, err := optionalArrayValue(value, key)
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for index, item := range items {
		toolCall, err := objectFromFixtureValue(item, fmt.Sprintf("%s[%d]", key, index))
		if err != nil {
			return err
		}
		id, err := stringField(toolCall, "tool_call_id")
		if err != nil {
			return err
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate tool_call_id %s", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validateStreamSequence(events []any) error {
	if len(events) == 0 {
		return fmt.Errorf("stream sequence must contain at least one event")
	}
	parsedEvents, err := parseStreamEvents(events)
	if err != nil {
		return err
	}
	return validateParsedStreamEvents(parsedEvents)
}

type streamEventView struct {
	streamID    string
	requestHash string
	eventType   string
	seq         int64
}

func parseStreamEvents(events []any) ([]streamEventView, error) {
	parsed := make([]streamEventView, 0, len(events))
	for index, item := range events {
		event, err := objectFromFixtureValue(item, fmt.Sprintf("events[%d]", index))
		if err != nil {
			return nil, err
		}
		parsedEvent, err := parseStreamEvent(event)
		if err != nil {
			return nil, fmt.Errorf("events[%d]: %w", index, err)
		}
		parsed = append(parsed, parsedEvent)
	}
	return parsed, nil
}

func parseStreamEvent(event map[string]any) (streamEventView, error) {
	streamID, err := stringField(event, "stream_id")
	if err != nil {
		return streamEventView{}, err
	}
	requestHash, err := digestIdentityField(event, "request_hash")
	if err != nil {
		return streamEventView{}, err
	}
	eventType, err := stringField(event, "event_type")
	if err != nil {
		return streamEventView{}, err
	}
	seq, err := integerField(event, "seq")
	if err != nil {
		return streamEventView{}, err
	}
	return streamEventView{streamID: streamID, requestHash: requestHash, eventType: eventType, seq: seq}, nil
}

func validateParsedStreamEvents(events []streamEventView) error {
	if err := requireStreamStartsAtSeqOne(events[0]); err != nil {
		return fmt.Errorf("first stream event: %w", err)
	}
	if err := requireFinalStreamEventTerminal(events[len(events)-1]); err != nil {
		return err
	}
	streamID := events[0].streamID
	requestHash := events[0].requestHash
	lastSeq := int64(0)

	for index, event := range events {
		if err := requireMatchingStreamIdentity(event, streamID, requestHash); err != nil {
			return err
		}
		if err := requireStrictlyMonotonicSeq(event.seq, lastSeq); err != nil {
			return err
		}
		if index < len(events)-1 && event.eventType == "response_terminal" {
			return fmt.Errorf("response_terminal must be the final event in the stream")
		}
		lastSeq = event.seq
	}
	return nil
}

func requireStreamStartsAtSeqOne(first streamEventView) error {
	if first.seq != 1 {
		return fmt.Errorf("first stream event must use seq=1")
	}
	return nil
}

func requireFinalStreamEventTerminal(last streamEventView) error {
	if last.eventType != "response_terminal" {
		return fmt.Errorf("stream must contain exactly one terminal event")
	}
	return nil
}

func requireMatchingStreamIdentity(event streamEventView, streamID string, requestHash string) error {
	if event.streamID != streamID {
		return fmt.Errorf("stream_id must remain constant across a stream")
	}
	if event.requestHash != requestHash {
		return fmt.Errorf("request_hash must remain constant across a stream")
	}
	return nil
}

func requireStrictlyMonotonicSeq(seq int64, lastSeq int64) error {
	if seq <= lastSeq {
		return fmt.Errorf("stream sequence numbers must be strictly monotonic")
	}
	return nil
}

func requiredArrayValue(object map[string]any, key string) ([]any, error) {
	value, ok := object[key]
	if !ok {
		return nil, fmt.Errorf("missing key %q", key)
	}
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("key %q has type %T, want []any", key, value)
	}
	return items, nil
}

func optionalArrayValue(object map[string]any, key string) ([]any, error) {
	value, ok := object[key]
	if !ok {
		return []any{}, nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("key %q has type %T, want []any", key, value)
	}
	return items, nil
}

func objectFromFixtureValue(value any, location string) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s has type %T, want map[string]any", location, value)
	}
	return object, nil
}

func requiredObjectField(object map[string]any, key string) (map[string]any, error) {
	value, ok := object[key]
	if !ok {
		return nil, fmt.Errorf("missing key %q", key)
	}
	child, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("key %q has type %T, want map[string]any", key, value)
	}
	return child, nil
}

func stringField(object map[string]any, key string) (string, error) {
	value, ok := object[key]
	if !ok {
		return "", fmt.Errorf("missing key %q", key)
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("key %q has type %T, want string", key, value)
	}
	return text, nil
}

func integerField(object map[string]any, key string) (int64, error) {
	value, ok := object[key]
	if !ok {
		return 0, fmt.Errorf("missing key %q", key)
	}
	return integerValue(value, key)
}

func integerValue(value any, location string) (int64, error) {
	switch typed := value.(type) {
	case json.Number:
		return canonicalIntegerFromText(typed.String(), location)
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return 0, fmt.Errorf("%s must be a finite integer", location)
		}
		if typed < -float64(maxCanonicalSafeInteger) || typed > float64(maxCanonicalSafeInteger) {
			return 0, fmt.Errorf("%s = %v is outside the shared Go/TS safe integer range", location, typed)
		}
		if typed != float64(int64(typed)) {
			return 0, fmt.Errorf("%s must be an integer", location)
		}
		text := strconv.FormatInt(int64(typed), 10)
		return canonicalIntegerFromText(text, location)
	default:
		return 0, fmt.Errorf("%s has type %T, want integer", location, value)
	}
}

func canonicalIntegerFromText(text string, location string) (int64, error) {
	parsed, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s = %q is not a supported integer: %w", location, text, err)
	}
	if parsed < -maxCanonicalSafeInteger || parsed > maxCanonicalSafeInteger {
		return 0, fmt.Errorf("%s = %q is outside the shared Go/TS safe integer range", location, text)
	}
	return parsed, nil
}

func digestIdentityField(object map[string]any, key string) (string, error) {
	digest, err := requiredObjectField(object, key)
	if err != nil {
		return "", err
	}
	return digestIdentity(digest)
}

func digestIdentity(digest map[string]any) (string, error) {
	hashAlg, err := stringField(digest, "hash_alg")
	if err != nil {
		return "", err
	}
	hash, err := stringField(digest, "hash")
	if err != nil {
		return "", err
	}
	return hashAlg + ":" + hash, nil
}

func toolIdentity(tool map[string]any) (string, error) {
	toolName, err := stringField(tool, "tool_name")
	if err != nil {
		return "", err
	}
	argumentsSchemaID, err := stringField(tool, "arguments_schema_id")
	if err != nil {
		return "", err
	}
	argumentsSchemaVersion, err := stringField(tool, "arguments_schema_version")
	if err != nil {
		return "", err
	}
	return toolName + "|" + argumentsSchemaID + "|" + argumentsSchemaVersion, nil
}

func isASCIIString(text string) bool {
	for _, r := range text {
		if r > 0x7f {
			return false
		}
	}
	return true
}

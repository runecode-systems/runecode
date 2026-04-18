package brokerapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

func destinationRefFromTypedRequest(typedRequest map[string]any) string {
	requestKind := strings.TrimSpace(stringField(typedRequest, "request_kind"))
	switch requestKind {
	case "git_ref_update":
		return destinationRefFromDescriptorField(typedRequest, "repository_identity")
	case "git_pull_request_create":
		return destinationRefFromDescriptorField(typedRequest, "base_repository_identity")
	default:
		return ""
	}
}

func destinationRefFromDescriptorField(value map[string]any, key string) string {
	raw, _ := value[key].(map[string]any)
	if raw == nil {
		return ""
	}
	if repoID := strings.TrimSpace(stringField(raw, "git_repository_identity")); repoID != "" {
		return repoID
	}
	host := strings.TrimSpace(stringField(raw, "canonical_host"))
	pathPrefix := strings.TrimSpace(stringField(raw, "canonical_path_prefix"))
	if host == "" {
		return ""
	}
	if pathPrefix == "" {
		return host
	}
	if !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}
	return host + pathPrefix
}

func nestedStringField(value map[string]any, keyA, keyB, keyC string) string {
	a, _ := value[keyA].(map[string]any)
	if a == nil {
		return ""
	}
	b, _ := a[keyB].(map[string]any)
	if b == nil {
		return ""
	}
	v, _ := b[keyC].(string)
	return strings.TrimSpace(v)
}

func digestField(value map[string]any, key string) (trustpolicy.Digest, error) {
	raw, ok := value[key]
	if !ok {
		return trustpolicy.Digest{}, fmt.Errorf("missing %s", key)
	}
	var digest trustpolicy.Digest
	if err := remarshalValue(raw, &digest); err != nil {
		return trustpolicy.Digest{}, err
	}
	if _, err := digest.Identity(); err != nil {
		return trustpolicy.Digest{}, err
	}
	return digest, nil
}

func digestSliceField(value map[string]any, key string) ([]trustpolicy.Digest, error) {
	raw, ok := value[key]
	if !ok {
		return nil, fmt.Errorf("missing %s", key)
	}
	var digests []trustpolicy.Digest
	if err := remarshalValue(raw, &digests); err != nil {
		return nil, err
	}
	for i := range digests {
		if _, err := digests[i].Identity(); err != nil {
			return nil, err
		}
	}
	return digests, nil
}

func mapFromValue(value any) (map[string]any, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func remarshalValue(in any, out any) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func cloneStringAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

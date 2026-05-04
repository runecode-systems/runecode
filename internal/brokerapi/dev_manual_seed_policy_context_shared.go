package brokerapi

import (
	"strings"
)

const devManualSeedInstanceID = "launcher-instance-1"

func devManualExternalAnchorTargetDescriptorDigest() (string, error) {
	descriptor := map[string]any{
		"descriptor_schema_id":   "runecode.protocol.audit.anchor_target.transparency_log.v0",
		"log_id":                 "manual-seed-transparency-log",
		"log_public_key_digest":  digestObjectForDevSeed("sha256:" + strings.Repeat("d", 64)),
		"entry_encoding_profile": "jcs_v1",
	}
	return externalAnchorCanonicalDescriptorDigestIdentity(descriptor)
}

func digestObjectForDevSeed(identity string) map[string]any {
	return map[string]any{"hash_alg": "sha256", "hash": strings.TrimPrefix(identity, "sha256:")}
}

package trustpolicy

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

func canonicalVerifierIdentity(keyIDValue string, decodedPublicKey []byte) (string, error) {
	keyID, err := keyIdentityFromPublicKey(decodedPublicKey)
	if err != nil {
		return "", err
	}
	if keyID != keyIDValue {
		return "", fmt.Errorf("key_id_value does not match canonical public-key digest")
	}
	return KeyIDProfile + ":" + keyIDValue, nil
}

func signatureVerifierIdentity(signature SignatureBlock) (string, error) {
	if signature.Alg != "ed25519" {
		return "", fmt.Errorf("unsupported signature algorithm %q", signature.Alg)
	}
	if signature.KeyID != KeyIDProfile {
		return "", fmt.Errorf("unsupported key_id profile %q", signature.KeyID)
	}
	if signature.KeyIDValue == "" {
		return "", fmt.Errorf("key_id_value is required")
	}
	if len(signature.KeyIDValue) != 64 {
		return "", fmt.Errorf("key_id_value must be 64 lowercase hex characters")
	}
	if strings.ToLower(signature.KeyIDValue) != signature.KeyIDValue {
		return "", fmt.Errorf("key_id_value must be lowercase hex")
	}
	if _, err := hex.DecodeString(signature.KeyIDValue); err != nil {
		return "", fmt.Errorf("invalid key_id_value: %w", err)
	}
	if signature.Signature == "" {
		return "", fmt.Errorf("signature is required")
	}
	if _, err := base64.StdEncoding.DecodeString(signature.Signature); err != nil {
		return "", fmt.Errorf("signature is not valid base64: %w", err)
	}
	return KeyIDProfile + ":" + signature.KeyIDValue, nil
}

func keyIdentityFromPublicKey(publicKey []byte) (string, error) {
	if len(publicKey) == 0 {
		return "", fmt.Errorf("public key is required")
	}
	digest := sha256.Sum256(publicKey)
	return hex.EncodeToString(digest[:]), nil
}

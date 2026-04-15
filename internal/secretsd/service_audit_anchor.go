package secretsd

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	auditAnchorPrivateKeyFileName = "audit-anchor-ed25519.private"
	envAuditAnchorPrivateKeyB64   = "RUNE_AUDIT_ANCHOR_PRIVATE_KEY_B64"
	envAuditAnchorPresenceMode    = "RUNE_AUDIT_ANCHOR_PRESENCE_MODE"
	envAuditAnchorKeyPosture      = "RUNE_AUDIT_ANCHOR_KEY_PROTECTION_POSTURE"
	envAuditAnchorAllowPassphrase = "RUNE_AUDIT_ANCHOR_ALLOW_PASSPHRASE"
)

type AuditAnchorSignRequest struct {
	PayloadCanonicalBytes []byte
	TargetSealDigest      trustpolicy.Digest
	LogicalScope          string
	ApprovalDecision      *trustpolicy.ApprovalDecision
}

type AuditAnchorSignResult struct {
	Signature            trustpolicy.SignatureBlock
	Preconditions        trustpolicy.SignRequestPreconditions
	AnchorWitnessDigest  trustpolicy.Digest
	AnchorWitnessKind    string
	SignerPublicKey      []byte
	SignerKeyIDValue     string
	SignatureRawMaterial []byte
}

func (s *Service) SignAuditAnchor(req AuditAnchorSignRequest) (AuditAnchorSignResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validateAuditAnchorSignRequest(req); err != nil {
		return AuditAnchorSignResult{}, err
	}
	privateKey, publicKey, keyIDValue, err := s.auditAnchorSigningMaterialLocked()
	if err != nil {
		return AuditAnchorSignResult{}, err
	}
	preconditions, err := buildAuditAnchorSignPreconditions(req)
	if err != nil {
		return AuditAnchorSignResult{}, err
	}
	witness := buildAuditAnchorWitness(req.TargetSealDigest, keyIDValue)
	rawSignature := ed25519.Sign(privateKey, req.PayloadCanonicalBytes)
	return buildAuditAnchorSignResult(preconditions, witness, publicKey, keyIDValue, rawSignature), nil
}

func validateAuditAnchorSignRequest(req AuditAnchorSignRequest) error {
	if len(req.PayloadCanonicalBytes) == 0 {
		return fmt.Errorf("payload canonical bytes are required")
	}
	if _, err := req.TargetSealDigest.Identity(); err != nil {
		return fmt.Errorf("target_seal_digest: %w", err)
	}
	return nil
}

func buildAuditAnchorSignPreconditions(req AuditAnchorSignRequest) (trustpolicy.SignRequestPreconditions, error) {
	logicalScope, err := resolveAuditAnchorScope(req.LogicalScope)
	if err != nil {
		return trustpolicy.SignRequestPreconditions{}, fmt.Errorf("audit anchor sign preconditions: %w", err)
	}
	preconditions := trustpolicy.SignRequestPreconditions{
		LogicalPurpose:          "audit_anchor",
		LogicalScope:            logicalScope,
		KeyProtectionPosture:    auditAnchorKeyProtectionPosture(),
		IdentityBindingPosture:  "attested",
		PresenceMode:            auditAnchorPresenceMode(),
		ApprovalDecisionContext: req.ApprovalDecision,
	}
	if err := trustpolicy.ValidateSignRequestPreconditions(preconditions); err != nil {
		return trustpolicy.SignRequestPreconditions{}, fmt.Errorf("audit anchor sign preconditions: %w", err)
	}
	if err := validateAuditAnchorSignerPosture(preconditions); err != nil {
		return trustpolicy.SignRequestPreconditions{}, fmt.Errorf("audit anchor sign preconditions: %w", err)
	}
	return preconditions, nil
}

func buildAuditAnchorWitness(targetSealDigest trustpolicy.Digest, keyIDValue string) trustpolicy.Digest {
	sealIdentity, _ := targetSealDigest.Identity()
	witnessHash := sha256.Sum256([]byte("audit_anchor|" + sealIdentity + "|" + keyIDValue))
	return trustpolicy.Digest{HashAlg: "sha256", Hash: hex.EncodeToString(witnessHash[:])}
}

func buildAuditAnchorSignResult(preconditions trustpolicy.SignRequestPreconditions, witness trustpolicy.Digest, publicKey ed25519.PublicKey, keyIDValue string, rawSignature []byte) AuditAnchorSignResult {
	return AuditAnchorSignResult{
		Signature: trustpolicy.SignatureBlock{
			Alg:        "ed25519",
			KeyID:      trustpolicy.KeyIDProfile,
			KeyIDValue: keyIDValue,
			Signature:  base64.StdEncoding.EncodeToString(rawSignature),
		},
		Preconditions:        preconditions,
		AnchorWitnessDigest:  witness,
		AnchorWitnessKind:    "local_user_presence_signature_v0",
		SignerPublicKey:      append([]byte(nil), publicKey...),
		SignerKeyIDValue:     keyIDValue,
		SignatureRawMaterial: append([]byte(nil), rawSignature...),
	}
}

func resolveAuditAnchorScope(scope string) (string, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "node", nil
	}
	if scope == "node" || scope == "deployment" {
		return scope, nil
	}
	return "", fmt.Errorf("unsupported logical_scope %q", scope)
}

func validateAuditAnchorSignerPosture(preconditions trustpolicy.SignRequestPreconditions) error {
	mode := strings.TrimSpace(preconditions.PresenceMode)
	posture := strings.TrimSpace(preconditions.KeyProtectionPosture)
	if mode == "os_confirmation" || mode == "hardware_touch" {
		return validateTouchPresencePosture(posture)
	}
	if mode == "passphrase" {
		return validatePassphrasePresencePosture(posture)
	}
	return fmt.Errorf("audit_anchor presence_mode must be os_confirmation, hardware_touch, or passphrase")
}

func validateTouchPresencePosture(posture string) error {
	if posture == "passphrase_wrapped" {
		return fmt.Errorf("inconsistent signer posture: passphrase_wrapped key_protection_posture requires presence_mode passphrase")
	}
	return nil
}

func validatePassphrasePresencePosture(posture string) error {
	if !auditAnchorPassphraseSupported() {
		return fmt.Errorf("presence_mode passphrase is not enabled for audit_anchor")
	}
	if posture != "passphrase_wrapped" {
		return fmt.Errorf("inconsistent signer posture: presence_mode passphrase requires key_protection_posture passphrase_wrapped")
	}
	return nil
}

func auditAnchorPassphraseSupported() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(envAuditAnchorAllowPassphrase)))
	return value == "1" || value == "true" || value == "yes"
}

func normalizeAuditAnchorScope(scope string) string {
	resolved, err := resolveAuditAnchorScope(scope)
	if err != nil {
		return "node"
	}
	scope = resolved
	if scope == "deployment" {
		return scope
	}
	return "node"
}

func auditAnchorPresenceMode() string {
	mode := strings.TrimSpace(os.Getenv(envAuditAnchorPresenceMode))
	if mode == "" {
		return "os_confirmation"
	}
	return mode
}

func auditAnchorKeyProtectionPosture() string {
	posture := strings.TrimSpace(os.Getenv(envAuditAnchorKeyPosture))
	if posture == "" {
		return "os_keystore"
	}
	return posture
}

func (s *Service) auditAnchorSigningMaterialLocked() (ed25519.PrivateKey, ed25519.PublicKey, string, error) {
	if inline := strings.TrimSpace(os.Getenv(envAuditAnchorPrivateKeyB64)); inline != "" {
		return decodeAuditAnchorPrivateKey(inline)
	}
	material, err := loadOrCreateAuditAnchorKeyMaterial(filepath.Join(s.root, auditAnchorPrivateKeyFileName), s.rand)
	if err != nil {
		return nil, nil, "", err
	}
	return decodeAuditAnchorPrivateKey(strings.TrimSpace(string(material)))
}

func loadOrCreateAuditAnchorKeyMaterial(path string, randSource io.Reader) ([]byte, error) {
	material, err := os.ReadFile(path)
	if err == nil {
		return material, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	_, privateKey, generateErr := ed25519.GenerateKey(randSource)
	if generateErr != nil {
		return nil, generateErr
	}
	encoded := base64.StdEncoding.EncodeToString(privateKey)
	if writeErr := writeFileAtomic(path, []byte(encoded), 0o600); writeErr != nil {
		return nil, writeErr
	}
	return []byte(encoded), nil
}

func decodeAuditAnchorPrivateKey(encoded string) (ed25519.PrivateKey, ed25519.PublicKey, string, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, nil, "", fmt.Errorf("decode audit anchor private key: %w", err)
	}
	defer func() {
		for i := range privateKeyBytes {
			privateKeyBytes[i] = 0
		}
	}()
	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		return nil, nil, "", fmt.Errorf("audit anchor private key must be %d bytes", ed25519.PrivateKeySize)
	}
	privateKey := ed25519.PrivateKey(append([]byte(nil), privateKeyBytes...))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	keyID := sha256.Sum256(publicKey)
	return privateKey, publicKey, hex.EncodeToString(keyID[:]), nil
}

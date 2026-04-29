package launcherdaemon

import (
	"crypto/ed25519"
	"encoding/hex"

	"github.com/runecode-ai/runecode/internal/trustpolicy"
)

const (
	runtimeImageVerifierSeedHex     = "1111111111111111111111111111111111111111111111111111111111111111"
	runtimeToolchainVerifierSeedHex = "2222222222222222222222222222222222222222222222222222222222222222"
)

func runtimeImageVerifierSignerForTests() (ed25519.PublicKey, ed25519.PrivateKey, string) {
	return deterministicRuntimeVerifierSigner(runtimeImageVerifierSeedHex)
}

func runtimeToolchainVerifierSignerForTests() (ed25519.PublicKey, ed25519.PrivateKey, string) {
	return deterministicRuntimeVerifierSigner(runtimeToolchainVerifierSeedHex)
}

func deterministicRuntimeVerifierSigner(seedHex string) (ed25519.PublicKey, ed25519.PrivateKey, string) {
	seed, err := hex.DecodeString(seedHex)
	if err != nil {
		panic(err)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return publicKey, privateKey, sha256HexString(publicKey)
}

func runtimeImageVerifierRecordForTests() trustpolicy.VerifierRecord {
	return builtInRuntimeImageVerifierRecord()
}

func runtimeToolchainVerifierRecordForTests() trustpolicy.VerifierRecord {
	return builtInRuntimeToolchainVerifierRecord()
}

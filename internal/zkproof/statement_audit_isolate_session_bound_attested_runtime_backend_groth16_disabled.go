package zkproof

func loadTrustedLocalSetupMaterialV0() (trustedLocalSetupMaterialV0, error) {
	setupOnceV0.Do(func() {
		setupMaterialErr = &FeasibilityError{Code: feasibilityCodeUnconfiguredProofBackend, Message: "trusted local Groth16 backend is disabled until reviewed setup assets are delivered; runtime deterministic setup generation is prohibited"}
	})
	return setupMaterialV0, setupMaterialErr
}

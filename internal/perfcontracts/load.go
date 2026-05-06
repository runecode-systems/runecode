package perfcontracts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func LoadManifest(root string) (Manifest, error) {
	path := filepath.Join(root, "manifest.json")
	var manifest Manifest
	if err := loadJSON(path, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func LoadFixtureInventory(root, relPath string) (FixtureInventory, error) {
	path := filepath.Join(root, filepath.FromSlash(relPath))
	var inventory FixtureInventory
	if err := loadJSON(path, &inventory); err != nil {
		return FixtureInventory{}, err
	}
	return inventory, nil
}

func LoadContract(root, relPath string) (ContractFile, error) {
	path := filepath.Join(root, filepath.FromSlash(relPath))
	var contract ContractFile
	if err := loadJSON(path, &contract); err != nil {
		return ContractFile{}, err
	}
	return contract, nil
}

func LoadBaseline(root, relPath string) (BaselineFile, error) {
	path := filepath.Join(root, filepath.FromSlash(relPath))
	var baseline BaselineFile
	if err := loadJSON(path, &baseline); err != nil {
		return BaselineFile{}, err
	}
	return baseline, nil
}

func LoadCheckOutput(path string) (CheckOutput, error) {
	var out CheckOutput
	if err := loadJSON(path, &out); err != nil {
		return CheckOutput{}, err
	}
	return out, nil
}

func loadJSON(path string, dst any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

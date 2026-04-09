package brokerapi

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/runecode-ai/runecode/internal/artifacts"
	"github.com/runecode-ai/runecode/internal/policyengine"
)

func (r policyRuntime) readRoleManifest(record artifacts.ArtifactRecord) (policyengine.ManifestInput, policyengine.RoleManifest, error) {
	input, err := r.readManifestInput(record)
	if err != nil {
		return policyengine.ManifestInput{}, policyengine.RoleManifest{}, err
	}
	manifest := policyengine.RoleManifest{}
	if err := json.Unmarshal(input.Payload, &manifest); err != nil {
		return policyengine.ManifestInput{}, policyengine.RoleManifest{}, fmt.Errorf("decode role manifest %q: %w", record.Reference.Digest, err)
	}
	return input, manifest, nil
}

func (r policyRuntime) readCapabilityManifest(record artifacts.ArtifactRecord) (policyengine.ManifestInput, policyengine.CapabilityManifest, error) {
	input, err := r.readManifestInput(record)
	if err != nil {
		return policyengine.ManifestInput{}, policyengine.CapabilityManifest{}, err
	}
	manifest := policyengine.CapabilityManifest{}
	if err := json.Unmarshal(input.Payload, &manifest); err != nil {
		return policyengine.ManifestInput{}, policyengine.CapabilityManifest{}, fmt.Errorf("decode capability manifest %q: %w", record.Reference.Digest, err)
	}
	return input, manifest, nil
}

func (r policyRuntime) readManifestInput(record artifacts.ArtifactRecord) (policyengine.ManifestInput, error) {
	payload, err := r.readArtifactBytes(record.Reference.Digest)
	if err != nil {
		return policyengine.ManifestInput{}, err
	}
	return policyengine.ManifestInput{Payload: payload, ExpectedHash: record.Reference.Digest}, nil
}

func (r policyRuntime) readArtifactBytes(digest string) ([]byte, error) {
	reader, err := r.service.Get(digest)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return body, nil
}

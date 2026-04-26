package brokerapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/runecode-ai/runecode/internal/artifacts"
)

type publicRegistryDeterministicFetcher struct{}

func (publicRegistryDeterministicFetcher) Fetch(_ context.Context, req DependencyFetchRequestObject, lease dependencyRegistryAuthLease) (io.ReadCloser, dependencyRegistryFetchMetadata, error) {
	if req.RegistryIdentity.DescriptorKind != "package_registry" {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("dependency registry descriptor_kind must be package_registry")
	}
	if lease == nil {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("dependency registry auth lease is required")
	}
	if lease.Posture() != dependencyRegistryAuthPosturePublicNoAuth {
		return nil, dependencyRegistryFetchMetadata{}, errors.New("public registry fetcher requires public_no_auth posture")
	}
	payload := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n", strings.TrimSpace(req.Ecosystem), strings.TrimSpace(req.PackageName), strings.TrimSpace(req.PackageVersion), strings.TrimSpace(req.RegistryIdentity.CanonicalHost), strings.TrimSpace(req.RegistryIdentity.CanonicalPathPrefix))
	payloadDigest := artifacts.DigestBytes([]byte(payload))
	upstreamDigest := artifacts.DigestBytes([]byte("manifest:" + payload))
	return io.NopCloser(strings.NewReader(payload)), dependencyRegistryFetchMetadata{ContentType: "application/octet-stream", ExpectedPayloadDigest: payloadDigest, UpstreamManifestDigest: upstreamDigest}, nil
}

type publicRegistryNoAuthSource struct{}

func (publicRegistryNoAuthSource) AcquireLease(_ context.Context, _ DependencyFetchRequestObject) (dependencyRegistryAuthLease, error) {
	return publicRegistryNoAuthLease{}, nil
}

type publicRegistryNoAuthLease struct{}

func (publicRegistryNoAuthLease) Posture() dependencyRegistryAuthPosture {
	return dependencyRegistryAuthPosturePublicNoAuth
}

func (publicRegistryNoAuthLease) LeaseID() string      { return "" }
func (publicRegistryNoAuthLease) ExpiresAt() time.Time { return time.Time{} }

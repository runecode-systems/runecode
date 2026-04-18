package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

func runtimeDestinationRefMatches(descriptor policyengine.DestinationDescriptor, destinationRef string) bool {
	if descriptor.DescriptorKind == "git_remote" {
		return runtimeGitRepositoryIdentityMatches(descriptor, destinationRef)
	}
	host, port, refPath := runtimeParseDestinationRef(destinationRef)
	if host == "" || !strings.EqualFold(host, descriptor.CanonicalHost) {
		return false
	}
	if !runtimeDestinationPortMatches(descriptor.CanonicalPort, port) {
		return false
	}
	if !runtimeDestinationPathPrefixMatches(descriptor.CanonicalPathPrefix, refPath) {
		return false
	}
	return true
}

func runtimeGitRepositoryIdentityMatches(descriptor policyengine.DestinationDescriptor, destinationRef string) bool {
	identityHost, identityPort, identityPath := runtimeParseGitRepositoryIdentity(descriptor.GitRepositoryIdentity)
	if identityHost == "" || identityPath == "" {
		return false
	}
	host, port, refPath := runtimeParseDestinationRef(destinationRef)
	if host == "" || !strings.EqualFold(host, identityHost) {
		return false
	}
	if !runtimeDestinationPortMatches(identityPort, port) {
		return false
	}
	return runtimeNormalizePath(refPath) == identityPath
}

func runtimeParseGitRepositoryIdentity(identity string) (string, *int, string) {
	host, port, refPath := runtimeParseDestinationRef(identity)
	if host == "" {
		return "", nil, ""
	}
	normalized := runtimeNormalizePath(refPath)
	if normalized == "/" {
		return "", nil, ""
	}
	return host, port, normalized
}

func containsStringValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

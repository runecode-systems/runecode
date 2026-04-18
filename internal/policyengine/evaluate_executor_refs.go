package policyengine

import (
	"path"
	"strconv"
	"strings"
)

func destinationRefMatches(descriptor DestinationDescriptor, destinationRef string) bool {
	if strings.TrimSpace(destinationRef) == "" {
		return false
	}
	if descriptor.DescriptorKind == "git_remote" {
		return gitRepositoryIdentityMatches(descriptor, destinationRef)
	}

	host, port, refPath := parseDestinationRef(destinationRef)
	if host == "" || strings.ToLower(host) != strings.ToLower(descriptor.CanonicalHost) {
		return false
	}
	if !destinationPortMatches(descriptor.CanonicalPort, port) {
		return false
	}
	if !destinationPathPrefixMatches(descriptor.CanonicalPathPrefix, refPath) {
		return false
	}
	return true
}

func destinationPortMatches(canonicalPort, observedPort *int) bool {
	if canonicalPort != nil {
		return observedPort != nil && *observedPort == *canonicalPort
	}
	const expectedPort = 443
	return observedPort == nil || *observedPort == expectedPort
}

func destinationPathPrefixMatches(prefix, observedPath string) bool {
	if prefix == "" {
		return true
	}
	normalizedPath := normalizeDestinationPath(observedPath)
	normalizedPrefix := normalizeDestinationPath(prefix)
	return strings.HasPrefix(normalizedPath, normalizedPrefix)
}

func normalizeDestinationPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	normalized := path.Clean(trimmed)
	if normalized == "." {
		return "/"
	}
	return normalized
}

func parseDestinationRef(ref string) (string, *int, string) {
	value := strings.TrimSpace(ref)
	if value == "" {
		return "", nil, ""
	}

	hostPort := value
	path := ""
	if slash := strings.Index(hostPort, "/"); slash >= 0 {
		path = hostPort[slash:]
		hostPort = hostPort[:slash]
	}

	host := hostPort
	var port *int
	if colon := strings.LastIndex(hostPort, ":"); colon > 0 && colon < len(hostPort)-1 {
		if parsed, err := strconv.Atoi(hostPort[colon+1:]); err == nil && parsed > 0 && parsed <= 65535 {
			h := hostPort[:colon]
			host = h
			port = &parsed
		}
	}

	if host == "" {
		return "", nil, ""
	}
	if path == "" {
		path = "/"
	}

	return host, port, path
}

func gitRepositoryIdentityMatches(descriptor DestinationDescriptor, destinationRef string) bool {
	identityHost, identityPath := parseGitRepositoryIdentity(descriptor.GitRepositoryIdentity)
	if identityHost == "" || identityPath == "" {
		return false
	}
	host, _, refPath := parseDestinationRef(destinationRef)
	if host == "" {
		return false
	}
	if !strings.EqualFold(host, identityHost) {
		return false
	}
	return normalizeDestinationPath(refPath) == identityPath
}

func parseGitRepositoryIdentity(value string) (string, string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", ""
	}
	host, _, path := parseDestinationRef(trimmed)
	if host == "" {
		return "", ""
	}
	if path == "/" {
		return "", ""
	}
	return strings.ToLower(host), normalizeDestinationPath(path)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

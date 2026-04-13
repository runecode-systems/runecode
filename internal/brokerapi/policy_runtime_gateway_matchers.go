package brokerapi

import (
	"strings"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

func runtimeDestinationRefMatches(descriptor policyengine.DestinationDescriptor, destinationRef string) bool {
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

func containsStringValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

package brokerapi

import "time"

const (
	defaultRequestIDFallback = "invalid_request"

	brokerArtifactListRequestSchemaPath  = "objects/BrokerArtifactListRequest.schema.json"
	brokerArtifactListResponseSchemaPath = "objects/BrokerArtifactListResponse.schema.json"
	brokerArtifactHeadRequestSchemaPath  = "objects/BrokerArtifactHeadRequest.schema.json"
	brokerArtifactHeadResponseSchemaPath = "objects/BrokerArtifactHeadResponse.schema.json"
	brokerArtifactPutRequestSchemaPath   = "objects/BrokerArtifactPutRequest.schema.json"
	brokerArtifactPutResponseSchemaPath  = "objects/BrokerArtifactPutResponse.schema.json"
	brokerErrorResponseSchemaPath        = "objects/BrokerErrorResponse.schema.json"
	errorEnvelopeSchemaVersion           = "0.3.0"
	errorResponseSchemaVersion           = "0.1.0"
)

type Limits struct {
	MaxMessageBytes        int
	MaxStructuralDepth     int
	MaxArrayLength         int
	MaxObjectProperties    int
	MaxRequestsPerClientPS int
	MaxInFlightPerClient   int
	MaxInFlightPerLane     int
	DefaultRequestDeadline time.Duration
	MaxStreamChunkBytes    int
	StreamIdleTimeout      time.Duration
	MaxResponseStreamBytes int
}

func DefaultLimits() Limits {
	return Limits{
		MaxMessageBytes:        1 << 20,
		MaxStructuralDepth:     64,
		MaxArrayLength:         10_000,
		MaxObjectProperties:    1_000,
		MaxRequestsPerClientPS: 256,
		MaxInFlightPerClient:   64,
		MaxInFlightPerLane:     32,
		DefaultRequestDeadline: 30 * time.Second,
		MaxStreamChunkBytes:    64 << 10,
		StreamIdleTimeout:      15 * time.Second,
		MaxResponseStreamBytes: 16 << 20,
	}
}

type APIConfig struct {
	Limits         Limits
	GatewayQuota   GatewayQuotaLimits
	RepositoryRoot string
}

func (c APIConfig) withDefaults() APIConfig {
	defaults := DefaultLimits()
	c.Limits.MaxMessageBytes = resolveIntLimit(c.Limits.MaxMessageBytes, defaults.MaxMessageBytes)
	c.Limits.MaxStructuralDepth = resolveIntLimit(c.Limits.MaxStructuralDepth, defaults.MaxStructuralDepth)
	c.Limits.MaxArrayLength = resolveIntLimit(c.Limits.MaxArrayLength, defaults.MaxArrayLength)
	c.Limits.MaxObjectProperties = resolveIntLimit(c.Limits.MaxObjectProperties, defaults.MaxObjectProperties)
	c.Limits.MaxRequestsPerClientPS = resolveIntLimit(c.Limits.MaxRequestsPerClientPS, defaults.MaxRequestsPerClientPS)
	c.Limits.MaxInFlightPerClient = resolveIntLimit(c.Limits.MaxInFlightPerClient, defaults.MaxInFlightPerClient)
	c.Limits.MaxInFlightPerLane = resolveIntLimit(c.Limits.MaxInFlightPerLane, defaults.MaxInFlightPerLane)
	c.Limits.DefaultRequestDeadline = resolveDurationLimit(c.Limits.DefaultRequestDeadline, defaults.DefaultRequestDeadline)
	c.Limits.MaxStreamChunkBytes = resolveIntLimit(c.Limits.MaxStreamChunkBytes, defaults.MaxStreamChunkBytes)
	c.Limits.StreamIdleTimeout = resolveDurationLimit(c.Limits.StreamIdleTimeout, defaults.StreamIdleTimeout)
	c.Limits.MaxResponseStreamBytes = resolveIntLimit(c.Limits.MaxResponseStreamBytes, defaults.MaxResponseStreamBytes)
	return c
}

func resolveIntLimit(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func resolveDurationLimit(value, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return fallback
}

type RequestContext struct {
	RequestID    string
	ClientID     string
	LaneID       string
	Deadline     *time.Time
	AdmissionErr error
}

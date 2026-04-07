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
	Limits Limits
}

func (c APIConfig) withDefaults() APIConfig {
	defaults := DefaultLimits()
	if c.Limits.MaxMessageBytes <= 0 {
		c.Limits.MaxMessageBytes = defaults.MaxMessageBytes
	}
	if c.Limits.MaxStructuralDepth <= 0 {
		c.Limits.MaxStructuralDepth = defaults.MaxStructuralDepth
	}
	if c.Limits.MaxArrayLength <= 0 {
		c.Limits.MaxArrayLength = defaults.MaxArrayLength
	}
	if c.Limits.MaxObjectProperties <= 0 {
		c.Limits.MaxObjectProperties = defaults.MaxObjectProperties
	}
	if c.Limits.MaxRequestsPerClientPS <= 0 {
		c.Limits.MaxRequestsPerClientPS = defaults.MaxRequestsPerClientPS
	}
	if c.Limits.MaxInFlightPerClient <= 0 {
		c.Limits.MaxInFlightPerClient = defaults.MaxInFlightPerClient
	}
	if c.Limits.MaxInFlightPerLane <= 0 {
		c.Limits.MaxInFlightPerLane = defaults.MaxInFlightPerLane
	}
	if c.Limits.DefaultRequestDeadline <= 0 {
		c.Limits.DefaultRequestDeadline = defaults.DefaultRequestDeadline
	}
	if c.Limits.MaxStreamChunkBytes <= 0 {
		c.Limits.MaxStreamChunkBytes = defaults.MaxStreamChunkBytes
	}
	if c.Limits.StreamIdleTimeout <= 0 {
		c.Limits.StreamIdleTimeout = defaults.StreamIdleTimeout
	}
	if c.Limits.MaxResponseStreamBytes <= 0 {
		c.Limits.MaxResponseStreamBytes = defaults.MaxResponseStreamBytes
	}
	return c
}

type RequestContext struct {
	RequestID    string
	ClientID     string
	LaneID       string
	Deadline     *time.Time
	AdmissionErr error
}

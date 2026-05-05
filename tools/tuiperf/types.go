//go:build linux

package main

import (
	"time"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

const checkSchemaVersion = "runecode.performance.check.v1"

type config struct {
	mode            string
	outputPath      string
	fixtureID       string
	runtimeDir      string
	socketName      string
	stateRoot       string
	auditLedgerRoot string
	targetAlias     string
	trials          int
	warmup          time.Duration
	window          time.Duration
	windows         int
	timeout         time.Duration
	benchOutput     string
}

type checkEnvelope struct {
	SchemaVersion string                            `json:"schema_version"`
	Metadata      map[string]any                    `json:"metadata,omitempty"`
	Measurements  []perfcontracts.MeasurementRecord `json:"measurements"`
}

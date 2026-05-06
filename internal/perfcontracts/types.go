package perfcontracts

type Manifest struct {
	SchemaVersion       string             `json:"schema_version"`
	ManifestVersion     string             `json:"manifest_version"`
	ChangeRef           string             `json:"change_ref"`
	FixtureInventoryRef string             `json:"fixture_inventory_ref"`
	Contracts           []ManifestContract `json:"contracts"`
	Baselines           []ManifestBaseline `json:"baselines,omitempty"`
	Taxonomy            MetricTaxonomy     `json:"metric_taxonomy"`
	LaneAuthorities     []string           `json:"lane_authorities"`
	ActivationStates    []string           `json:"activation_states"`
	Deferrals           []ManifestDeferral `json:"deferrals,omitempty"`
}

type ManifestContract struct {
	Surface string `json:"surface"`
	Path    string `json:"path"`
}

type ManifestBaseline struct {
	MetricID string `json:"metric_id"`
	Path     string `json:"path"`
}

type ManifestDeferral struct {
	ChangeRef string `json:"change_ref"`
	Reason    string `json:"reason"`
}

type MetricTaxonomy struct {
	BudgetClasses []string `json:"budget_classes"`
}

type FixtureInventory struct {
	SchemaVersion string          `json:"schema_version"`
	Fixtures      []FixtureRecord `json:"fixtures"`
}

type FixtureRecord struct {
	FixtureID     string `json:"fixture_id"`
	Surface       string `json:"surface"`
	RuntimeRegime string `json:"runtime_regime"`
	Status        string `json:"status"`
	Notes         string `json:"notes,omitempty"`
}

type ContractFile struct {
	SchemaVersion string           `json:"schema_version"`
	ContractID    string           `json:"contract_id"`
	Surface       string           `json:"surface"`
	Metrics       []MetricContract `json:"metrics"`
}

type MetricContract struct {
	MetricID         string          `json:"metric_id"`
	Subsystem        string          `json:"subsystem"`
	RuntimeRegime    string          `json:"runtime_regime"`
	FixtureID        string          `json:"fixture_id"`
	MeasurementKind  string          `json:"measurement_kind"`
	Unit             string          `json:"unit"`
	AuthoritativeEnv string          `json:"authoritative_environment"`
	SamplingPolicy   SamplingPolicy  `json:"sampling_policy"`
	BudgetClass      string          `json:"budget_class"`
	Threshold        MetricThreshold `json:"threshold"`
	LaneAuthority    string          `json:"lane_authority"`
	ActivationState  string          `json:"activation_state"`
	BaselineSource   string          `json:"baseline_source,omitempty"`
	BaselineRef      string          `json:"baseline_ref,omitempty"`
	ComparisonMethod string          `json:"comparison_method"`
	NoiseFloor       float64         `json:"practical_noise_floor,omitempty"`
	ThresholdOrigin  string          `json:"threshold_origin"`
	TimingBoundary   TimingBoundary  `json:"timing_boundary"`
	Notes            string          `json:"notes,omitempty"`
}

type SamplingPolicy struct {
	Trials                 int  `json:"trials,omitempty"`
	RepeatedSamples        int  `json:"repeated_samples,omitempty"`
	WarmupMillis           int  `json:"warmup_millis,omitempty"`
	ObservationWindowMs    int  `json:"observation_window_millis,omitempty"`
	ObservationWindows     int  `json:"observation_windows,omitempty"`
	P95Authoritative       bool `json:"p95_authoritative,omitempty"`
	MedianMaxAuthoritative bool `json:"median_max_authoritative,omitempty"`
}

type MetricThreshold struct {
	ExactValue           *float64 `json:"exact_value,omitempty"`
	MaxValue             *float64 `json:"max_value,omitempty"`
	MaxRegressionPercent *float64 `json:"max_regression_percent,omitempty"`
}

type TimingBoundary struct {
	StartEvent     string   `json:"start_event"`
	EndEvent       string   `json:"end_event"`
	ClockSource    string   `json:"clock_source"`
	EvidenceSource string   `json:"evidence_source"`
	IncludedPhases []string `json:"included_phases"`
}

type CheckOutput struct {
	SchemaVersion string              `json:"schema_version"`
	Measurements  []MeasurementRecord `json:"measurements"`
}

type MeasurementRecord struct {
	MetricID string  `json:"metric_id"`
	Value    float64 `json:"value"`
	Unit     string  `json:"unit"`
}

type BaselineFile struct {
	SchemaVersion string    `json:"schema_version"`
	MetricID      string    `json:"metric_id"`
	Unit          string    `json:"unit"`
	BaselineValue *float64  `json:"baseline_value,omitempty"`
	Samples       []float64 `json:"samples,omitempty"`
	Summary       struct {
		Median *float64 `json:"median,omitempty"`
	} `json:"summary,omitempty"`
}

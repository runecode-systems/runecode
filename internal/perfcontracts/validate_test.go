package perfcontracts

import "testing"

func TestValidateAcceptsReviewedMetricContract(t *testing.T) {
	manifest := Manifest{SchemaVersion: "runecode.performance.manifest.v1"}
	inventory := FixtureInventory{SchemaVersion: "runecode.performance.fixtures.v1", Fixtures: []FixtureRecord{{FixtureID: "tui.empty.v1"}}}
	contracts := []ContractFile{{
		SchemaVersion: "runecode.performance.contract.v1",
		ContractID:    "performance.tui.v1",
		Metrics: []MetricContract{{
			MetricID:        "metric.tui.attach.latency.p95",
			FixtureID:       "tui.empty.v1",
			BudgetClass:     "absolute-budget",
			LaneAuthority:   "required_shared_linux",
			ActivationState: "required",
			ThresholdOrigin: "product_budget",
			TimingBoundary: TimingBoundary{
				StartEvent:     "spawn",
				EndEvent:       "ready",
				ClockSource:    "monotonic",
				EvidenceSource: "events",
				IncludedPhases: []string{"launch"},
			},
		}},
	}}
	if err := Validate(manifest, inventory, contracts); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateRejectsMissingFixtureReference(t *testing.T) {
	manifest := Manifest{SchemaVersion: "runecode.performance.manifest.v1"}
	inventory := FixtureInventory{SchemaVersion: "runecode.performance.fixtures.v1", Fixtures: []FixtureRecord{{FixtureID: "tui.empty.v1"}}}
	contracts := []ContractFile{{
		SchemaVersion: "runecode.performance.contract.v1",
		ContractID:    "performance.tui.v1",
		Metrics: []MetricContract{{
			MetricID:        "metric.tui.attach.latency.p95",
			FixtureID:       "missing.v1",
			BudgetClass:     "absolute-budget",
			LaneAuthority:   "required_shared_linux",
			ActivationState: "required",
			ThresholdOrigin: "product_budget",
			TimingBoundary:  TimingBoundary{StartEvent: "spawn", EndEvent: "ready", ClockSource: "monotonic", EvidenceSource: "events", IncludedPhases: []string{"launch"}},
		}},
	}}
	if err := Validate(manifest, inventory, contracts); err == nil {
		t.Fatal("Validate error = nil, want missing fixture failure")
	}
}

func TestValidateRejectsUnsupportedThresholdOrigin(t *testing.T) {
	manifest := Manifest{SchemaVersion: "runecode.performance.manifest.v1"}
	inventory := FixtureInventory{SchemaVersion: "runecode.performance.fixtures.v1", Fixtures: []FixtureRecord{{FixtureID: "tui.empty.v1"}}}
	contracts := []ContractFile{{
		SchemaVersion: "runecode.performance.contract.v1",
		ContractID:    "performance.tui.v1",
		Metrics: []MetricContract{{
			MetricID:        "metric.tui.attach.latency.p95",
			FixtureID:       "tui.empty.v1",
			BudgetClass:     "absolute-budget",
			LaneAuthority:   "required_shared_linux",
			ActivationState: "required",
			ThresholdOrigin: "unsupported_origin",
			TimingBoundary:  TimingBoundary{StartEvent: "spawn", EndEvent: "ready", ClockSource: "monotonic", EvidenceSource: "events", IncludedPhases: []string{"launch"}},
		}},
	}}
	if err := Validate(manifest, inventory, contracts); err == nil {
		t.Fatal("Validate error = nil, want threshold_origin failure")
	}
}

func TestValidateWithBaselinesRejectsRegressionMetricWithoutBaseline(t *testing.T) {
	manifest := Manifest{SchemaVersion: "runecode.performance.manifest.v1"}
	inventory := FixtureInventory{SchemaVersion: "runecode.performance.fixtures.v1", Fixtures: []FixtureRecord{{FixtureID: "tui.empty.v1"}}}
	threshold := 15.0
	contracts := []ContractFile{{
		SchemaVersion: "runecode.performance.contract.v1",
		ContractID:    "performance.tui.v1",
		Metrics: []MetricContract{{
			MetricID:        "metric.tui.render.shell_view_empty.ns_op",
			FixtureID:       "tui.empty.v1",
			Unit:            "ns/op",
			BudgetClass:     "regression-budget",
			BaselineRef:     "baselines/metric.tui.render.shell_view_empty.ns_op.v1.json",
			LaneAuthority:   "required_shared_linux",
			ActivationState: "required",
			ThresholdOrigin: "first_calibration",
			Threshold:       MetricThreshold{MaxRegressionPercent: &threshold},
			TimingBoundary:  TimingBoundary{StartEvent: "start", EndEvent: "end", ClockSource: "monotonic", EvidenceSource: "bench", IncludedPhases: []string{"render"}},
		}},
	}}
	if err := ValidateWithBaselines(manifest, inventory, contracts, map[string]BaselineFile{}); err == nil {
		t.Fatal("ValidateWithBaselines error = nil, want missing baseline failure")
	}
}

func TestValidateWithBaselinesAllowsNonRequiredRegressionMetricWithoutBaseline(t *testing.T) {
	manifest := Manifest{SchemaVersion: "runecode.performance.manifest.v1"}
	inventory := FixtureInventory{SchemaVersion: "runecode.performance.fixtures.v1", Fixtures: []FixtureRecord{{FixtureID: "tui.empty.v1"}}}
	threshold := 15.0
	contracts := []ContractFile{{
		SchemaVersion: "runecode.performance.contract.v1",
		ContractID:    "performance.tui.v1",
		Metrics: []MetricContract{{
			MetricID:        "metric.tui.render.shell_view_empty.ns_op",
			FixtureID:       "tui.empty.v1",
			Unit:            "ns/op",
			BudgetClass:     "regression-budget",
			LaneAuthority:   "required_shared_linux",
			ActivationState: "defined",
			ThresholdOrigin: "first_calibration",
			Threshold:       MetricThreshold{MaxRegressionPercent: &threshold},
			TimingBoundary:  TimingBoundary{StartEvent: "start", EndEvent: "end", ClockSource: "monotonic", EvidenceSource: "bench", IncludedPhases: []string{"render"}},
		}},
	}}
	if err := ValidateWithBaselines(manifest, inventory, contracts, map[string]BaselineFile{}); err != nil {
		t.Fatalf("ValidateWithBaselines returned error for non-required regression metric: %v", err)
	}
}

func TestValidateWithBaselinesRejectsMismatchedBaselineUnit(t *testing.T) {
	manifest := Manifest{SchemaVersion: "runecode.performance.manifest.v1"}
	inventory := FixtureInventory{SchemaVersion: "runecode.performance.fixtures.v1", Fixtures: []FixtureRecord{{FixtureID: "tui.empty.v1"}}}
	threshold := 15.0
	median := 1000.0
	contracts := []ContractFile{{
		SchemaVersion: "runecode.performance.contract.v1",
		ContractID:    "performance.tui.v1",
		Metrics: []MetricContract{{
			MetricID:        "metric.tui.render.shell_view_empty.ns_op",
			FixtureID:       "tui.empty.v1",
			Unit:            "ns/op",
			BudgetClass:     "regression-budget",
			BaselineRef:     "baselines/metric.tui.render.shell_view_empty.ns_op.v1.json",
			LaneAuthority:   "required_shared_linux",
			ActivationState: "required",
			ThresholdOrigin: "first_calibration",
			Threshold:       MetricThreshold{MaxRegressionPercent: &threshold},
			TimingBoundary:  TimingBoundary{StartEvent: "start", EndEvent: "end", ClockSource: "monotonic", EvidenceSource: "bench", IncludedPhases: []string{"render"}},
		}},
	}}
	baselines := map[string]BaselineFile{
		"metric.tui.render.shell_view_empty.ns_op": {
			SchemaVersion: "runecode.performance.baseline.v1",
			MetricID:      "metric.tui.render.shell_view_empty.ns_op",
			Unit:          "ms",
			Summary: struct {
				Median *float64 `json:"median,omitempty"`
			}{Median: &median},
		},
	}
	if err := ValidateWithBaselines(manifest, inventory, contracts, baselines); err == nil {
		t.Fatal("ValidateWithBaselines error = nil, want baseline unit mismatch failure")
	}
}

package perfcontracts

import (
	"fmt"
	"strings"
)

var allowedBudgetClasses = map[string]struct{}{
	"exact":             {},
	"absolute-budget":   {},
	"regression-budget": {},
	"hybrid-budget":     {},
}

var allowedLaneAuthorities = map[string]struct{}{
	"required_shared_linux":       {},
	"required_tight_linux":        {},
	"informational_until_stable":  {},
	"contract_pending_dependency": {},
	"extended":                    {},
}

var allowedActivationStates = map[string]struct{}{
	"defined":                     {},
	"informational":               {},
	"required":                    {},
	"contract_pending_dependency": {},
}

var allowedThresholdOrigins = map[string]struct{}{
	"product_budget":         {},
	"investigation_baseline": {},
	"first_calibration":      {},
	"temporary_guardrail":    {},
}

func Validate(manifest Manifest, inventory FixtureInventory, contracts []ContractFile) error {
	return ValidateWithBaselines(manifest, inventory, contracts, nil)
}

func ValidateWithBaselines(manifest Manifest, inventory FixtureInventory, contracts []ContractFile, baselinesByMetric map[string]BaselineFile) error {
	if err := validateManifestAndInventory(manifest, inventory); err != nil {
		return err
	}
	baselineRefsByMetric, err := baselineRefSet(manifest.Baselines)
	if err != nil {
		return err
	}
	fixtures, err := fixtureSet(inventory)
	if err != nil {
		return err
	}
	return validateContracts(contracts, fixtures, baselinesByMetric, baselineRefsByMetric)
}

func validateManifestAndInventory(manifest Manifest, inventory FixtureInventory) error {
	if strings.TrimSpace(manifest.SchemaVersion) == "" {
		return fmt.Errorf("manifest schema_version is required")
	}
	if strings.TrimSpace(inventory.SchemaVersion) == "" {
		return fmt.Errorf("fixture inventory schema_version is required")
	}
	return nil
}

func fixtureSet(inventory FixtureInventory) (map[string]struct{}, error) {
	fixtures := map[string]struct{}{}
	for _, fixture := range inventory.Fixtures {
		if strings.TrimSpace(fixture.FixtureID) == "" {
			return nil, fmt.Errorf("fixture_id is required")
		}
		fixtures[fixture.FixtureID] = struct{}{}
	}
	return fixtures, nil
}

func baselineRefSet(entries []ManifestBaseline) (map[string]string, error) {
	refs := map[string]string{}
	for _, entry := range entries {
		if existing, ok := refs[entry.MetricID]; ok {
			return nil, fmt.Errorf("manifest baseline metric_id %q duplicated with paths %q and %q", entry.MetricID, existing, entry.Path)
		}
		refs[entry.MetricID] = entry.Path
	}
	return refs, nil
}

func validateContracts(contracts []ContractFile, fixtures map[string]struct{}, baselinesByMetric map[string]BaselineFile, baselineRefsByMetric map[string]string) error {
	for _, contract := range contracts {
		if err := validateContract(contract, fixtures, baselinesByMetric, baselineRefsByMetric); err != nil {
			return err
		}
	}
	return nil
}

func validateContract(contract ContractFile, fixtures map[string]struct{}, baselinesByMetric map[string]BaselineFile, baselineRefsByMetric map[string]string) error {
	if strings.TrimSpace(contract.SchemaVersion) == "" {
		return fmt.Errorf("contract %s missing schema_version", contract.ContractID)
	}
	for _, metric := range contract.Metrics {
		if err := validateMetric(metric, fixtures, baselinesByMetric, baselineRefsByMetric); err != nil {
			return fmt.Errorf("contract %s metric %s invalid: %w", contract.ContractID, metric.MetricID, err)
		}
	}
	return nil
}

func validateMetric(metric MetricContract, fixtures map[string]struct{}, baselinesByMetric map[string]BaselineFile, baselineRefsByMetric map[string]string) error {
	checks := []func(MetricContract, map[string]struct{}, map[string]BaselineFile, map[string]string) error{
		validateMetricIdentity,
		validateMetricEnums,
		validateMetricFixture,
		validateMetricThresholdOrigin,
		validateMetricTimingBoundary,
		validateMetricBaseline,
	}
	for _, check := range checks {
		if err := check(metric, fixtures, baselinesByMetric, baselineRefsByMetric); err != nil {
			return err
		}
	}
	return nil
}

func validateMetricIdentity(metric MetricContract, _ map[string]struct{}, _ map[string]BaselineFile, _ map[string]string) error {
	if strings.TrimSpace(metric.MetricID) == "" {
		return fmt.Errorf("metric_id is required")
	}
	return nil
}

func validateMetricEnums(metric MetricContract, _ map[string]struct{}, _ map[string]BaselineFile, _ map[string]string) error {
	if _, ok := allowedBudgetClasses[metric.BudgetClass]; !ok {
		return fmt.Errorf("budget_class %q unsupported", metric.BudgetClass)
	}
	if _, ok := allowedLaneAuthorities[metric.LaneAuthority]; !ok {
		return fmt.Errorf("lane_authority %q unsupported", metric.LaneAuthority)
	}
	if _, ok := allowedActivationStates[metric.ActivationState]; !ok {
		return fmt.Errorf("activation_state %q unsupported", metric.ActivationState)
	}
	return nil
}

func validateMetricFixture(metric MetricContract, fixtures map[string]struct{}, _ map[string]BaselineFile, _ map[string]string) error {
	if _, ok := fixtures[metric.FixtureID]; !ok {
		return fmt.Errorf("fixture_id %q missing from inventory", metric.FixtureID)
	}
	return nil
}

func validateMetricThresholdOrigin(metric MetricContract, _ map[string]struct{}, _ map[string]BaselineFile, _ map[string]string) error {
	if strings.TrimSpace(metric.ThresholdOrigin) == "" {
		return fmt.Errorf("threshold_origin is required")
	}
	if _, ok := allowedThresholdOrigins[metric.ThresholdOrigin]; !ok {
		return fmt.Errorf("threshold_origin %q unsupported", metric.ThresholdOrigin)
	}
	return nil
}

func validateMetricTimingBoundary(metric MetricContract, _ map[string]struct{}, _ map[string]BaselineFile, _ map[string]string) error {
	boundary := metric.TimingBoundary
	if strings.TrimSpace(boundary.StartEvent) == "" || strings.TrimSpace(boundary.EndEvent) == "" {
		return fmt.Errorf("timing_boundary start_event/end_event are required")
	}
	if strings.TrimSpace(boundary.ClockSource) == "" || strings.TrimSpace(boundary.EvidenceSource) == "" {
		return fmt.Errorf("timing_boundary clock_source/evidence_source are required")
	}
	if len(boundary.IncludedPhases) == 0 {
		return fmt.Errorf("timing_boundary included_phases is required")
	}
	return nil
}

func validateMetricBaseline(metric MetricContract, _ map[string]struct{}, baselinesByMetric map[string]BaselineFile, baselineRefsByMetric map[string]string) error {
	if !requiresBaselineValidation(metric) {
		return nil
	}
	if strings.TrimSpace(metric.BaselineRef) == "" {
		return fmt.Errorf("baseline_ref is required for %s", metric.BudgetClass)
	}
	if err := validateBaselineRefProvenance(metric, baselineRefsByMetric); err != nil {
		return err
	}
	if baselinesByMetric == nil {
		return nil
	}
	baseline, ok := baselinesByMetric[metric.MetricID]
	if !ok {
		return fmt.Errorf("baseline for metric_id %q missing from manifest baselines", metric.MetricID)
	}
	if strings.TrimSpace(baseline.MetricID) != metric.MetricID {
		return fmt.Errorf("baseline metric_id %q does not match contract metric_id %q", baseline.MetricID, metric.MetricID)
	}
	if strings.TrimSpace(baseline.Unit) != metric.Unit {
		return fmt.Errorf("baseline unit %q does not match contract unit %q", baseline.Unit, metric.Unit)
	}
	if _, ok := baselineValue(baseline); !ok {
		return fmt.Errorf("baseline for metric_id %q has no usable baseline value", metric.MetricID)
	}
	return nil
}

func requiresBaselineValidation(metric MetricContract) bool {
	if metric.BudgetClass != "regression-budget" && metric.BudgetClass != "hybrid-budget" {
		return false
	}
	return metric.ActivationState == "required"
}

func validateBaselineRefProvenance(metric MetricContract, baselineRefsByMetric map[string]string) error {
	authoritativeRef, ok := baselineRefsByMetric[metric.MetricID]
	if !ok {
		return fmt.Errorf("baseline_ref for metric_id %q missing from manifest baselines", metric.MetricID)
	}
	if strings.TrimSpace(authoritativeRef) != metric.BaselineRef {
		return fmt.Errorf("baseline_ref %q does not match manifest baseline path %q for metric_id %q", metric.BaselineRef, authoritativeRef, metric.MetricID)
	}
	return nil
}

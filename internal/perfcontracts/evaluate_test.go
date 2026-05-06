package perfcontracts

import (
	"strings"
	"testing"
)

func TestEvaluateHonorsComparisonMethodContracts(t *testing.T) {
	max500 := 500.0
	max200 := 200.0
	reg15 := 15.0
	exact2 := 2.0

	for _, tc := range comparisonMethodContractTests(max500, max200, reg15, exact2) {
		t.Run(tc.name, func(t *testing.T) {
			assertEvaluationResult(t, tc)
		})
	}
}

type comparisonMethodTestCase struct {
	name           string
	metric         MetricContract
	measurement    MeasurementRecord
	baseline       BaselineFile
	wantViolation  bool
	wantReasonLike string
}

func comparisonMethodContractTests(max500, max200, reg15, exact2 float64) []comparisonMethodTestCase {
	tests := append([]comparisonMethodTestCase{}, absoluteComparisonMethodTests(max500, max200, exact2)...)
	tests = append(tests, regressionComparisonMethodTests(max500, reg15)...)
	return tests
}

func absoluteComparisonMethodTests(max500, max200, exact2 float64) []comparisonMethodTestCase {
	tests := append([]comparisonMethodTestCase{}, absoluteCeilingComparisonTests(max500, max200)...)
	return append(tests, absoluteComparisonOverrideTests(max200, exact2)...)
}

func absoluteCeilingComparisonTests(max500, max200 float64) []comparisonMethodTestCase {
	tests := append([]comparisonMethodTestCase{}, exactAndAbsoluteComparisonTests(max500)...)
	return append(tests, percentileAndWindowComparisonTests(max200)...)
}

func exactAndAbsoluteComparisonTests(max500 float64) []comparisonMethodTestCase {
	return []comparisonMethodTestCase{
		{
			name:          "exact_match passes on exact value",
			metric:        MetricContract{MetricID: "m.exact.pass", Unit: "count", BudgetClass: "exact", ComparisonMethod: "exact_match", Threshold: MetricThreshold{ExactValue: floatPtr(2)}},
			measurement:   MeasurementRecord{MetricID: "m.exact.pass", Unit: "count", Value: 2},
			wantViolation: false,
		},
		{
			name:           "absolute_ceiling fails above max",
			metric:         MetricContract{MetricID: "m.abs.fail", Unit: "ms", BudgetClass: "absolute-budget", ComparisonMethod: "absolute_ceiling", Threshold: MetricThreshold{MaxValue: &max500}},
			measurement:    MeasurementRecord{MetricID: "m.abs.fail", Unit: "ms", Value: 501},
			wantViolation:  true,
			wantReasonLike: "exceeds max",
		},
		{
			name:          "max_ceiling passes under max",
			metric:        MetricContract{MetricID: "m.max.pass", Unit: "ms", BudgetClass: "absolute-budget", ComparisonMethod: "max_ceiling", Threshold: MetricThreshold{MaxValue: &max500}},
			measurement:   MeasurementRecord{MetricID: "m.max.pass", Unit: "ms", Value: 499},
			wantViolation: false,
		},
	}
}

func percentileAndWindowComparisonTests(max200 float64) []comparisonMethodTestCase {
	return []comparisonMethodTestCase{
		{
			name:           "p95_ceiling fails above max",
			metric:         MetricContract{MetricID: "m.p95.fail", Unit: "ms", BudgetClass: "absolute-budget", ComparisonMethod: "p95_ceiling", Threshold: MetricThreshold{MaxValue: &max200}},
			measurement:    MeasurementRecord{MetricID: "m.p95.fail", Unit: "ms", Value: 220},
			wantViolation:  true,
			wantReasonLike: "exceeds max",
		},
		{
			name:           "window_average fails above max",
			metric:         MetricContract{MetricID: "m.win.avg.fail", Unit: "percent", BudgetClass: "absolute-budget", ComparisonMethod: "window_average", Threshold: MetricThreshold{MaxValue: &max200}},
			measurement:    MeasurementRecord{MetricID: "m.win.avg.fail", Unit: "percent", Value: 220},
			wantViolation:  true,
			wantReasonLike: "exceeds max",
		},
		{
			name:           "window_max fails above max",
			metric:         MetricContract{MetricID: "m.win.max.fail", Unit: "percent", BudgetClass: "absolute-budget", ComparisonMethod: "window_max", Threshold: MetricThreshold{MaxValue: &max200}},
			measurement:    MeasurementRecord{MetricID: "m.win.max.fail", Unit: "percent", Value: 220},
			wantViolation:  true,
			wantReasonLike: "exceeds max",
		},
	}
}

func absoluteComparisonOverrideTests(max200, exact2 float64) []comparisonMethodTestCase {
	return []comparisonMethodTestCase{
		{
			name:           "comparison method takes precedence over budget class",
			metric:         MetricContract{MetricID: "m.method.overrides.budget", Unit: "ms", BudgetClass: "exact", ComparisonMethod: "absolute_ceiling", Threshold: MetricThreshold{ExactValue: &exact2, MaxValue: &max200}},
			measurement:    MeasurementRecord{MetricID: "m.method.overrides.budget", Unit: "ms", Value: 220},
			wantViolation:  true,
			wantReasonLike: "exceeds max",
		},
	}
}

func regressionComparisonMethodTests(max500, reg15 float64) []comparisonMethodTestCase {
	return []comparisonMethodTestCase{
		{
			name:          "median_regression_with_noise_floor ignores low-noise deltas",
			metric:        MetricContract{MetricID: "m.reg.noise.pass", Unit: "ns/op", BudgetClass: "regression-budget", ComparisonMethod: "median_regression_with_noise_floor", Threshold: MetricThreshold{MaxRegressionPercent: &reg15}, NoiseFloor: 10},
			measurement:   MeasurementRecord{MetricID: "m.reg.noise.pass", Unit: "ns/op", Value: 105},
			baseline:      medianBaseline(100),
			wantViolation: false,
		},
		{
			name:           "median_plus_regression checks absolute max",
			metric:         MetricContract{MetricID: "m.med.plus.abs.fail", Unit: "ms", BudgetClass: "hybrid-budget", ComparisonMethod: "median_plus_regression", Threshold: MetricThreshold{MaxValue: &max500, MaxRegressionPercent: &reg15}, NoiseFloor: 10},
			measurement:    MeasurementRecord{MetricID: "m.med.plus.abs.fail", Unit: "ms", Value: 550},
			baseline:       BaselineFile{BaselineValue: floatPtr(1000)},
			wantViolation:  true,
			wantReasonLike: "exceeds max",
		},
		{
			name:           "p95_ceiling_plus_regression checks regression baseline",
			metric:         MetricContract{MetricID: "m.p95.plus.reg.fail", Unit: "ms", BudgetClass: "hybrid-budget", ComparisonMethod: "p95_ceiling_plus_regression", Threshold: MetricThreshold{MaxValue: &max500, MaxRegressionPercent: &reg15}, NoiseFloor: 5},
			measurement:    MeasurementRecord{MetricID: "m.p95.plus.reg.fail", Unit: "ms", Value: 130},
			baseline:       medianBaseline(100),
			wantViolation:  true,
			wantReasonLike: "hybrid regression threshold exceeded",
		},
	}
}

func assertEvaluationResult(t *testing.T, tc comparisonMethodTestCase) {
	t.Helper()
	violations := Evaluate(
		CheckOutput{Measurements: []MeasurementRecord{tc.measurement}},
		[]ContractFile{{ContractID: "c", Metrics: []MetricContract{tc.metric}}},
		map[string]BaselineFile{tc.metric.MetricID: tc.baseline},
	)

	if tc.wantViolation && len(violations) == 0 {
		t.Fatalf("violations = %#v, want at least one violation", violations)
	}
	if !tc.wantViolation && len(violations) != 0 {
		t.Fatalf("violations = %#v, want no violations", violations)
	}
	if tc.wantReasonLike != "" && !containsViolationReason(violations, tc.metric.MetricID, tc.wantReasonLike) {
		t.Fatalf("violations = %#v, want reason containing %q", violations, tc.wantReasonLike)
	}
}

func containsViolationReason(violations []Violation, metricID, wantReasonLike string) bool {
	for _, v := range violations {
		if v.MetricID == metricID && strings.Contains(v.Reason, wantReasonLike) {
			return true
		}
	}
	return false
}

func medianBaseline(v float64) BaselineFile {
	return BaselineFile{Summary: struct {
		Median *float64 `json:"median,omitempty"`
	}{Median: floatPtr(v)}}
}

func floatPtr(v float64) *float64 { return &v }

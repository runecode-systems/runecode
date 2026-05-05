package perfcontracts

import "testing"

func TestEvaluateHonorsBudgetClasses(t *testing.T) {
	exact := 2.0
	absMax := 500.0
	reg := 15.0
	contracts := []ContractFile{{ContractID: "c", Metrics: []MetricContract{
		{MetricID: "m.exact", Unit: "count", BudgetClass: "exact", Threshold: MetricThreshold{ExactValue: &exact}},
		{MetricID: "m.abs", Unit: "ms", BudgetClass: "absolute-budget", Threshold: MetricThreshold{MaxValue: &absMax}},
		{MetricID: "m.reg", Unit: "ns/op", BudgetClass: "regression-budget", Threshold: MetricThreshold{MaxRegressionPercent: &reg}, NoiseFloor: 0.1},
	}}}
	check := CheckOutput{Measurements: []MeasurementRecord{{MetricID: "m.exact", Unit: "count", Value: 2}, {MetricID: "m.abs", Unit: "ms", Value: 499}, {MetricID: "m.reg", Unit: "ns/op", Value: 120}}}
	base := 100.0
	violations := Evaluate(check, contracts, map[string]BaselineFile{"m.reg": {BaselineValue: &base}})
	if len(violations) != 1 || violations[0].MetricID != "m.reg" {
		t.Fatalf("violations = %#v, want regression-only violation", violations)
	}
}

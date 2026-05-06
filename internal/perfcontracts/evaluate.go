package perfcontracts

import (
	"fmt"
	"math"
)

type Violation struct {
	MetricID string
	Reason   string
}

func Evaluate(check CheckOutput, contracts []ContractFile, baselineByMetric map[string]BaselineFile) []Violation {
	measurementByMetric := map[string]MeasurementRecord{}
	for _, measurement := range check.Measurements {
		measurementByMetric[measurement.MetricID] = measurement
	}
	var violations []Violation
	for _, contract := range contracts {
		for _, metric := range contract.Metrics {
			measurement, ok := measurementByMetric[metric.MetricID]
			if !ok {
				violations = append(violations, Violation{MetricID: metric.MetricID, Reason: "measurement missing from check output"})
				continue
			}
			if measurement.Unit != metric.Unit {
				violations = append(violations, Violation{MetricID: metric.MetricID, Reason: fmt.Sprintf("unit mismatch: got %s want %s", measurement.Unit, metric.Unit)})
				continue
			}
			violations = append(violations, evaluateMetric(metric, measurement.Value, baselineByMetric[metric.MetricID])...)
		}
	}
	return violations
}

func evaluateMetric(metric MetricContract, measured float64, baseline BaselineFile) []Violation {
	switch metric.ComparisonMethod {
	case "exact_match":
		return evaluateExactMetric(metric, measured)
	case "absolute_ceiling", "max_ceiling", "p95_ceiling", "window_average", "window_max":
		return evaluateAbsoluteBudgetMetric(metric, measured)
	case "median_regression_with_noise_floor":
		return evaluateRegressionMetric(metric, measured, baseline)
	case "median_plus_regression", "p95_ceiling_plus_regression":
		return evaluateHybridMetric(metric, measured, baseline)
	case "":
		return evaluateMetricByBudgetClass(metric, measured, baseline)
	default:
		return []Violation{{MetricID: metric.MetricID, Reason: fmt.Sprintf("unsupported comparison method %q", metric.ComparisonMethod)}}
	}
}

func evaluateMetricByBudgetClass(metric MetricContract, measured float64, baseline BaselineFile) []Violation {
	switch metric.BudgetClass {
	case "exact":
		return evaluateExactMetric(metric, measured)
	case "absolute-budget":
		return evaluateAbsoluteBudgetMetric(metric, measured)
	case "regression-budget":
		return evaluateRegressionMetric(metric, measured, baseline)
	case "hybrid-budget":
		return evaluateHybridMetric(metric, measured, baseline)
	default:
		return []Violation{{MetricID: metric.MetricID, Reason: "unsupported budget class"}}
	}
}

func evaluateExactMetric(metric MetricContract, measured float64) []Violation {
	if metric.Threshold.ExactValue == nil {
		return []Violation{{MetricID: metric.MetricID, Reason: "exact threshold missing exact_value"}}
	}
	if measured == *metric.Threshold.ExactValue {
		return nil
	}
	return []Violation{{MetricID: metric.MetricID, Reason: fmt.Sprintf("exact mismatch: got %.4f want %.4f", measured, *metric.Threshold.ExactValue)}}
}

func evaluateAbsoluteBudgetMetric(metric MetricContract, measured float64) []Violation {
	if metric.Threshold.MaxValue == nil {
		return []Violation{{MetricID: metric.MetricID, Reason: "absolute-budget threshold missing max_value"}}
	}
	if measured <= *metric.Threshold.MaxValue {
		return nil
	}
	return []Violation{{MetricID: metric.MetricID, Reason: fmt.Sprintf("value %.4f exceeds max %.4f", measured, *metric.Threshold.MaxValue)}}
}

func evaluateRegressionMetric(metric MetricContract, measured float64, baseline BaselineFile) []Violation {
	if !regressionViolation(metric, measured, baseline) {
		return nil
	}
	return []Violation{{MetricID: metric.MetricID, Reason: "regression threshold exceeded"}}
}

func evaluateHybridMetric(metric MetricContract, measured float64, baseline BaselineFile) []Violation {
	violations := evaluateAbsoluteBudgetMetric(metric, measured)
	if regressionViolation(metric, measured, baseline) {
		violations = append(violations, Violation{MetricID: metric.MetricID, Reason: "hybrid regression threshold exceeded"})
	}
	return violations
}

func regressionViolation(metric MetricContract, measured float64, baseline BaselineFile) bool {
	if metric.Threshold.MaxRegressionPercent == nil {
		return true
	}
	base, ok := baselineValue(baseline)
	if !ok || base == 0 {
		return true
	}
	delta := measured - base
	if delta <= 0 {
		return false
	}
	if delta < metric.NoiseFloor {
		return false
	}
	percent := (delta / base) * 100.0
	return percent > *metric.Threshold.MaxRegressionPercent
}

func baselineValue(file BaselineFile) (float64, bool) {
	if file.BaselineValue != nil {
		return *file.BaselineValue, true
	}
	if file.Summary.Median != nil {
		return *file.Summary.Median, true
	}
	if len(file.Samples) == 0 {
		return 0, false
	}
	return median(file.Samples), true
}

func median(values []float64) float64 {
	cp := append([]float64{}, values...)
	for i := 0; i < len(cp); i++ {
		for j := i + 1; j < len(cp); j++ {
			if cp[j] < cp[i] {
				cp[i], cp[j] = cp[j], cp[i]
			}
		}
	}
	m := len(cp) / 2
	if len(cp)%2 == 0 {
		return (cp[m-1] + cp[m]) / 2
	}
	return cp[m]
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

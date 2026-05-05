package tuiperf

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/runecode-ai/runecode/internal/perfcontracts"
)

var benchSuffixPattern = regexp.MustCompile(`-\d+$`)

type BenchmarkMetricMap struct {
	Benchmark string
	MetricID  string
	Unit      string
	Field     string
}

func ParseGoTestBenchOutput(r io.Reader, mappings []BenchmarkMetricMap) ([]perfcontracts.MeasurementRecord, error) {
	mapByKey := benchmarkMappingIndex(mappings)
	measurements := make([]perfcontracts.MeasurementRecord, 0, len(mappings))
	seen := map[string]struct{}{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields, ok := benchmarkFields(scanner.Text())
		if !ok {
			continue
		}
		benchName := fields[0]
		normalizedBench := benchSuffixPattern.ReplaceAllString(benchName, "")
		appendBenchMeasurements(&measurements, seen, mapByKey, benchName, normalizedBench, parseBenchMetrics(fields))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	for _, m := range mappings {
		if _, ok := seen[m.MetricID]; !ok {
			return nil, fmt.Errorf("missing benchmark measurement for %s", m.MetricID)
		}
	}
	return measurements, nil
}

func benchmarkMappingIndex(mappings []BenchmarkMetricMap) map[string]BenchmarkMetricMap {
	mapByKey := map[string]BenchmarkMetricMap{}
	for _, m := range mappings {
		k := strings.TrimSpace(m.Benchmark) + ":" + strings.TrimSpace(m.Field)
		mapByKey[k] = m
	}
	return mapByKey
}

func benchmarkFields(line string) ([]string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "Benchmark") {
		return nil, false
	}
	fields := strings.Fields(trimmed)
	if len(fields) < 3 {
		return nil, false
	}
	return fields, true
}

func appendBenchMeasurements(measurements *[]perfcontracts.MeasurementRecord, seen map[string]struct{}, mapByKey map[string]BenchmarkMetricMap, benchName, normalizedBench string, metrics map[string]float64) {
	for fieldName, val := range metrics {
		mapping, ok := resolveBenchMapping(mapByKey, benchName, normalizedBench, fieldName)
		if !ok || metricSeen(seen, mapping.MetricID) {
			continue
		}
		*measurements = append(*measurements, perfcontracts.MeasurementRecord{MetricID: mapping.MetricID, Value: val, Unit: mapping.Unit})
	}
}

func resolveBenchMapping(mapByKey map[string]BenchmarkMetricMap, benchName, normalizedBench, fieldName string) (BenchmarkMetricMap, bool) {
	mapping, ok := mapByKey[benchName+":"+fieldName]
	if ok {
		return mapping, true
	}
	mapping, ok = mapByKey[normalizedBench+":"+fieldName]
	return mapping, ok
}

func metricSeen(seen map[string]struct{}, metricID string) bool {
	if _, dup := seen[metricID]; dup {
		return true
	}
	seen[metricID] = struct{}{}
	return false
}

func parseBenchMetrics(fields []string) map[string]float64 {
	out := map[string]float64{}
	for i := 1; i+1 < len(fields); i++ {
		value, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			continue
		}
		unit := fields[i+1]
		switch unit {
		case "ns/op":
			out["ns/op"] = value
		case "B/op":
			out["B/op"] = value
		case "allocs/op":
			out["allocs/op"] = value
		}
	}
	return out
}

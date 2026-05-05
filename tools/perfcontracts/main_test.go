package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPassesForRequiredSharedLinuxMetrics(t *testing.T) {
	root := t.TempDir()
	writeContractsFixture(t, root)
	checkOutput := filepath.Join(root, "check.json")
	writeFile(t, checkOutput, `{"schema_version":"v1","measurements":[{"metric_id":"metric.tui.attach.latency.p95","value":420,"unit":"ms"}]}`)
	if err := run([]string{"--contracts-root", root, "--check-output", checkOutput, "--lane", "required_shared_linux"}); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
}

func TestRunFailsOnThresholdViolation(t *testing.T) {
	root := t.TempDir()
	writeContractsFixture(t, root)
	checkOutput := filepath.Join(root, "check.json")
	writeFile(t, checkOutput, `{"schema_version":"v1","measurements":[{"metric_id":"metric.tui.attach.latency.p95","value":999,"unit":"ms"}]}`)
	if err := run([]string{"--contracts-root", root, "--check-output", checkOutput, "--lane", "required_shared_linux"}); err == nil {
		t.Fatal("run error = nil, want threshold violation")
	}
}

func TestRunIgnoresInformationalAndPendingMetrics(t *testing.T) {
	root := t.TempDir()
	writeContractsFixture(t, root)
	checkOutput := filepath.Join(root, "check.json")
	writeFile(t, checkOutput, `{"schema_version":"v1","measurements":[{"metric_id":"metric.tui.attach.latency.p95","value":420,"unit":"ms"},{"metric_id":"metric.broker.watch.latency.p95","value":999,"unit":"ms"},{"metric_id":"metric.anchor.prepare.latency.p95","value":999,"unit":"ms"}]}`)
	if err := run([]string{"--contracts-root", root, "--check-output", checkOutput, "--lane", "required_shared_linux"}); err != nil {
		t.Fatalf("run returned error for non-required metrics: %v", err)
	}
}

func TestRunFiltersRequiredLaneByMetricID(t *testing.T) {
	root := t.TempDir()
	writeContractsFixture(t, root)
	checkOutput := filepath.Join(root, "check.json")
	writeFile(t, checkOutput, `{"schema_version":"v1","measurements":[{"metric_id":"metric.tui.attach.latency.p95","value":420,"unit":"ms"},{"metric_id":"metric.tui.key_response.quiet.p95_ms","value":999,"unit":"ms"}]}`)
	appendRequiredMetric(t, root)
	if err := run([]string{"--contracts-root", root, "--check-output", checkOutput, "--lane", "required_shared_linux", "--metric-id", "metric.tui.attach.latency.p95"}); err != nil {
		t.Fatalf("run returned error for filtered required metric: %v", err)
	}
}

func writeContractsFixture(t *testing.T, root string) {
	t.Helper()
	contractsDir, baselinesDir := createFixtureDirs(t, root)
	writeFixtureManifest(t, root)
	writeFixtureInventory(t, root)
	writeFixtureContract(t, contractsDir)
	writeFixtureBaseline(t, baselinesDir)
}

func createFixtureDirs(t *testing.T, root string) (string, string) {
	t.Helper()
	contractsDir := filepath.Join(root, "contracts")
	baselinesDir := filepath.Join(root, "baselines")
	if err := os.MkdirAll(contractsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll contracts: %v", err)
	}
	if err := os.MkdirAll(baselinesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll baselines: %v", err)
	}
	return contractsDir, baselinesDir
}

func writeFixtureManifest(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "manifest.json"), `{
		"schema_version":"runecode.performance.manifest.v1",
		"manifest_version":"1",
		"change_ref":"CHG-2026-053-9d2b-performance-baselines-verification-gates-v0",
		"fixture_inventory_ref":"fixtures.json",
		"contracts":[{"surface":"tui","path":"contracts/tui.json"}],
		"baselines":[{"metric_id":"metric.tui.render.ns","path":"baselines/metric.tui.render.ns.json"}],
		"metric_taxonomy":{"budget_classes":["exact","absolute-budget","regression-budget","hybrid-budget"]},
		"lane_authorities":["required_shared_linux","required_tight_linux","informational_until_stable","contract_pending_dependency","extended"],
		"activation_states":["defined","informational","required","contract_pending_dependency"]
	}`)
}

func writeFixtureInventory(t *testing.T, root string) {
	t.Helper()
	writeFile(t, filepath.Join(root, "fixtures.json"), `{
		"schema_version":"runecode.performance.fixtures.v1",
		"fixtures":[
			{"fixture_id":"tui.empty.v1","surface":"tui","runtime_regime":"empty","status":"mvp_reviewed"},
			{"fixture_id":"broker.watch.run.snapshot-follow.v1","surface":"broker","runtime_regime":"watch","status":"mvp_reviewed"},
			{"fixture_id":"anchor.fast-complete.stub.v1","surface":"external-anchor","runtime_regime":"prepare","status":"mvp_reviewed"}
		]
	}`)
}

func writeFixtureContract(t *testing.T, contractsDir string) {
	t.Helper()
	writeFile(t, filepath.Join(contractsDir, "tui.json"), fixtureContractJSON)
}

const fixtureContractJSON = `{
		"schema_version":"runecode.performance.contract.v1",
		"contract_id":"performance.tui.v1",
		"surface":"tui",
		"metrics":[
			{
				"metric_id":"metric.tui.attach.latency.p95",
				"subsystem":"tui",
				"runtime_regime":"attach",
				"fixture_id":"tui.empty.v1",
				"measurement_kind":"latency",
				"unit":"ms",
				"authoritative_environment":"linux_shared_ci",
				"sampling_policy":{"trials":30,"p95_authoritative":true},
				"budget_class":"absolute-budget",
				"threshold":{"max_value":500},
				"lane_authority":"required_shared_linux",
				"activation_state":"required",
				"comparison_method":"p95_ceiling",
				"threshold_origin":"product_budget",
				"timing_boundary":{"start_event":"tui.process.spawn","end_event":"broker.attach.ready","clock_source":"monotonic","evidence_source":"pty_transcript","included_phases":["launch","attach"]}
			},
			{
				"metric_id":"metric.broker.watch.latency.p95",
				"subsystem":"broker",
				"runtime_regime":"watch",
				"fixture_id":"broker.watch.run.snapshot-follow.v1",
				"measurement_kind":"latency",
				"unit":"ms",
				"authoritative_environment":"linux_shared_ci",
				"sampling_policy":{"trials":30,"p95_authoritative":true},
				"budget_class":"absolute-budget",
				"threshold":{"max_value":200},
				"lane_authority":"informational_until_stable",
				"activation_state":"informational",
				"comparison_method":"p95_ceiling",
				"threshold_origin":"first_calibration",
				"timing_boundary":{"start_event":"rpc.request_sent","end_event":"watch.snapshot_follow_received","clock_source":"monotonic","evidence_source":"broker_events","included_phases":["watch"]}
			},
			{
				"metric_id":"metric.anchor.prepare.latency.p95",
				"subsystem":"external-anchor",
				"runtime_regime":"prepare",
				"fixture_id":"anchor.fast-complete.stub.v1",
				"measurement_kind":"latency",
				"unit":"ms",
				"authoritative_environment":"linux_shared_ci",
				"sampling_policy":{"trials":30,"p95_authoritative":true},
				"budget_class":"absolute-budget",
				"threshold":{"max_value":500},
				"lane_authority":"contract_pending_dependency",
				"activation_state":"contract_pending_dependency",
				"comparison_method":"p95_ceiling",
				"threshold_origin":"temporary_guardrail",
				"timing_boundary":{"start_event":"anchor.prepare.begin","end_event":"anchor.prepare.persisted","clock_source":"monotonic","evidence_source":"broker_events","included_phases":["prepare"]}
			}
		]
	}`

func writeFixtureBaseline(t *testing.T, baselinesDir string) {
	t.Helper()
	writeFile(t, filepath.Join(baselinesDir, "metric.tui.render.ns.json"), `{"schema_version":"runecode.performance.baseline.v1","metric_id":"metric.tui.render.ns","unit":"ns/op","samples":[100,101,99],"summary":{"median":100}}`)
}

func appendRequiredMetric(t *testing.T, root string) {
	t.Helper()
	contractPath := filepath.Join(root, "contracts", "tui.json")
	raw, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", contractPath, err)
	}
	content := strings.TrimSpace(string(raw))
	content = strings.TrimSuffix(content, "}")
	content = strings.TrimSpace(content)
	content = strings.TrimSuffix(content, "]") + `,
			{
				"metric_id":"metric.tui.key_response.quiet.p95_ms",
				"subsystem":"tui",
				"runtime_regime":"key_response_quiet",
				"fixture_id":"tui.empty.v1",
				"measurement_kind":"latency",
				"unit":"ms",
				"authoritative_environment":"linux_shared_ci",
				"sampling_policy":{"trials":30,"p95_authoritative":true},
				"budget_class":"absolute-budget",
				"threshold":{"max_value":50},
				"lane_authority":"required_shared_linux",
				"activation_state":"required",
				"comparison_method":"p95_ceiling",
				"threshold_origin":"product_budget",
				"timing_boundary":{"start_event":"tui.key.injected","end_event":"broker.attach.ready.frame_delta","clock_source":"monotonic","evidence_source":"pty_transcript","included_phases":["input_dispatch","render"]}
			}
		]
	}`
	writeFile(t, contractPath, content)
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error: %v", path, err)
	}
}

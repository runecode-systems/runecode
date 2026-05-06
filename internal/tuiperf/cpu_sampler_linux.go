//go:build linux

package tuiperf

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type CPUSampleConfig struct {
	Warmup          time.Duration
	Window          time.Duration
	Windows         int
	TicksPerSecond  float64
	CPUCount        float64
	ProcRoot        string
	ExpectedComm    string
	ChildLookupWait time.Duration
	PollInterval    time.Duration
}

type CPUWindowSample struct {
	Index      int     `json:"index"`
	CPUPercent float64 `json:"cpu_percent"`
}

type CPUSampleResult struct {
	TargetPID          int               `json:"target_pid"`
	TargetComm         string            `json:"target_comm"`
	WarmupMillis       int64             `json:"warmup_millis"`
	WindowMillis       int64             `json:"observation_window_millis"`
	ObservationWindows int               `json:"observation_windows"`
	AverageCPUPercent  float64           `json:"average_cpu_percent"`
	MaxCPUPercent      float64           `json:"max_cpu_percent"`
	Windows            []CPUWindowSample `json:"windows"`
}

func DefaultCPUSampleConfig() CPUSampleConfig {
	return CPUSampleConfig{
		Warmup:          3 * time.Second,
		Window:          20 * time.Second,
		Windows:         3,
		ProcRoot:        "/proc",
		ExpectedComm:    "runecode-tui",
		ChildLookupWait: 3 * time.Second,
		PollInterval:    20 * time.Millisecond,
	}
}

func (c *CPUSampleConfig) normalize() error {
	c.applyDefaults()
	if err := c.validateDurations(); err != nil {
		return err
	}
	if err := c.resolveTicksPerSecond(); err != nil {
		return err
	}
	return c.resolveCPUCount()
}

func (c *CPUSampleConfig) applyDefaults() {
	if c.ProcRoot == "" {
		c.ProcRoot = "/proc"
	}
	if c.ExpectedComm == "" {
		c.ExpectedComm = "runecode-tui"
	}
	if c.ChildLookupWait <= 0 {
		c.ChildLookupWait = 3 * time.Second
	}
	if c.PollInterval <= 0 {
		c.PollInterval = 20 * time.Millisecond
	}
}

func (c *CPUSampleConfig) validateDurations() error {
	if c.Warmup < 0 || c.Window <= 0 || c.Windows <= 0 {
		return fmt.Errorf("invalid cpu sampling durations")
	}
	return nil
}

func (c *CPUSampleConfig) resolveTicksPerSecond() error {
	if c.TicksPerSecond > 0 {
		return nil
	}
	tps, err := systemTicksPerSecond()
	if err != nil {
		return err
	}
	c.TicksPerSecond = tps
	return nil
}

func (c *CPUSampleConfig) resolveCPUCount() error {
	if c.CPUCount > 0 {
		return nil
	}
	cpus, err := cpuCountFromProc(c.ProcRoot)
	if err != nil {
		return err
	}
	c.CPUCount = cpus
	return nil
}

func systemTicksPerSecond() (float64, error) {
	v := os.Getenv("RUNECODE_TUIPERF_CLK_TCK")
	if strings.TrimSpace(v) != "" {
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil {
			return 0, fmt.Errorf("parse RUNECODE_TUIPERF_CLK_TCK: %w", err)
		}
		if f <= 0 {
			return 0, fmt.Errorf("RUNECODE_TUIPERF_CLK_TCK must be > 0")
		}
		return f, nil
	}
	return 100.0, nil
}

func cpuCountFromProc(procRoot string) (float64, error) {
	raw, err := os.ReadFile(procRoot + "/stat")
	if err != nil {
		return 0, err
	}
	count := 0
	for _, line := range strings.Split(string(raw), "\n") {
		if len(line) < 4 {
			continue
		}
		if strings.HasPrefix(line, "cpu") && len(line) > 3 && line[3] >= '0' && line[3] <= '9' {
			count++
		}
	}
	if count <= 0 {
		return 0, fmt.Errorf("failed to detect cpu count")
	}
	return float64(count), nil
}

func WaitForChildByComm(procRoot string, wrapperPID int, comm string, timeout time.Duration, pollInterval time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	for {
		pid, err := FindDescendantByComm(procRoot, wrapperPID, comm)
		if err == nil {
			return pid, nil
		}
		if time.Now().After(deadline) {
			return 0, err
		}
		time.Sleep(pollInterval)
	}
}

func SampleProcessCPU(targetPID int, cfg CPUSampleConfig) (CPUSampleResult, error) {
	if err := cfg.normalize(); err != nil {
		return CPUSampleResult{}, err
	}
	cpuWindow, err := collectCPUWindows(targetPID, cfg)
	if err != nil {
		return CPUSampleResult{}, err
	}
	return cpuWindow.toResult(targetPID, cfg), nil
}

type cpuWindowCollection struct {
	targetComm string
	windows    []CPUWindowSample
	sum        float64
	max        float64
}

func collectCPUWindows(targetPID int, cfg CPUSampleConfig) (cpuWindowCollection, error) {
	if cfg.Warmup > 0 {
		time.Sleep(cfg.Warmup)
	}
	out := cpuWindowCollection{windows: make([]CPUWindowSample, 0, cfg.Windows)}
	for i := 0; i < cfg.Windows; i++ {
		sample, comm, err := sampleCPUWindow(targetPID, cfg, i+1)
		if err != nil {
			return cpuWindowCollection{}, err
		}
		if out.targetComm == "" {
			out.targetComm = comm
		}
		if sample.CPUPercent > out.max || i == 0 {
			out.max = sample.CPUPercent
		}
		out.sum += sample.CPUPercent
		out.windows = append(out.windows, sample)
	}
	return out, nil
}

func sampleCPUWindow(targetPID int, cfg CPUSampleConfig, index int) (CPUWindowSample, string, error) {
	before, err := ReadProcStat(cfg.ProcRoot, targetPID)
	if err != nil {
		return CPUWindowSample{}, "", err
	}
	time.Sleep(cfg.Window)
	after, err := ReadProcStat(cfg.ProcRoot, targetPID)
	if err != nil {
		return CPUWindowSample{}, "", err
	}
	cpu := cpuPercentForWindow(before.TotalTicks(), after.TotalTicks(), cfg.Window, cfg.TicksPerSecond, cfg.CPUCount)
	return CPUWindowSample{Index: index, CPUPercent: cpu}, before.Comm, nil
}

func cpuPercentForWindow(beforeTicks, afterTicks uint64, window time.Duration, ticksPerSecond, cpuCount float64) float64 {
	deltaTicks := float64(afterTicks - beforeTicks)
	deltaSeconds := window.Seconds()
	cpu := (deltaTicks / ticksPerSecond / deltaSeconds / cpuCount) * 100.0
	if cpu < 0 {
		return 0
	}
	return cpu
}

func (c cpuWindowCollection) toResult(targetPID int, cfg CPUSampleConfig) CPUSampleResult {
	return CPUSampleResult{
		TargetPID:          targetPID,
		TargetComm:         c.targetComm,
		WarmupMillis:       cfg.Warmup.Milliseconds(),
		WindowMillis:       cfg.Window.Milliseconds(),
		ObservationWindows: cfg.Windows,
		AverageCPUPercent:  c.sum / float64(cfg.Windows),
		MaxCPUPercent:      c.max,
		Windows:            c.windows,
	}
}

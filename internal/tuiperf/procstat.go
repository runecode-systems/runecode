package tuiperf

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ProcStat struct {
	PID         int
	Comm        string
	UserTicks   uint64
	SystemTicks uint64
	StartTicks  uint64
}

func (p ProcStat) TotalTicks() uint64 {
	return p.UserTicks + p.SystemTicks
}

func ReadProcStat(procRoot string, pid int) (ProcStat, error) {
	raw, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
	if err != nil {
		return ProcStat{}, err
	}
	return ParseProcStatLine(strings.TrimSpace(string(raw)))
}

func ParseProcStatLine(line string) (ProcStat, error) {
	open := strings.Index(line, "(")
	close := strings.LastIndex(line, ")")
	if open <= 0 || close <= open {
		return ProcStat{}, fmt.Errorf("invalid /proc stat format")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(line[:open]))
	if err != nil {
		return ProcStat{}, fmt.Errorf("parse pid: %w", err)
	}
	comm := line[open+1 : close]
	fields := strings.Fields(strings.TrimSpace(line[close+1:]))
	if len(fields) < 20 {
		return ProcStat{}, fmt.Errorf("invalid /proc stat field count")
	}
	userTicks, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return ProcStat{}, fmt.Errorf("parse utime: %w", err)
	}
	systemTicks, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return ProcStat{}, fmt.Errorf("parse stime: %w", err)
	}
	startTicks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return ProcStat{}, fmt.Errorf("parse starttime: %w", err)
	}
	return ProcStat{PID: pid, Comm: comm, UserTicks: userTicks, SystemTicks: systemTicks, StartTicks: startTicks}, nil
}

func FindDescendantByComm(procRoot string, rootPID int, comm string) (int, error) {
	want := strings.TrimSpace(comm)
	if want == "" {
		return 0, fmt.Errorf("comm is required")
	}
	search := procTreeSearch{procRoot: procRoot, wantComm: want, queue: []int{rootPID}, seen: map[int]struct{}{rootPID: {}}}
	for search.hasQueue() {
		if pid, found := search.scanNext(); found {
			return pid, nil
		}
	}
	return 0, fmt.Errorf("descendant with comm %q not found", want)
}

type procTreeSearch struct {
	procRoot string
	wantComm string
	queue    []int
	seen     map[int]struct{}
}

func (s *procTreeSearch) hasQueue() bool {
	return len(s.queue) > 0
}

func (s *procTreeSearch) popQueue() int {
	pid := s.queue[0]
	s.queue = s.queue[1:]
	return pid
}

func (s *procTreeSearch) scanNext() (int, bool) {
	pid := s.popQueue()
	children, err := readChildren(s.procRoot, pid)
	if err != nil {
		return 0, false
	}
	for _, child := range children {
		if s.markSeen(child) {
			continue
		}
		if s.childMatches(child) {
			return child, true
		}
		s.queue = append(s.queue, child)
	}
	return 0, false
}

func (s *procTreeSearch) markSeen(child int) bool {
	if _, ok := s.seen[child]; ok {
		return true
	}
	s.seen[child] = struct{}{}
	return false
}

func (s *procTreeSearch) childMatches(child int) bool {
	childComm, err := os.ReadFile(filepath.Join(s.procRoot, strconv.Itoa(child), "comm"))
	return err == nil && strings.TrimSpace(string(childComm)) == s.wantComm
}

func readChildren(procRoot string, pid int) ([]int, error) {
	raw, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "task", strconv.Itoa(pid), "children"))
	if err != nil {
		return nil, err
	}
	parts := strings.Fields(string(raw))
	out := make([]int, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

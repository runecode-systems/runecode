package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type workbenchLocalState struct {
	SidebarVisible     bool
	InspectorVisible   bool
	InspectorMode      contentPresentationMode
	ThemePreset        themePreset
	LeaderKey          string
	LastRouteID        routeID
	LastSessionID      string
	LastSessionByWS    map[string]string
	PinnedSessions     []workbenchSessionRef
	RecentSessions     []workbenchSessionRef
	RecentObjects      []workbenchObjectRef
	ViewedActivity     map[string]string
	SidebarPaneRatio   float64
	InspectorPaneRatio float64
	SidebarCollapsed   bool
	InspectorCollapsed bool
}

type shellWorkbenchStateStore interface {
	Read(targetKey string) workbenchLocalState
	Write(targetKey string, next workbenchLocalState)
}

type shellWorkbenchStateFlusher interface {
	Flush()
}

type shellWorkbenchStateErrorReporter interface {
	LastError() error
}

type memoryWorkbenchStateStore struct {
	states map[string]workbenchLocalState
}

func (s *memoryWorkbenchStateStore) Read(targetKey string) workbenchLocalState {
	if s == nil || strings.TrimSpace(targetKey) == "" {
		return workbenchLocalState{}
	}
	if s.states == nil {
		return workbenchLocalState{}
	}
	return cloneWorkbenchLocalState(s.states[targetKey])
}

func (s *memoryWorkbenchStateStore) Write(targetKey string, next workbenchLocalState) {
	if s == nil || strings.TrimSpace(targetKey) == "" {
		return
	}
	if s.states == nil {
		s.states = map[string]workbenchLocalState{}
	}
	s.states[targetKey] = cloneWorkbenchLocalState(next)
}

func (s *memoryWorkbenchStateStore) Flush() {}

type workbenchSessionRef struct {
	WorkspaceID string `json:"workspace_id"`
	SessionID   string `json:"session_id"`
}

type workbenchObjectRef struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
}

type workbenchPersistenceEnvelope struct {
	SchemaVersion int                            `json:"schema_version"`
	Targets       map[string]workbenchLocalState `json:"targets"`
}

const (
	workbenchPersistenceSchemaVersion = 1
	workbenchPersistenceDebounce      = 25 * time.Millisecond
)

type fileWorkbenchStateStore struct {
	path string

	mu             sync.Mutex
	cond           *sync.Cond
	loaded         bool
	env            workbenchPersistenceEnvelope
	dirty          bool
	flushScheduled bool
	flushing       bool
	lastErr        error

	writeDebounce time.Duration
	writeDelay    func()
}

func newDefaultWorkbenchStateStore() shellWorkbenchStateStore {
	configRoot, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(configRoot) == "" {
		return &memoryWorkbenchStateStore{}
	}
	return newFileWorkbenchStateStore(filepath.Join(configRoot, "runecode", "tui", "workbench-state.json"))
}

func newFileWorkbenchStateStore(path string) *fileWorkbenchStateStore {
	store := &fileWorkbenchStateStore{path: path, writeDebounce: workbenchPersistenceDebounce}
	store.cond = sync.NewCond(&store.mu)
	return store
}

func (s *fileWorkbenchStateStore) Read(targetKey string) workbenchLocalState {
	targetKey = strings.TrimSpace(targetKey)
	if s == nil || targetKey == "" {
		return workbenchLocalState{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLoadedLocked()
	if s.env.Targets == nil {
		return workbenchLocalState{}
	}
	return cloneWorkbenchLocalState(s.env.Targets[targetKey])
}

func (s *fileWorkbenchStateStore) Write(targetKey string, next workbenchLocalState) {
	targetKey = strings.TrimSpace(targetKey)
	if s == nil || targetKey == "" || strings.TrimSpace(s.path) == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLoadedLocked()
	if s.env.Targets == nil {
		s.env.Targets = map[string]workbenchLocalState{}
	}
	s.env.SchemaVersion = workbenchPersistenceSchemaVersion
	s.env.Targets[targetKey] = cloneWorkbenchLocalState(next)
	s.dirty = true
	if !s.flushScheduled && !s.flushing {
		s.flushScheduled = true
		go s.flushLoop()
	}
	s.cond.Broadcast()
}

func (s *fileWorkbenchStateStore) Flush() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLoadedLocked()
	if s.dirty && !s.flushScheduled && !s.flushing {
		s.flushScheduled = true
		go s.flushLoop()
	}
	for s.flushScheduled || s.flushing {
		s.cond.Wait()
	}
}

func (s *fileWorkbenchStateStore) LastError() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastErr
}

func (s *fileWorkbenchStateStore) ensureLoadedLocked() {
	if s.cond == nil {
		s.cond = sync.NewCond(&s.mu)
	}
	if s.loaded {
		if s.env.Targets == nil {
			s.env = workbenchPersistenceEnvelope{SchemaVersion: workbenchPersistenceSchemaVersion, Targets: map[string]workbenchLocalState{}}
		}
		return
	}
	s.env = s.readEnvelopeFromDisk()
	s.loaded = true
}

func (s *fileWorkbenchStateStore) flushLoop() {
	debounce := s.writeDebounce
	if debounce > 0 {
		time.Sleep(debounce)
	}
	for {
		env, delay, ok := s.beginFlushCycle()
		if !ok {
			return
		}

		if delay != nil {
			delay()
		}
		err := s.persistEnvelope(env)

		pending := s.finishFlushCycle(err)
		if !pending {
			return
		}
		if debounce > 0 {
			time.Sleep(debounce)
		}
	}
}

func (s *fileWorkbenchStateStore) beginFlushCycle() (workbenchPersistenceEnvelope, func(), bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLoadedLocked()
	if !s.dirty {
		s.flushScheduled = false
		s.cond.Broadcast()
		return workbenchPersistenceEnvelope{}, nil, false
	}
	env := cloneWorkbenchPersistenceEnvelope(s.env)
	s.dirty = false
	s.flushScheduled = false
	s.flushing = true
	return env, s.writeDelay, true
}

func (s *fileWorkbenchStateStore) finishFlushCycle(err error) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.dirty = true
		s.lastErr = err
	} else {
		s.lastErr = nil
	}
	s.flushing = false
	pending := s.dirty
	if pending && err == nil {
		s.flushScheduled = true
	}
	s.cond.Broadcast()
	return pending
}

func (s *fileWorkbenchStateStore) persistEnvelope(env workbenchPersistenceEnvelope) error {
	if strings.TrimSpace(s.path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0o600)
}

func (s *fileWorkbenchStateStore) readEnvelopeFromDisk() workbenchPersistenceEnvelope {
	if strings.TrimSpace(s.path) == "" {
		return workbenchPersistenceEnvelope{SchemaVersion: workbenchPersistenceSchemaVersion, Targets: map[string]workbenchLocalState{}}
	}
	raw, err := os.ReadFile(s.path)
	if err != nil || len(raw) == 0 {
		return workbenchPersistenceEnvelope{SchemaVersion: workbenchPersistenceSchemaVersion, Targets: map[string]workbenchLocalState{}}
	}
	var env workbenchPersistenceEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return workbenchPersistenceEnvelope{SchemaVersion: workbenchPersistenceSchemaVersion, Targets: map[string]workbenchLocalState{}}
	}
	if env.Targets == nil {
		env.Targets = map[string]workbenchLocalState{}
	}
	if env.SchemaVersion <= 0 {
		env.SchemaVersion = workbenchPersistenceSchemaVersion
	}
	if env.SchemaVersion > workbenchPersistenceSchemaVersion {
		return workbenchPersistenceEnvelope{SchemaVersion: workbenchPersistenceSchemaVersion, Targets: map[string]workbenchLocalState{}}
	}
	return cloneWorkbenchPersistenceEnvelope(env)
}

func cloneWorkbenchPersistenceEnvelope(env workbenchPersistenceEnvelope) workbenchPersistenceEnvelope {
	out := workbenchPersistenceEnvelope{
		SchemaVersion: env.SchemaVersion,
		Targets:       make(map[string]workbenchLocalState, len(env.Targets)),
	}
	if out.SchemaVersion <= 0 {
		out.SchemaVersion = workbenchPersistenceSchemaVersion
	}
	for key, state := range env.Targets {
		out.Targets[key] = cloneWorkbenchLocalState(state)
	}
	return out
}

func cloneWorkbenchLocalState(state workbenchLocalState) workbenchLocalState {
	state.LastSessionByWS = cloneSessionMap(state.LastSessionByWS)
	state.PinnedSessions = append([]workbenchSessionRef(nil), state.PinnedSessions...)
	state.RecentSessions = append([]workbenchSessionRef(nil), state.RecentSessions...)
	state.RecentObjects = append([]workbenchObjectRef(nil), state.RecentObjects...)
	state.ViewedActivity = cloneViewedActivity(state.ViewedActivity)
	return state
}

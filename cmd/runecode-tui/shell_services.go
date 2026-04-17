package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

type shellFocusManager struct {
	current focusArea
}

func newShellFocusManager(initial focusArea) shellFocusManager {
	return shellFocusManager{current: initial}
}

func (m *shellFocusManager) Current() focusArea {
	return m.current
}

func (m *shellFocusManager) Set(area focusArea) {
	m.current = area
}

func (m *shellFocusManager) Next(layout shellLayoutPlan, overlayOpen bool) {
	if overlayOpen {
		m.current = focusPalette
		return
	}
	order := shellFocusOrder(layout)
	if len(order) == 0 {
		m.current = focusContent
		return
	}
	idx := shellFocusIndex(order, m.current)
	if idx < 0 {
		m.current = order[0]
		return
	}
	m.current = order[(idx+1)%len(order)]
}

func (m *shellFocusManager) Prev(layout shellLayoutPlan, overlayOpen bool) {
	if overlayOpen {
		m.current = focusPalette
		return
	}
	order := shellFocusOrder(layout)
	if len(order) == 0 {
		m.current = focusContent
		return
	}
	idx := shellFocusIndex(order, m.current)
	if idx < 0 {
		m.current = order[len(order)-1]
		return
	}
	idx--
	if idx < 0 {
		idx = len(order) - 1
	}
	m.current = order[idx]
}

func (m *shellFocusManager) Normalize(layout shellLayoutPlan, overlayOpen bool) {
	if overlayOpen {
		m.current = focusPalette
		return
	}
	order := shellFocusOrder(layout)
	if len(order) == 0 {
		m.current = focusContent
		return
	}
	if shellFocusIndex(order, m.current) >= 0 {
		return
	}
	m.current = order[0]
}

func shellFocusOrder(layout shellLayoutPlan) []focusArea {
	order := []focusArea{}
	if layout.NavigationVisible {
		order = append(order, focusNav)
	}
	order = append(order, focusContent)
	if layout.InspectorVisible {
		order = append(order, focusInspector)
	}
	return order
}

func shellFocusIndex(order []focusArea, target focusArea) int {
	for i, area := range order {
		if area == target {
			return i
		}
	}
	return -1
}

type shellOverlayManager struct {
	stack []shellOverlayID
}

func (m *shellOverlayManager) Open(id shellOverlayID) {
	if id == "" {
		return
	}
	for _, existing := range m.stack {
		if existing == id {
			return
		}
	}
	m.stack = append(m.stack, id)
}

func (m *shellOverlayManager) Close(id shellOverlayID) {
	if id == "" || len(m.stack) == 0 {
		return
	}
	filtered := make([]shellOverlayID, 0, len(m.stack))
	for _, existing := range m.stack {
		if existing != id {
			filtered = append(filtered, existing)
		}
	}
	m.stack = filtered
}

func (m *shellOverlayManager) Replace(ids ...shellOverlayID) {
	m.stack = m.stack[:0]
	for _, id := range ids {
		m.Open(id)
	}
}

func (m *shellOverlayManager) Contains(id shellOverlayID) bool {
	for _, existing := range m.stack {
		if existing == id {
			return true
		}
	}
	return false
}

func (m *shellOverlayManager) Stack() []shellOverlayID {
	out := make([]shellOverlayID, len(m.stack))
	copy(out, m.stack)
	return out
}

type shellCommand struct {
	ID          string
	Title       string
	Description string
	Run         func(*shellModel)
	PostRun     func(*shellModel) tea.Cmd
}

type shellCommandRegistry struct {
	commands map[string]shellCommand
	order    []string
}

func newShellCommandRegistry() shellCommandRegistry {
	return shellCommandRegistry{commands: map[string]shellCommand{}}
}

func (r *shellCommandRegistry) Register(cmd shellCommand) {
	if strings.TrimSpace(cmd.ID) == "" || cmd.Run == nil {
		return
	}
	if _, exists := r.commands[cmd.ID]; !exists {
		r.order = append(r.order, cmd.ID)
	}
	r.commands[cmd.ID] = cmd
}

func (r *shellCommandRegistry) List() []shellCommand {
	out := make([]shellCommand, 0, len(r.commands))
	for _, id := range r.order {
		cmd, ok := r.commands[id]
		if ok {
			out = append(out, cmd)
		}
	}
	if len(out) == 0 {
		return out
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Title < out[j].Title
	})
	return out
}

func (r *shellCommandRegistry) Execute(id string, model *shellModel) tea.Cmd {
	cmd, ok := r.commands[id]
	if !ok || cmd.Run == nil || model == nil {
		return nil
	}
	cmd.Run(model)
	if cmd.PostRun == nil {
		return nil
	}
	return cmd.PostRun(model)
}

type shellClipboardService interface {
	Copy(text string)
	Last() string
	IntegrationHint() string
}

type memoryClipboardService struct {
	last        string
	osc52       bool
	osc52Writer io.Writer
}

func (m *memoryClipboardService) Copy(text string) {
	text = strings.TrimSpace(redactSecrets(text))
	m.last = text
	if !m.osc52 || m.osc52Writer == nil || strings.TrimSpace(text) == "" {
		return
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	_, _ = io.WriteString(m.osc52Writer, "\x1b]52;c;"+encoded+"\x07")
}

func (m *memoryClipboardService) Last() string {
	return m.last
}

func (m *memoryClipboardService) IntegrationHint() string {
	if m.osc52 {
		return "shell clipboard + OSC52"
	}
	return "shell clipboard"
}

func newShellClipboardService() shellClipboardService {
	clip := &memoryClipboardService{}
	if term.IsTerminal(int(os.Stdout.Fd())) && osc52EnabledByEnv(os.Getenv("RUNECODE_TUI_OSC52")) {
		clip.osc52 = true
		clip.osc52Writer = os.Stdout
	}
	return clip
}

func osc52EnabledByEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

type workbenchLocalState struct {
	SidebarVisible     bool
	InspectorVisible   bool
	InspectorMode      contentPresentationMode
	ThemePreset        themePreset
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
	return s.states[targetKey]
}

func (s *memoryWorkbenchStateStore) Write(targetKey string, next workbenchLocalState) {
	if s == nil || strings.TrimSpace(targetKey) == "" {
		return
	}
	if s.states == nil {
		s.states = map[string]workbenchLocalState{}
	}
	s.states[targetKey] = next
}

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

const workbenchPersistenceSchemaVersion = 1

type fileWorkbenchStateStore struct {
	path string
}

func newDefaultWorkbenchStateStore() shellWorkbenchStateStore {
	configRoot, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(configRoot) == "" {
		return &memoryWorkbenchStateStore{}
	}
	return &fileWorkbenchStateStore{path: filepath.Join(configRoot, "runecode", "tui", "workbench-state.json")}
}

func (s *fileWorkbenchStateStore) Read(targetKey string) workbenchLocalState {
	env := s.readEnvelope()
	if env.Targets == nil {
		return workbenchLocalState{}
	}
	return env.Targets[strings.TrimSpace(targetKey)]
}

func (s *fileWorkbenchStateStore) Write(targetKey string, next workbenchLocalState) {
	targetKey = strings.TrimSpace(targetKey)
	if targetKey == "" || strings.TrimSpace(s.path) == "" {
		return
	}
	env := s.readEnvelope()
	if env.Targets == nil {
		env.Targets = map[string]workbenchLocalState{}
	}
	env.SchemaVersion = workbenchPersistenceSchemaVersion
	env.Targets[targetKey] = next
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return
	}
	raw, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, raw, 0o600)
}

func (s *fileWorkbenchStateStore) readEnvelope() workbenchPersistenceEnvelope {
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
	return env
}

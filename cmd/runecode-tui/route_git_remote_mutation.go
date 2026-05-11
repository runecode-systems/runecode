package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-systems/runecode/internal/brokerapi"
	"github.com/runecode-systems/runecode/internal/trustpolicy"
)

type gitRemoteMutationLoadedMsg struct {
	resp brokerapi.GitRemoteMutationGetResponse
	err  error
	seq  uint64
}

type gitRemoteMutationExecutedMsg struct {
	resp brokerapi.GitRemoteMutationExecuteResponse
	err  error
}

type gitRemoteMutationLeaseIssuedMsg struct {
	resp brokerapi.GitRemoteMutationIssueExecuteLeaseResponse
	err  error
}

type gitRemoteMutationRouteModel struct {
	def                 routeDefinition
	client              localBrokerClient
	loading             bool
	executing           bool
	errText             string
	status              string
	loadSeq             uint64
	prepared            brokerapi.GitRemoteMutationPreparedState
	request             brokerapi.GitRemoteMutationGetRequest
	providerAuthLeaseID string
}

func newGitRemoteMutationRouteModel(def routeDefinition, client localBrokerClient) routeModel {
	return gitRemoteMutationRouteModel{
		def:                 def,
		client:              client,
		request:             brokerapi.GitRemoteMutationGetRequest{PreparedMutationID: "sha256:" + strings.Repeat("1", 64)},
		providerAuthLeaseID: "",
	}
}

func (m gitRemoteMutationRouteModel) ID() routeID { return m.def.ID }

func (m gitRemoteMutationRouteModel) Title() string { return m.def.Label }

func (m gitRemoteMutationRouteModel) Update(msg tea.Msg) (routeModel, tea.Cmd) {
	switch typed := msg.(type) {
	case routeActivatedMsg:
		if typed.RouteID != m.def.ID {
			return m, nil
		}
		m = m.beginLoad()
		return m, m.loadCmd(m.loadSeq)
	case tea.KeyMsg:
		return m.handleKey(typed.String())
	case gitRemoteMutationLoadedMsg:
		return m.handleLoaded(typed)
	case gitRemoteMutationLeaseIssuedMsg:
		return m.handleLeaseIssued(typed)
	case gitRemoteMutationExecutedMsg:
		return m.handleExecuted(typed)
	default:
		return m, nil
	}
}

func (m gitRemoteMutationRouteModel) handleKey(key string) (routeModel, tea.Cmd) {
	switch key {
	case "r":
		m = m.beginLoad()
		return m, m.loadCmd(m.loadSeq)
	case "e":
		if m.loading || m.executing {
			return m, nil
		}
		leaseReq, err := m.buildIssueExecuteLeaseRequest()
		if err != nil {
			m.errText = safeUIErrorText(err)
			m.status = ""
			return m, nil
		}
		m.executing = true
		m.errText = ""
		m.status = "Issuing a broker-bound provider auth lease for this prepared mutation..."
		return m, m.issueExecuteLeaseCmd(leaseReq)
	default:
		return m, nil
	}
}

func (m gitRemoteMutationRouteModel) handleLoaded(msg gitRemoteMutationLoadedMsg) (routeModel, tea.Cmd) {
	if msg.seq != m.loadSeq {
		return m, nil
	}
	m.loading = false
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		return m, nil
	}
	m.errText = ""
	if !gitRemotePreparedBindingMatches(m.prepared, msg.resp.Prepared) {
		m.providerAuthLeaseID = ""
	}
	m.prepared = msg.resp.Prepared
	if m.status == "" {
		m.status = "Review the prepared change, confirm approval evidence, then press e for guarded broker execution."
	}
	return m, nil
}

func (m gitRemoteMutationRouteModel) handleExecuted(msg gitRemoteMutationExecutedMsg) (routeModel, tea.Cmd) {
	m.executing = false
	m.providerAuthLeaseID = ""
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		m.status = ""
		return m, nil
	}
	m.errText = ""
	m.prepared = msg.resp.Prepared
	m.status = "Remote execution completed; refreshing prepared state from the broker."
	m = m.beginLoad()
	return m, m.loadCmd(m.loadSeq)
}

func (m gitRemoteMutationRouteModel) handleLeaseIssued(msg gitRemoteMutationLeaseIssuedMsg) (routeModel, tea.Cmd) {
	if msg.err != nil {
		m.executing = false
		m.errText = safeUIErrorText(msg.err)
		m.status = ""
		return m, nil
	}
	m.providerAuthLeaseID = strings.TrimSpace(msg.resp.ProviderAuthLeaseID)
	m.status = "Provider credential lease is ready; executing the prepared remote change..."
	execReq, err := m.buildExecuteRequest()
	if err != nil {
		m.executing = false
		m.errText = safeUIErrorText(err)
		m.status = ""
		return m, nil
	}
	return m, m.executeCmd(execReq)
}

func (m gitRemoteMutationRouteModel) beginLoad() gitRemoteMutationRouteModel {
	m.loading = true
	m.errText = ""
	m.providerAuthLeaseID = ""
	m.loadSeq++
	return m
}

func (m gitRemoteMutationRouteModel) View(width, height int, focus focusArea) string {
	_ = width
	_ = height
	if m.loading {
		return renderStateCard(routeLoadStateLoading, "Git Remote Mutation", "Loading prepared git remote mutation state via typed broker get contract...")
	}
	if m.executing {
		return renderStateCard(routeLoadStateLoading, "Git Remote Mutation", "Executing prepared mutation through typed broker execute contract...")
	}
	if m.errText != "" {
		return renderStateCard(routeLoadStateError, "Git Remote Mutation", "Load failed: "+m.errText+" (press r to retry)")
	}
	if strings.TrimSpace(m.prepared.PreparedMutationID) == "" {
		return renderStateCard(routeLoadStateEmpty, "Git Remote Mutation", "No prepared mutation loaded yet. Press r to fetch prepared state.")
	}
	summary := m.prepared.DerivedSummary
	return compactLines(
		sectionTitle("Git Remote Mutation")+" "+focusBadge(focus),
		renderStateCardSpec(gitRemoteStateCard(m.prepared, m.providerAuthLeaseID)),
		fmt.Sprintf("Planned change: %s", gitRemotePreparedSummary(summary)),
		fmt.Sprintf("Target: %s", gitRemoteTargetSummary(summary)),
		fmt.Sprintf("Approval check: %s", gitRemoteApprovalSummary(m.prepared)),
		fmt.Sprintf("Execution access: %s", gitRemoteLeaseSummary(m.providerAuthLeaseID)),
		fmt.Sprintf("Execution: %s", gitRemoteExecutionSummary(m.prepared)),
		fmt.Sprintf("Next safe action: %s", gitRemoteNextSafeAction(m.prepared, m.providerAuthLeaseID)),
		"Safety: RuneCode keeps execution blocked until approval evidence and broker-issued execution access both match this prepared change.",
		m.status,
	)
}

func gitRemoteStateCard(prepared brokerapi.GitRemoteMutationPreparedState, leaseID string) stateCardSpec {
	state := routeLoadStateWaiting
	message := "The prepared remote change is ready for review."
	next := "Confirm the planned change, target, and approval before executing."
	if strings.TrimSpace(prepared.RequiredApprovalID) != "" && prepared.RequiredApprovalDecisionHash == nil {
		state = routeLoadStateApprovalRequired
		message = "Remote execution is waiting for approval."
		next = "Open Approvals before executing this remote change."
	}
	if strings.TrimSpace(leaseID) != "" && gitRemoteApprovalBindingComplete(prepared) {
		state = routeLoadStateReady
		message = "Execution access is ready for this remote change."
		next = "Press e only after confirming the planned change and approval state."
	}
	if strings.Contains(strings.ToLower(prepared.ExecutionState), "fail") {
		state = routeLoadStateDegraded
		message = "Remote change execution failed."
		next = "Review the failure reason before retrying."
	}
	return stateCardSpec{State: state, Title: "Guarded remote review", Message: message, Reason: "RuneCode only executes remote changes after the review, approval evidence, and execution access all agree.", NextAction: next, ShortcutCue: "r reload • e execute", EvidenceCue: "review, approval evidence, execution access"}
}

func gitRemotePreparedSummary(summary brokerapi.GitRemoteMutationDerivedSummary) string {
	if strings.TrimSpace(summary.CommitSubject) != "" {
		return summary.CommitSubject
	}
	if strings.TrimSpace(summary.PullRequestTitle) != "" {
		return summary.PullRequestTitle
	}
	return "review prepared repository change"
}

func gitRemoteTargetSummary(summary brokerapi.GitRemoteMutationDerivedSummary) string {
	repository := valueOrNA(summary.RepositoryIdentity)
	refs := joinCSV(summary.TargetRefs)
	if strings.TrimSpace(refs) == "" || refs == "n/a" {
		return repository
	}
	return fmt.Sprintf("%s on %s", repository, refs)
}

func gitRemoteApprovalSummary(prepared brokerapi.GitRemoteMutationPreparedState) string {
	if strings.TrimSpace(prepared.RequiredApprovalID) == "" {
		return "approval evidence is missing, so execution stays blocked"
	}
	if prepared.RequiredApprovalDecisionHash != nil {
		return "approval evidence is ready for execution review"
	}
	return "waiting for approval decision before execution"
}

func gitRemoteLeaseSummary(leaseID string) string {
	if strings.TrimSpace(leaseID) == "" {
		return "not ready yet; RuneCode will request execution access when you start the change"
	}
	return "ready for this prepared change"
}

func gitRemoteExecutionSummary(prepared brokerapi.GitRemoteMutationPreparedState) string {
	state := strings.TrimSpace(prepared.ExecutionState)
	if state == "" {
		state = strings.TrimSpace(prepared.LifecycleState)
	}
	switch strings.ToLower(state) {
	case "", "prepared", "not_started":
		return "not started; approval and execution checks still gate execution"
	case "completed", "complete", "succeeded", "success":
		return "completed"
	}
	if strings.Contains(strings.ToLower(state), "fail") {
		return "failed; review execution details before retrying"
	}
	return valueOrNA(state)
}

func gitRemoteNextSafeAction(prepared brokerapi.GitRemoteMutationPreparedState, leaseID string) string {
	state := strings.ToLower(strings.TrimSpace(prepared.ExecutionState))
	if strings.Contains(state, "fail") {
		return "Review the broker failure, then reload before retrying."
	}
	if state == "completed" || state == "complete" || state == "succeeded" || state == "success" {
		return "Review the refreshed state to confirm the remote change landed as expected."
	}
	if !gitRemoteApprovalBindingComplete(prepared) {
		return "Complete the bound approval review before executing this remote change."
	}
	if strings.TrimSpace(leaseID) != "" {
		return "Press e to execute this prepared remote change through the broker."
	}
	return "Press e to have RuneCode request a broker-bound credential lease and execute safely."
}

func gitRemoteApprovalBindingComplete(prepared brokerapi.GitRemoteMutationPreparedState) bool {
	return strings.TrimSpace(prepared.RequiredApprovalID) != "" && prepared.RequiredApprovalRequestHash != nil && prepared.RequiredApprovalDecisionHash != nil
}

func gitRemotePreparedBindingMatches(current brokerapi.GitRemoteMutationPreparedState, next brokerapi.GitRemoteMutationPreparedState) bool {
	if strings.TrimSpace(current.PreparedMutationID) != strings.TrimSpace(next.PreparedMutationID) {
		return false
	}
	if strings.TrimSpace(current.RequiredApprovalID) != strings.TrimSpace(next.RequiredApprovalID) {
		return false
	}
	return digestPtrString(current.RequiredApprovalRequestHash) == digestPtrString(next.RequiredApprovalRequestHash) && digestPtrString(current.RequiredApprovalDecisionHash) == digestPtrString(next.RequiredApprovalDecisionHash)
}

func digestPtrString(digest *trustpolicy.Digest) string {
	if digest == nil {
		return ""
	}
	identity, err := digest.Identity()
	if err != nil {
		return strings.TrimSpace(digest.HashAlg) + ":" + strings.TrimSpace(digest.Hash)
	}
	return identity
}

func (m gitRemoteMutationRouteModel) ShellSurface(ctx routeShellContext) routeSurface {
	mainWidth := routeRegionWidth(ctx.Regions.Main, ctx.Width)
	mainHeight := routeRegionHeight(ctx.Regions.Main, ctx.Height)
	status := strings.TrimSpace(m.status)
	if status == "" && strings.TrimSpace(m.errText) != "" {
		status = "Load failed: " + strings.TrimSpace(m.errText)
	}
	return routeSurface{
		Regions: routeSurfaceRegions{
			Main:   routeSurfaceRegion{Title: "Git remote mutation", Body: m.View(mainWidth, mainHeight, ctx.Focus)},
			Bottom: routeSurfaceRegion{Body: keyHint("Route keys: r reload prepared state, e execute prepared mutation")},
			Status: routeSurfaceRegion{Body: status},
		},
		Capabilities: routeSurfaceCapabilities{},
		Chrome:       routeSurfaceChrome{Breadcrumbs: []string{"Home", m.def.Label}},
	}
}

func (m gitRemoteMutationRouteModel) loadCmd(seq uint64) tea.Cmd {
	request := m.request
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.GitRemoteMutationGet(ctx, request)
		if err != nil {
			return gitRemoteMutationLoadedMsg{err: err, seq: seq}
		}
		return gitRemoteMutationLoadedMsg{resp: resp, seq: seq}
	}
}

func (m gitRemoteMutationRouteModel) executeCmd(req brokerapi.GitRemoteMutationExecuteRequest) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.GitRemoteMutationExecute(ctx, req)
		if err != nil {
			return gitRemoteMutationExecutedMsg{err: err}
		}
		return gitRemoteMutationExecutedMsg{resp: resp}
	}
}

func (m gitRemoteMutationRouteModel) issueExecuteLeaseCmd(req brokerapi.GitRemoteMutationIssueExecuteLeaseRequest) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := withLoadTimeout()
		defer cancel()
		resp, err := m.client.GitRemoteMutationIssueExecuteLease(ctx, req)
		if err != nil {
			return gitRemoteMutationLeaseIssuedMsg{err: err}
		}
		return gitRemoteMutationLeaseIssuedMsg{resp: resp}
	}
}

func (m gitRemoteMutationRouteModel) buildIssueExecuteLeaseRequest() (brokerapi.GitRemoteMutationIssueExecuteLeaseRequest, error) {
	if strings.TrimSpace(m.prepared.PreparedMutationID) == "" {
		return brokerapi.GitRemoteMutationIssueExecuteLeaseRequest{}, fmt.Errorf("prepared_mutation_id missing; reload prepared state before issuing execute lease")
	}
	if strings.TrimSpace(m.prepared.RequiredApprovalID) == "" || m.prepared.RequiredApprovalRequestHash == nil || m.prepared.RequiredApprovalDecisionHash == nil {
		return brokerapi.GitRemoteMutationIssueExecuteLeaseRequest{}, fmt.Errorf("required approval binding is incomplete in prepared state; execute remains fail-closed")
	}
	return brokerapi.GitRemoteMutationIssueExecuteLeaseRequest{PreparedMutationID: strings.TrimSpace(m.prepared.PreparedMutationID)}, nil
}

func (m gitRemoteMutationRouteModel) buildExecuteRequest() (brokerapi.GitRemoteMutationExecuteRequest, error) {
	if strings.TrimSpace(m.prepared.PreparedMutationID) == "" {
		return brokerapi.GitRemoteMutationExecuteRequest{}, fmt.Errorf("prepared_mutation_id missing; reload prepared state before execute")
	}
	if strings.TrimSpace(m.prepared.RequiredApprovalID) == "" || m.prepared.RequiredApprovalRequestHash == nil || m.prepared.RequiredApprovalDecisionHash == nil {
		return brokerapi.GitRemoteMutationExecuteRequest{}, fmt.Errorf("required approval binding is incomplete in prepared state; execute remains fail-closed")
	}
	if strings.TrimSpace(m.providerAuthLeaseID) == "" {
		return brokerapi.GitRemoteMutationExecuteRequest{}, fmt.Errorf("provider auth lease missing; issue a broker-bound execute lease before execute")
	}
	return brokerapi.GitRemoteMutationExecuteRequest{
		PreparedMutationID:   strings.TrimSpace(m.prepared.PreparedMutationID),
		ApprovalID:           strings.TrimSpace(m.prepared.RequiredApprovalID),
		ApprovalRequestHash:  *m.prepared.RequiredApprovalRequestHash,
		ApprovalDecisionHash: *m.prepared.RequiredApprovalDecisionHash,
		ProviderAuthLeaseID:  strings.TrimSpace(m.providerAuthLeaseID),
	}, nil
}

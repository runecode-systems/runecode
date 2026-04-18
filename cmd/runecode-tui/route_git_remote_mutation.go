package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/runecode-ai/runecode/internal/brokerapi"
	"github.com/runecode-ai/runecode/internal/trustpolicy"
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
	m.prepared = msg.resp.Prepared
	if m.status == "" {
		m.status = "Review derived summary + stable identities, then press e to execute via broker API."
	}
	return m, nil
}

func (m gitRemoteMutationRouteModel) handleExecuted(msg gitRemoteMutationExecutedMsg) (routeModel, tea.Cmd) {
	m.executing = false
	if msg.err != nil {
		m.errText = safeUIErrorText(msg.err)
		m.status = ""
		return m, nil
	}
	m.errText = ""
	m.prepared = msg.resp.Prepared
	m.status = fmt.Sprintf("Execute completed: execution_state=%s lifecycle_state=%s", valueOrNA(msg.resp.ExecutionState), valueOrNA(msg.resp.Prepared.LifecycleState))
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
	m.status = fmt.Sprintf("Issued execute lease %s; executing prepared mutation...", valueOrNA(m.providerAuthLeaseID))
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
	requestHash := digestIdentityOrNA(m.prepared.TypedRequestHash)
	actionHash := digestIdentityOrNA(m.prepared.ActionRequestHash)
	decisionHash := digestIdentityOrNA(m.prepared.PolicyDecisionHash)
	approvalReqHash := optionalDigestIdentityOrNA(m.prepared.RequiredApprovalRequestHash)
	approvalDecisionHash := optionalDigestIdentityOrNA(m.prepared.RequiredApprovalDecisionHash)
	patches := make([]string, 0, len(summary.ReferencedPatchArtifactHashes))
	for _, d := range summary.ReferencedPatchArtifactHashes {
		patches = append(patches, digestIdentityOrNA(d))
	}
	if len(patches) == 0 {
		patches = append(patches, "n/a")
	}
	return compactLines(
		sectionTitle("Git Remote Mutation")+" "+focusBadge(focus),
		"Review-centric broker flow over canonical prepare/get/execute contracts:",
		fmt.Sprintf("Prepared mutation: %s", valueOrNA(m.prepared.PreparedMutationID)),
		fmt.Sprintf("Lifecycle: lifecycle_state=%s reason=%s execution_state=%s execution_reason=%s", valueOrNA(m.prepared.LifecycleState), valueOrNA(m.prepared.LifecycleReasonCode), valueOrNA(m.prepared.ExecutionState), valueOrNA(m.prepared.ExecutionReasonCode)),
		fmt.Sprintf("Scope: run=%s provider=%s destination_ref=%s request_kind=%s", valueOrNA(m.prepared.RunID), valueOrNA(m.prepared.Provider), valueOrNA(m.prepared.DestinationRef), valueOrNA(m.prepared.RequestKind)),
		fmt.Sprintf("Stable identities: typed_request_hash=%s", requestHash),
		fmt.Sprintf("Bindings: action_request_hash=%s policy_decision_hash=%s", actionHash, decisionHash),
		fmt.Sprintf("Approval binding: approval_id=%s approval_request_hash=%s approval_decision_hash=%s", valueOrNA(m.prepared.RequiredApprovalID), approvalReqHash, approvalDecisionHash),
		fmt.Sprintf("Execute credential lease: provider_auth_lease_id=%s", valueOrNA(m.providerAuthLeaseID)),
		fmt.Sprintf("Derived summary: repository=%s target_refs=%s", valueOrNA(summary.RepositoryIdentity), joinCSV(summary.TargetRefs)),
		fmt.Sprintf("Derived result: expected_result_tree_hash=%s patch_artifacts=%s", digestIdentityOrNA(summary.ExpectedResultTreeHash), joinCSV(patches)),
		fmt.Sprintf("Derived intent: commit_subject=%s pr_title=%s pr_base=%s pr_head=%s", valueOrNA(summary.CommitSubject), valueOrNA(summary.PullRequestTitle), valueOrNA(summary.PullRequestBaseRef), valueOrNA(summary.PullRequestHeadRef)),
		"Fail-closed: execute requires required approval bindings and a broker-issued provider credential lease bound to this prepared mutation.",
		m.status,
		keyHint("Route keys: r reload prepared state, e execute prepared mutation"),
	)
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

func digestIdentityOrNA(d trustpolicy.Digest) string {
	identity, err := d.Identity()
	if err != nil {
		return "n/a"
	}
	return identity
}

func optionalDigestIdentityOrNA(d *trustpolicy.Digest) string {
	if d == nil {
		return "n/a"
	}
	identity, err := d.Identity()
	if err != nil {
		return "n/a"
	}
	return identity
}

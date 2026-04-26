------------------------------ MODULE SecurityKernelV0 ------------------------------
EXTENDS Naturals, FiniteSets, TLC

\* Bounded kernel model for CHG-2026-015 (workflow security-kernel v0).
\* This module intentionally models token/hash identities as opaque deterministic values.

CONSTANTS
  Runs,
  Plans,
  Stages,
  Actions,
  Approvals,
  Gates,
  GateAttempts,
  HashTokens,
  ArtifactDigests,
  PolicyHashes,
  PolicyInputHashes,
  EvidenceRefs,
  ProjectDigests,
  ExecutionScopes,
  MaxRevision,
  MaxSteps

ApprovalLifecycle ==
  {"pending", "approved", "denied", "expired", "cancelled", "superseded", "consumed"}

ApprovalTerminalStates == {"consumed", "denied", "expired", "cancelled", "superseded"}

ExecutionLifecycle ==
  {"queued", "planning", "running", "waiting", "blocked", "failed", "completed"}

WaitKinds == {"none", "operator_input", "approval", "external_dependency", "project_blocked"}

WaitStates ==
  {"none", "waiting_operator_input", "waiting_approval", "waiting_external_dependency",
   "waiting_project_blocked"}

PublicRunLifecycle ==
  {"pending", "starting", "active", "blocked", "recovering", "completed", "failed", "cancelled"}

TerminalRunLifecycle == {"completed", "failed", "cancelled"}

ApprovalBindingKinds == {"exact_action", "stage_sign_off"}

ActionKinds == {"generic", "stage_sign_off", "gate_override"}

GateLifecycle == {"planned", "running", "passed", "failed", "overridden", "superseded"}

GateAttemptOutcomes == {"not_run", "passed", "failed", "superseded"}

AuditObligationStates == {"not_required", "required_missing", "required_recorded"}

AuditTransitionKinds == {
  "approval_request_created",
  "approval_decision_accepted",
  "approval_consumed",
  "stage_signoff_consumed",
  "gate_result_accepted",
  "gate_override_continued",
  "run_terminal_transition",
  "plan_supersession"
}

NoEvidence == "no_evidence"

NoPendingApproval == "no_pending_approval"

ASSUME
  /\ Runs # {}
  /\ Plans # {}
  /\ Stages # {}
  /\ Actions # {}
  /\ Approvals # {}
  /\ Gates # {}
  /\ GateAttempts # {}
  /\ HashTokens # {}
  /\ ArtifactDigests # {}
  /\ PolicyHashes # {}
  /\ PolicyInputHashes # {}
  /\ EvidenceRefs # {}
  /\ ProjectDigests # {}
  /\ ExecutionScopes # {}
  /\ MaxRevision \in Nat
  /\ MaxSteps \in Nat
  /\ NoEvidence \notin EvidenceRefs
  /\ NoPendingApproval \notin Approvals
  /\ Cardinality(Runs) >= 2
  /\ Cardinality(Plans) >= 2
  /\ Cardinality(Stages) >= 1
  /\ Cardinality(Actions) >= 3
  /\ Cardinality(Approvals) >= 4
  /\ Cardinality(GateAttempts) >= 1
  /\ Cardinality(HashTokens) >= 2
  /\ Cardinality(ArtifactDigests) >= 1
  /\ Cardinality(PolicyHashes) >= 1
  /\ Cardinality(PolicyInputHashes) >= 1
  /\ Cardinality(EvidenceRefs) >= 2
  /\ Cardinality(ProjectDigests) >= 2
  /\ Cardinality(ExecutionScopes) >= 3

PrimaryRun == CHOOSE r \in Runs : TRUE

SecondaryRun == CHOOSE r \in (Runs \ {PrimaryRun}) : TRUE

PrimaryPlan == CHOOSE p \in Plans : TRUE

SupersedingPlan == CHOOSE p \in (Plans \ {PrimaryPlan}) : TRUE

PrimaryStage == CHOOSE s \in Stages : TRUE

ExactAction == CHOOSE a \in Actions : TRUE

StageAction == CHOOSE a \in (Actions \ {ExactAction}) : TRUE

OverrideAction == CHOOSE a \in (Actions \ {ExactAction, StageAction}) : TRUE

ExactApproval == CHOOSE a \in Approvals : TRUE

StageApproval == CHOOSE a \in (Approvals \ {ExactApproval}) : TRUE

OverrideApproval == CHOOSE a \in (Approvals \ {ExactApproval, StageApproval}) : TRUE

DeniedApproval == CHOOSE a \in (Approvals \ {ExactApproval, StageApproval, OverrideApproval}) : TRUE

PrimaryGateAttempt == CHOOSE ga \in GateAttempts : TRUE

PrimaryEvidence == CHOOSE e \in EvidenceRefs : TRUE

SecondaryEvidence == CHOOSE e \in (EvidenceRefs \ {PrimaryEvidence}) : TRUE

PrimaryHash == CHOOSE h \in HashTokens : TRUE

SecondaryHash == CHOOSE h \in (HashTokens \ {PrimaryHash}) : TRUE

PrimaryArtifactDigest == CHOOSE d \in ArtifactDigests : TRUE

PrimaryPolicyHash == CHOOSE h \in PolicyHashes : TRUE

PrimaryPolicyInputHash == CHOOSE h \in PolicyInputHashes : TRUE

PrimaryProjectDigest == CHOOSE d \in ProjectDigests : TRUE

DriftedProjectDigest == CHOOSE d \in (ProjectDigests \ {PrimaryProjectDigest}) : TRUE

PrimaryScope == CHOOSE s \in ExecutionScopes : TRUE

DownstreamScope == CHOOSE s \in (ExecutionScopes \ {PrimaryScope}) : TRUE

IndependentScope == CHOOSE s \in (ExecutionScopes \ {PrimaryScope, DownstreamScope}) : TRUE

MirrorDriftApprovalState == "denied"

MirrorDriftRunLifecycle == "blocked"

WaitStateFor(kind) ==
  CASE kind = "operator_input" -> "waiting_operator_input"
    [] kind = "approval" -> "waiting_approval"
    [] kind = "external_dependency" -> "waiting_external_dependency"
    [] kind = "project_blocked" -> "waiting_project_blocked"
    [] OTHER -> "none"

ScopeApproval(scope) ==
  CASE scope = PrimaryScope -> ExactApproval
    [] scope = DownstreamScope -> StageApproval
    [] OTHER -> OverrideApproval

VARIABLES
  \* Broker-authoritative run / plan / coordination state.
  runPlan,
  runLifecycle,
  noEligibleWork,
  partialBlocked,
  stageSummaryHash,
  summaryRevision,

  \* Canonical action request metadata (opaque hashes/tokens are identity bindings).
  actionReqRun,
  actionReqPlan,
  actionReqStage,
  actionKind,
  actionReqHash,
  actionManifestHash,
  actionPolicyInputHash,
  actionArtifactDigest,
  actionOverrideAttempt,
  actionOverriddenFailedRef,

  \* Canonical approval records (broker authority).
  approvalRun,
  approvalPlan,
  approvalStage,
  approvalBindingKind,
  approvalBoundAction,
  approvalBoundStageHash,
  approvalManifestHash,
  approvalPolicyInputHash,
  approvalState,
  approvalConsumeCount,
  consumedByAction,

  \* Gate attempt and evidence references (broker materialized evidence identity).
  gateAttemptRun,
  gateAttemptPlan,
  gateAttemptGate,
  gateState,
  gateOutcome,
  gateFailedResultRef,
  gateWasFailedBeforeOverride,
  brokerGateEvidenceRef,

  \* Runner durable mirrors (advisory only).
  runnerApprovalMirror,
  runnerRunLifecycleMirror,
  runnerPlanMirror,

  \* Effective/public projection state (must remain broker-derived).
  effectiveApprovalState,
  effectiveRunLifecycle,
  effectiveRunPlan,
  effectiveGateEvidenceRef,

  \* Minimal audit-obligation states for closed v0 transition matrix.
  approvalRequestAuditState,
  approvalDecisionAuditState,
  approvalConsumptionAuditState,
  stageSignoffConsumptionAuditState,
  gateResultAuditState,
  gateOverrideAuditState,
  runTerminalAuditState,
  planSupersessionAuditState,

  \* Run plan supersession fact.
  planSuperseded,

  \* Broker-owned execution scope state and wait vocabulary.
  executionRun,
  executionState,
  executionWaitKind,
  executionWaitState,
  executionDependsOn,
  executionProjectSensitive,
  executionBoundProjectDigest,
  executionCurrentProjectDigest,
  executionPendingApproval,

  \* Bounded trace length for deterministic TLC runs.
  stepCount

executionVars == <<
  executionRun, executionState, executionWaitKind, executionWaitState, executionDependsOn,
  executionProjectSensitive, executionBoundProjectDigest, executionCurrentProjectDigest,
  executionPendingApproval
>>

RunHasScopedWait(r) ==
  \E es \in ExecutionScopes :
    /\ executionRun[es] = r
    /\ executionState[es] \in {"waiting", "blocked"}

vars == <<
  runPlan, runLifecycle, noEligibleWork, partialBlocked, stageSummaryHash, summaryRevision,
  actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
  actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
  approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
  approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalState,
  approvalConsumeCount, consumedByAction,
  gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
  gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
  runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
  effectiveApprovalState, effectiveRunLifecycle, effectiveRunPlan, effectiveGateEvidenceRef,
  approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
  stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
  runTerminalAuditState, planSupersessionAuditState, planSuperseded,
  executionRun, executionState, executionWaitKind, executionWaitState, executionDependsOn,
  executionProjectSensitive, executionBoundProjectDigest, executionCurrentProjectDigest,
  executionPendingApproval, stepCount
>>

Init ==
  LET
    initRunPlan == [r \in Runs |-> IF r = PrimaryRun THEN PrimaryPlan ELSE SupersedingPlan]
    initActionReqRun ==
      [a \in Actions |->
        CASE a = ExactAction -> PrimaryRun
          [] a = StageAction -> PrimaryRun
          [] OTHER -> PrimaryRun]
    initActionReqPlan == [a \in Actions |-> initRunPlan[initActionReqRun[a]]]
    initActionReqStage == [a \in Actions |-> PrimaryStage]
    initStageSummaryHash == [r \in Runs |-> [p \in Plans |-> [s \in Stages |-> PrimaryHash]]]
    initApprovalRun ==
      [a \in Approvals |->
        CASE a = ExactApproval -> PrimaryRun
          [] a = StageApproval -> PrimaryRun
          [] a = OverrideApproval -> PrimaryRun
          [] OTHER -> SecondaryRun]
    initApprovalPlan == [a \in Approvals |-> initRunPlan[initApprovalRun[a]]]
    initApprovalStage == [a \in Approvals |-> PrimaryStage]
    initApprovalBoundStageHash ==
      [a \in Approvals |-> IF a = StageApproval THEN initStageSummaryHash[PrimaryRun][PrimaryPlan][PrimaryStage] ELSE PrimaryHash]
  IN
  /\ runPlan = initRunPlan
  /\ runLifecycle = [r \in Runs |-> "pending"]
  /\ noEligibleWork = [r \in Runs |-> FALSE]
  /\ partialBlocked = [r \in Runs |-> FALSE]
  /\ stageSummaryHash = initStageSummaryHash
  /\ summaryRevision = [r \in Runs |-> [p \in Plans |-> [s \in Stages |-> 0]]]

  /\ actionReqRun = initActionReqRun
  /\ actionReqPlan = initActionReqPlan
  /\ actionReqStage = initActionReqStage
  /\ actionKind =
       [a \in Actions |->
         CASE a = ExactAction -> "generic"
           [] a = StageAction -> "stage_sign_off"
           [] OTHER -> "gate_override"]
  /\ actionReqHash = [a \in Actions |-> PrimaryHash]
  /\ actionManifestHash = [a \in Actions |-> PrimaryPolicyHash]
  /\ actionPolicyInputHash = [a \in Actions |-> PrimaryPolicyInputHash]
  /\ actionArtifactDigest = [a \in Actions |-> PrimaryArtifactDigest]
  /\ actionOverrideAttempt = [a \in Actions |-> PrimaryGateAttempt]
  /\ actionOverriddenFailedRef = [a \in Actions |-> NoEvidence]

  /\ approvalRun = initApprovalRun
  /\ approvalPlan = initApprovalPlan
  /\ approvalStage = initApprovalStage
  /\ approvalBindingKind =
       [a \in Approvals |-> IF a = StageApproval THEN "stage_sign_off" ELSE "exact_action"]
  /\ approvalBoundAction =
       [a \in Approvals |->
          CASE a = ExactApproval -> ExactAction
            [] a = StageApproval -> StageAction
            [] a = OverrideApproval -> OverrideAction
            [] OTHER -> ExactAction]
  /\ approvalBoundStageHash = initApprovalBoundStageHash
  /\ approvalManifestHash = [a \in Approvals |-> PrimaryPolicyHash]
  /\ approvalPolicyInputHash = [a \in Approvals |-> PrimaryPolicyInputHash]
  /\ approvalState = [a \in Approvals |-> "pending"]
  /\ approvalConsumeCount = [a \in Approvals |-> 0]
  /\ consumedByAction = [a \in Approvals |-> {}]

  /\ gateAttemptRun = [ga \in GateAttempts |-> PrimaryRun]
  /\ gateAttemptPlan = [ga \in GateAttempts |-> PrimaryPlan]
  /\ gateAttemptGate = [ga \in GateAttempts |-> CHOOSE g \in Gates : TRUE]
  /\ gateState = [ga \in GateAttempts |-> "planned"]
  /\ gateOutcome = [ga \in GateAttempts |-> "not_run"]
  /\ gateFailedResultRef = [ga \in GateAttempts |-> NoEvidence]
  /\ gateWasFailedBeforeOverride = [ga \in GateAttempts |-> FALSE]
  /\ brokerGateEvidenceRef = [ga \in GateAttempts |-> NoEvidence]

  /\ runnerApprovalMirror = approvalState
  /\ runnerRunLifecycleMirror = runLifecycle
  /\ runnerPlanMirror = runPlan

  /\ effectiveApprovalState = approvalState
  /\ effectiveRunLifecycle = runLifecycle
  /\ effectiveRunPlan = runPlan
  /\ effectiveGateEvidenceRef = brokerGateEvidenceRef

  /\ executionRun = [es \in ExecutionScopes |-> PrimaryRun]
  /\ executionState =
       [es \in ExecutionScopes |->
         CASE es = PrimaryScope -> "running"
           [] es = DownstreamScope -> "queued"
           [] OTHER -> "running"]
  /\ executionWaitKind = [es \in ExecutionScopes |-> "none"]
  /\ executionWaitState = [es \in ExecutionScopes |-> "none"]
  /\ executionDependsOn =
       [es \in ExecutionScopes |->
         CASE es = PrimaryScope -> {}
           [] es = DownstreamScope -> {PrimaryScope}
           [] OTHER -> {}]
  /\ executionProjectSensitive =
       [es \in ExecutionScopes |-> es # IndependentScope]
  /\ executionBoundProjectDigest = [es \in ExecutionScopes |-> PrimaryProjectDigest]
  /\ executionCurrentProjectDigest = [es \in ExecutionScopes |-> PrimaryProjectDigest]
  /\ executionPendingApproval = [es \in ExecutionScopes |-> NoPendingApproval]

  /\ approvalRequestAuditState = [a \in Approvals |-> "required_recorded"]
  /\ approvalDecisionAuditState = [a \in Approvals |-> "not_required"]
  /\ approvalConsumptionAuditState = [a \in Approvals |-> "not_required"]
  /\ stageSignoffConsumptionAuditState = [a \in Approvals |-> "not_required"]
  /\ gateResultAuditState = [ga \in GateAttempts |-> "not_required"]
  /\ gateOverrideAuditState = [a \in Approvals |-> "not_required"]
  /\ runTerminalAuditState = [r \in Runs |-> "not_required"]
  /\ planSupersessionAuditState = [r \in Runs |-> "not_required"]
  /\ planSuperseded = [r \in Runs |-> FALSE]
  /\ stepCount = 0

AcceptApprovalDecision ==
  \E a \in {ExactApproval, StageApproval, OverrideApproval, DeniedApproval}, decision \in {"approved", "denied"} :
    /\ approvalState[a] = "pending"
    /\ (a = DeniedApproval => decision = "denied")
    /\ (a \in {ExactApproval, StageApproval} => decision = "approved")
    /\ approvalState' = [approvalState EXCEPT ![a] = decision]
    /\ effectiveApprovalState' = [effectiveApprovalState EXCEPT ![a] = decision]
    /\ approvalDecisionAuditState' = [approvalDecisionAuditState EXCEPT ![a] = "required_recorded"]
    /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runPlan, runLifecycle, noEligibleWork, partialBlocked, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalConsumeCount,
         consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
         effectiveRunLifecycle, effectiveRunPlan, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalConsumptionAuditState,
         stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
         runTerminalAuditState, planSupersessionAuditState, planSuperseded
         >>
    /\ UNCHANGED executionVars

ExpireOrCancelApproval ==
  \E a \in {DeniedApproval}, terminal \in {"expired", "cancelled"} :
    /\ approvalState[DeniedApproval] = "pending"
    /\ approvalState' = [approvalState EXCEPT ![a] = terminal]
    /\ effectiveApprovalState' = [effectiveApprovalState EXCEPT ![a] = terminal]
    /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runPlan, runLifecycle, noEligibleWork, partialBlocked, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalConsumeCount,
         consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
         effectiveRunLifecycle, effectiveRunPlan, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
         stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
         runTerminalAuditState, planSupersessionAuditState, planSuperseded
         >>
    /\ UNCHANGED executionVars

ConsumeApproval ==
  \E a \in {ExactApproval, StageApproval, OverrideApproval}, act \in {ExactAction, StageAction, OverrideAction} :
    LET
      isStageSignoff == approvalBindingKind[a] = "stage_sign_off"
      isGateOverride == actionKind[act] = "gate_override"
      targetAttempt == actionOverrideAttempt[act]
      newGateState ==
        IF isGateOverride
          THEN [gateState EXCEPT ![targetAttempt] = "overridden"]
          ELSE gateState
      newGateOverrideAudit ==
        IF isGateOverride
          THEN [gateOverrideAuditState EXCEPT ![a] = "required_recorded"]
          ELSE gateOverrideAuditState
      newOverriddenFailedRef ==
        IF isGateOverride
          THEN [actionOverriddenFailedRef EXCEPT ![act] = gateFailedResultRef[targetAttempt]]
          ELSE actionOverriddenFailedRef
    IN
    /\ approvalState[a] = "approved"
    /\ approvalConsumeCount[a] = 0
    /\ Cardinality(consumedByAction[a]) = 0
    /\ approvalRun[a] = actionReqRun[act]
    /\ approvalPlan[a] = actionReqPlan[act]
    /\ approvalManifestHash[a] = actionManifestHash[act]
    /\ approvalPolicyInputHash[a] = actionPolicyInputHash[act]
    /\ IF approvalBindingKind[a] = "exact_action"
         THEN approvalBoundAction[a] = act
         ELSE /\ actionKind[act] = "stage_sign_off"
              /\ approvalStage[a] = actionReqStage[act]
              /\ approvalBoundStageHash[a] =
                    stageSummaryHash[approvalRun[a]][approvalPlan[a]][approvalStage[a]]
    /\ IF isGateOverride THEN gateState[targetAttempt] = "failed" ELSE TRUE
    /\ approvalState' = [approvalState EXCEPT ![a] = "consumed"]
    /\ effectiveApprovalState' = [effectiveApprovalState EXCEPT ![a] = "consumed"]
    /\ approvalConsumeCount' = [approvalConsumeCount EXCEPT ![a] = @ + 1]
    /\ consumedByAction' = [consumedByAction EXCEPT ![a] = {act}]
    /\ stepCount' = stepCount + 1
    /\ approvalConsumptionAuditState' = [approvalConsumptionAuditState EXCEPT ![a] = "required_recorded"]
    /\ stageSignoffConsumptionAuditState' =
         IF isStageSignoff
           THEN [stageSignoffConsumptionAuditState EXCEPT ![a] = "required_recorded"]
           ELSE stageSignoffConsumptionAuditState
    /\ actionOverriddenFailedRef' = newOverriddenFailedRef
    /\ gateState' = newGateState
    /\ gateOverrideAuditState' = newGateOverrideAudit
    /\ UNCHANGED <<
         runPlan, runLifecycle, noEligibleWork, partialBlocked, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
         effectiveRunLifecycle, effectiveRunPlan, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalDecisionAuditState,
          gateResultAuditState, runTerminalAuditState, planSupersessionAuditState, planSuperseded
         >>
    /\ UNCHANGED executionVars

UpdateStageSummary ==
  \E newHash \in (HashTokens \ {stageSummaryHash[PrimaryRun][PrimaryPlan][PrimaryStage]}) :
    LET
      r == PrimaryRun
      p == runPlan[r]
      s == PrimaryStage
      oldHash == stageSummaryHash[r][p][s]
      supersededApprovals ==
        {a \in Approvals :
           /\ approvalBindingKind[a] = "stage_sign_off"
           /\ approvalRun[a] = r
           /\ approvalPlan[a] = p
           /\ approvalStage[a] = s
           /\ approvalState[a] \in {"pending", "approved"}
           /\ approvalBoundStageHash[a] # newHash}
      nextApprovalState ==
        [a \in Approvals |-> IF a \in supersededApprovals THEN "superseded" ELSE approvalState[a]]
    IN
    /\ newHash # oldHash
    /\ summaryRevision[r][p][s] < MaxRevision
    /\ stageSummaryHash' = [stageSummaryHash EXCEPT ![r][p][s] = newHash]
    /\ summaryRevision' = [summaryRevision EXCEPT ![r][p][s] = @ + 1]
    /\ approvalState' = nextApprovalState
    /\ effectiveApprovalState' = nextApprovalState
    /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runPlan, runLifecycle, noEligibleWork, partialBlocked,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalConsumeCount,
         consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
         effectiveRunLifecycle, effectiveRunPlan, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
         stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
         runTerminalAuditState, planSupersessionAuditState, planSuperseded
         >>
    /\ UNCHANGED executionVars

ReportGateAttemptResult ==
  \E outcome \in {"passed", "failed"}, ref \in EvidenceRefs :
    LET
      ga == PrimaryGateAttempt
      nextGateState == [gateState EXCEPT ![ga] = IF outcome = "passed" THEN "passed" ELSE "failed"]
      nextGateOutcome == [gateOutcome EXCEPT ![ga] = outcome]
      nextFailedRef == [gateFailedResultRef EXCEPT ![ga] = IF outcome = "failed" THEN ref ELSE NoEvidence]
      nextWasFailed == [gateWasFailedBeforeOverride EXCEPT ![ga] = IF outcome = "failed" THEN TRUE ELSE @]
      nextEvidence == [brokerGateEvidenceRef EXCEPT ![ga] = ref]
    IN
    /\ gateState[ga] \in {"planned", "running", "failed"}
    /\ gateState' = nextGateState
    /\ gateOutcome' = nextGateOutcome
    /\ gateFailedResultRef' = nextFailedRef
    /\ gateWasFailedBeforeOverride' = nextWasFailed
    /\ brokerGateEvidenceRef' = nextEvidence
    /\ effectiveGateEvidenceRef' = nextEvidence
    /\ gateResultAuditState' = [gateResultAuditState EXCEPT ![ga] = "required_recorded"]
    /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runPlan, runLifecycle, noEligibleWork, partialBlocked, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalState,
         approvalConsumeCount, consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate,
         runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
         effectiveApprovalState, effectiveRunLifecycle, effectiveRunPlan,
         approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
         stageSignoffConsumptionAuditState, gateOverrideAuditState,
         runTerminalAuditState, planSupersessionAuditState, planSuperseded
         >>
    /\ UNCHANGED executionVars

SupersedePlan ==
  LET
    r == PrimaryRun
    newPlan == SupersedingPlan
  IN
    LET
      oldPlan == runPlan[r]
      supersededApprovals ==
        {a \in Approvals :
          /\ approvalRun[a] = r
          /\ approvalPlan[a] = oldPlan
          /\ approvalState[a] \in {"pending", "approved"}}
      nextApprovalState ==
        [a \in Approvals |-> IF a \in supersededApprovals THEN "superseded" ELSE approvalState[a]]
      nextRunPlan == [runPlan EXCEPT ![r] = newPlan]
    IN
    /\ newPlan # oldPlan
    /\ runPlan' = nextRunPlan
    /\ effectiveRunPlan' = nextRunPlan
    /\ planSuperseded' = [planSuperseded EXCEPT ![r] = TRUE]
    /\ planSupersessionAuditState' = [planSupersessionAuditState EXCEPT ![r] = "required_recorded"]
    /\ approvalState' = nextApprovalState
    /\ effectiveApprovalState' = nextApprovalState
    /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runLifecycle, noEligibleWork, partialBlocked, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalConsumeCount,
         consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
         effectiveRunLifecycle, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
         stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
         runTerminalAuditState
         >>
    /\ UNCHANGED executionVars

UpdateRunCoordination ==
  \E pblocked \in BOOLEAN, noWork \in BOOLEAN,
     nextState \in {"active", "blocked", "recovering"} :
    LET
      r == PrimaryRun
      nextLifecycle == [runLifecycle EXCEPT ![r] = nextState]
      nextPartialBlocked == pblocked \/ RunHasScopedWait(r)
    IN
    /\ nextState = "blocked" => noWork
    /\ nextPartialBlocked => nextState \in {"starting", "active", "recovering", "blocked"}
    /\ partialBlocked' = [partialBlocked EXCEPT ![r] = nextPartialBlocked]
    /\ noEligibleWork' = [noEligibleWork EXCEPT ![r] = noWork]
    /\ runLifecycle' = nextLifecycle
    /\ effectiveRunLifecycle' = [effectiveRunLifecycle EXCEPT ![r] = nextState]
    /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runPlan, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalState,
         approvalConsumeCount, consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
         effectiveApprovalState, effectiveRunPlan, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
         stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
         runTerminalAuditState, planSupersessionAuditState, planSuperseded
         >>
    /\ UNCHANGED executionVars

SetRunTerminal ==
  \E r \in {PrimaryRun}, terminal \in TerminalRunLifecycle :
    /\ partialBlocked' = [partialBlocked EXCEPT ![r] = FALSE]
    /\ noEligibleWork' = [noEligibleWork EXCEPT ![r] = FALSE]
    /\ runLifecycle' = [runLifecycle EXCEPT ![r] = terminal]
    /\ effectiveRunLifecycle' = [effectiveRunLifecycle EXCEPT ![r] = terminal]
    /\ runTerminalAuditState' = [runTerminalAuditState EXCEPT ![r] = "required_recorded"]
    /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runPlan, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalState,
         approvalConsumeCount, consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
         effectiveApprovalState, effectiveRunPlan, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
         stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
           planSupersessionAuditState, planSuperseded
         >>
    /\ UNCHANGED executionVars

EnterExecutionWait ==
  \E es \in {PrimaryScope, IndependentScope}, kind \in {"operator_input", "approval", "external_dependency"} :
    LET
      pending == IF kind = "approval" THEN ScopeApproval(es) ELSE NoPendingApproval
    IN
    /\ runLifecycle[executionRun[es]] \in {"starting", "active", "recovering", "blocked"}
    /\ executionState[es] \in {"queued", "planning", "running"}
    /\ executionWaitKind[es] = "none"
    /\ executionWaitState[es] = "none"
    /\ IF kind = "approval"
         THEN /\ pending # NoPendingApproval
              /\ approvalState[pending] = "pending"
              /\ approvalRun[pending] = executionRun[es]
         ELSE pending = NoPendingApproval
    /\ executionState' = [executionState EXCEPT ![es] = "waiting"]
    /\ executionWaitKind' = [executionWaitKind EXCEPT ![es] = kind]
    /\ executionWaitState' = [executionWaitState EXCEPT ![es] = WaitStateFor(kind)]
    /\ executionPendingApproval' = [executionPendingApproval EXCEPT ![es] = pending]
    /\ partialBlocked' = [partialBlocked EXCEPT ![executionRun[es]] = TRUE]
    /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runPlan, runLifecycle, noEligibleWork, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalState,
         approvalConsumeCount, consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
         effectiveApprovalState, effectiveRunLifecycle, effectiveRunPlan, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
         stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
         runTerminalAuditState, planSupersessionAuditState, planSuperseded,
         executionRun, executionDependsOn, executionProjectSensitive,
         executionBoundProjectDigest, executionCurrentProjectDigest
        >>

BlockDependentExecution ==
  \E dep \in executionDependsOn[DownstreamScope] :
    /\ runLifecycle[executionRun[DownstreamScope]] \in {"starting", "active", "recovering", "blocked"}
    /\ executionState[DownstreamScope] \in {"queued", "planning", "running"}
    /\ executionState[dep] = "waiting"
    /\ executionState' = [executionState EXCEPT ![DownstreamScope] = "blocked"]
    /\ executionWaitKind' = [executionWaitKind EXCEPT ![DownstreamScope] = executionWaitKind[dep]]
    /\ executionWaitState' = [executionWaitState EXCEPT ![DownstreamScope] = executionWaitState[dep]]
    /\ executionPendingApproval' =
         [executionPendingApproval EXCEPT ![DownstreamScope] = executionPendingApproval[dep]]
    /\ partialBlocked' = [partialBlocked EXCEPT ![executionRun[DownstreamScope]] = TRUE]
    /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runPlan, runLifecycle, noEligibleWork, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalState,
         approvalConsumeCount, consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
         effectiveApprovalState, effectiveRunLifecycle, effectiveRunPlan, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
         stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
         runTerminalAuditState, planSupersessionAuditState, planSuperseded,
         executionRun, executionDependsOn, executionProjectSensitive,
         executionBoundProjectDigest, executionCurrentProjectDigest
        >>

ResolveExecutionWait ==
  \E es \in ExecutionScopes :
    LET
      resumeDownstream ==
        /\ es # DownstreamScope
        /\ executionState[DownstreamScope] = "blocked"
        /\ executionWaitKind[DownstreamScope] # "project_blocked"
        /\ es \in executionDependsOn[DownstreamScope]
        /\ \A dep \in (executionDependsOn[DownstreamScope] \ {es}) :
             executionState[dep] \notin {"waiting", "blocked"}
      nextExecutionState ==
        [executionState EXCEPT
          ![es] = "running",
          ![DownstreamScope] = IF resumeDownstream THEN "running" ELSE @]
      nextExecutionWaitKind ==
        [executionWaitKind EXCEPT
          ![es] = "none",
          ![DownstreamScope] = IF resumeDownstream THEN "none" ELSE @]
      nextExecutionWaitState ==
        [executionWaitState EXCEPT
          ![es] = "none",
          ![DownstreamScope] = IF resumeDownstream THEN "none" ELSE @]
      nextExecutionPendingApproval ==
        [executionPendingApproval EXCEPT
          ![es] = NoPendingApproval,
          ![DownstreamScope] = IF resumeDownstream THEN NoPendingApproval ELSE @]
    IN
    /\ executionState[es] = "waiting"
    /\ IF executionWaitKind[es] = "approval"
         THEN /\ executionPendingApproval[es] # NoPendingApproval
              /\ effectiveApprovalState[executionPendingApproval[es]] = "consumed"
         ELSE TRUE
    /\ executionState' = nextExecutionState
    /\ executionWaitKind' = nextExecutionWaitKind
    /\ executionWaitState' = nextExecutionWaitState
    /\ executionPendingApproval' = nextExecutionPendingApproval
    /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runPlan, runLifecycle, noEligibleWork, partialBlocked, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalState,
         approvalConsumeCount, consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
         effectiveApprovalState, effectiveRunLifecycle, effectiveRunPlan, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
         stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
         runTerminalAuditState, planSupersessionAuditState, planSuperseded,
         executionRun, executionDependsOn, executionProjectSensitive,
         executionBoundProjectDigest, executionCurrentProjectDigest
        >>

ResumeDependentExecution ==
  /\ executionState[DownstreamScope] = "blocked"
  /\ executionWaitKind[DownstreamScope] # "project_blocked"
  /\ \A dep \in executionDependsOn[DownstreamScope] :
       executionState[dep] \notin {"waiting", "blocked"}
  /\ executionState' = [executionState EXCEPT ![DownstreamScope] = "running"]
  /\ executionWaitKind' = [executionWaitKind EXCEPT ![DownstreamScope] = "none"]
  /\ executionWaitState' = [executionWaitState EXCEPT ![DownstreamScope] = "none"]
  /\ executionPendingApproval' = [executionPendingApproval EXCEPT ![DownstreamScope] = NoPendingApproval]
  /\ stepCount' = stepCount + 1
  /\ UNCHANGED <<
       runPlan, runLifecycle, noEligibleWork, partialBlocked, stageSummaryHash, summaryRevision,
       actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
       actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
       approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
       approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalState,
       approvalConsumeCount, consumedByAction,
       gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
       gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
       runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
       effectiveApprovalState, effectiveRunLifecycle, effectiveRunPlan, effectiveGateEvidenceRef,
       approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
       stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
       runTerminalAuditState, planSupersessionAuditState, planSuperseded,
       executionRun, executionDependsOn, executionProjectSensitive,
       executionBoundProjectDigest, executionCurrentProjectDigest
      >>

DriftProjectBinding ==
  \E es \in {PrimaryScope, DownstreamScope} :
    \E digest \in (ProjectDigests \ {executionBoundProjectDigest[es]}) :
    LET
      propagateDownstream ==
        /\ es # DownstreamScope
        /\ es \in executionDependsOn[DownstreamScope]
        /\ executionState[DownstreamScope] \in {"queued", "planning", "running", "waiting", "blocked"}
      nextExecutionState ==
        [executionState EXCEPT
          ![es] = "blocked",
          ![DownstreamScope] = IF propagateDownstream THEN "blocked" ELSE @]
      nextExecutionWaitKind ==
        [executionWaitKind EXCEPT
          ![es] = "project_blocked",
          ![DownstreamScope] = IF propagateDownstream THEN "project_blocked" ELSE @]
      nextExecutionWaitState ==
        [executionWaitState EXCEPT
          ![es] = "waiting_project_blocked",
          ![DownstreamScope] = IF propagateDownstream THEN "waiting_project_blocked" ELSE @]
      nextExecutionPendingApproval ==
        [executionPendingApproval EXCEPT
          ![es] = NoPendingApproval,
          ![DownstreamScope] = IF propagateDownstream THEN NoPendingApproval ELSE @]
    IN
    /\ runLifecycle[executionRun[es]] \in {"starting", "active", "recovering", "blocked"}
    /\ executionProjectSensitive[es]
    /\ executionCurrentProjectDigest[es] = executionBoundProjectDigest[es]
    /\ executionState[es] \in {"queued", "planning", "running", "waiting"}
    /\ executionCurrentProjectDigest' = [executionCurrentProjectDigest EXCEPT ![es] = digest]
    /\ executionState' = nextExecutionState
    /\ executionWaitKind' = nextExecutionWaitKind
    /\ executionWaitState' = nextExecutionWaitState
    /\ executionPendingApproval' = nextExecutionPendingApproval
    /\ partialBlocked' = [partialBlocked EXCEPT ![executionRun[es]] = TRUE]
    /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runPlan, runLifecycle, noEligibleWork, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalState,
         approvalConsumeCount, consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
         effectiveApprovalState, effectiveRunLifecycle, effectiveRunPlan, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
         stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
         runTerminalAuditState, planSupersessionAuditState, planSuperseded,
         executionRun, executionDependsOn, executionProjectSensitive, executionBoundProjectDigest
        >>

ReconcileProjectBinding ==
  \E es \in ExecutionScopes :
    /\ executionProjectSensitive[es]
    /\ executionState[es] = "blocked"
    /\ executionWaitKind[es] = "project_blocked"
    /\ executionCurrentProjectDigest[es] # executionBoundProjectDigest[es]
    /\ executionCurrentProjectDigest' =
         [executionCurrentProjectDigest EXCEPT ![es] = executionBoundProjectDigest[es]]
    /\ executionState' = [executionState EXCEPT ![es] = "running"]
    /\ executionWaitKind' = [executionWaitKind EXCEPT ![es] = "none"]
    /\ executionWaitState' = [executionWaitState EXCEPT ![es] = "none"]
    /\ executionPendingApproval' = [executionPendingApproval EXCEPT ![es] = NoPendingApproval]
    /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runPlan, runLifecycle, noEligibleWork, partialBlocked, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalState,
         approvalConsumeCount, consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         runnerApprovalMirror, runnerRunLifecycleMirror, runnerPlanMirror,
         effectiveApprovalState, effectiveRunLifecycle, effectiveRunPlan, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
         stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
         runTerminalAuditState, planSupersessionAuditState, planSuperseded,
         executionRun, executionDependsOn, executionProjectSensitive, executionBoundProjectDigest
        >>

RunnerAdvisoryDrift ==
  /\ runnerRunLifecycleMirror[PrimaryRun] = runLifecycle[PrimaryRun]
  /\ runnerApprovalMirror[ExactApproval] = approvalState[ExactApproval]
  /\ runnerPlanMirror[PrimaryRun] = runPlan[PrimaryRun]
  /\ runnerRunLifecycleMirror' = [runnerRunLifecycleMirror EXCEPT ![PrimaryRun] = MirrorDriftRunLifecycle]
  /\ runnerApprovalMirror' = [runnerApprovalMirror EXCEPT ![ExactApproval] = MirrorDriftApprovalState]
  /\ runnerPlanMirror' = [runnerPlanMirror EXCEPT ![PrimaryRun] = SupersedingPlan]
  /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runPlan, runLifecycle, noEligibleWork, partialBlocked, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalState,
         approvalConsumeCount, consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         effectiveApprovalState, effectiveRunLifecycle, effectiveRunPlan, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
         stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
         runTerminalAuditState, planSupersessionAuditState, planSuperseded
         >>
  /\ UNCHANGED executionVars

ReconcileRunnerAdvisoryFromBroker ==
  /\ runnerRunLifecycleMirror[PrimaryRun] # runLifecycle[PrimaryRun]
     \/ runnerApprovalMirror[ExactApproval] # approvalState[ExactApproval]
     \/ runnerPlanMirror[PrimaryRun] # runPlan[PrimaryRun]
  /\ runnerRunLifecycleMirror' = [runnerRunLifecycleMirror EXCEPT ![PrimaryRun] = runLifecycle[PrimaryRun]]
  /\ runnerApprovalMirror' = [runnerApprovalMirror EXCEPT ![ExactApproval] = approvalState[ExactApproval]]
  /\ runnerPlanMirror' = [runnerPlanMirror EXCEPT ![PrimaryRun] = runPlan[PrimaryRun]]
  /\ stepCount' = stepCount + 1
    /\ UNCHANGED <<
         runPlan, runLifecycle, noEligibleWork, partialBlocked, stageSummaryHash, summaryRevision,
         actionReqRun, actionReqPlan, actionReqStage, actionKind, actionReqHash, actionManifestHash,
         actionPolicyInputHash, actionArtifactDigest, actionOverrideAttempt, actionOverriddenFailedRef,
         approvalRun, approvalPlan, approvalStage, approvalBindingKind, approvalBoundAction,
         approvalBoundStageHash, approvalManifestHash, approvalPolicyInputHash, approvalState,
         approvalConsumeCount, consumedByAction,
         gateAttemptRun, gateAttemptPlan, gateAttemptGate, gateState, gateOutcome,
         gateFailedResultRef, gateWasFailedBeforeOverride, brokerGateEvidenceRef,
         effectiveApprovalState, effectiveRunLifecycle, effectiveRunPlan, effectiveGateEvidenceRef,
         approvalRequestAuditState, approvalDecisionAuditState, approvalConsumptionAuditState,
           stageSignoffConsumptionAuditState, gateResultAuditState, gateOverrideAuditState,
           runTerminalAuditState, planSupersessionAuditState, planSuperseded
         >>
  /\ UNCHANGED executionVars

BoundedStutter ==
  /\ stepCount = MaxSteps
  /\ UNCHANGED vars

Next ==
  \/ /\ stepCount < MaxSteps
      /\ (\/ AcceptApprovalDecision
          \/ ExpireOrCancelApproval
          \/ ConsumeApproval
          \/ UpdateStageSummary
          \/ ReportGateAttemptResult
          \/ SupersedePlan
          \/ UpdateRunCoordination
          \/ SetRunTerminal
          \/ EnterExecutionWait
          \/ BlockDependentExecution
          \/ ResolveExecutionWait
          \/ ResumeDependentExecution
          \/ DriftProjectBinding
          \/ ReconcileProjectBinding
          \/ RunnerAdvisoryDrift
          \/ ReconcileRunnerAdvisoryFromBroker)
  \/ BoundedStutter

Spec == Init /\ [][Next]_vars

TypeOK ==
  /\ runPlan \in [Runs -> Plans]
  /\ runLifecycle \in [Runs -> PublicRunLifecycle]
  /\ noEligibleWork \in [Runs -> BOOLEAN]
  /\ partialBlocked \in [Runs -> BOOLEAN]
  /\ stageSummaryHash \in [Runs -> [Plans -> [Stages -> HashTokens]]]
  /\ summaryRevision \in [Runs -> [Plans -> [Stages -> 0..MaxRevision]]]

  /\ actionReqRun \in [Actions -> Runs]
  /\ actionReqPlan \in [Actions -> Plans]
  /\ actionReqStage \in [Actions -> Stages]
  /\ actionKind \in [Actions -> ActionKinds]
  /\ actionReqHash \in [Actions -> HashTokens]
  /\ actionManifestHash \in [Actions -> PolicyHashes]
  /\ actionPolicyInputHash \in [Actions -> PolicyInputHashes]
  /\ actionArtifactDigest \in [Actions -> ArtifactDigests]
  /\ actionOverrideAttempt \in [Actions -> GateAttempts]
  /\ actionOverriddenFailedRef \in [Actions -> (EvidenceRefs \cup {NoEvidence})]

  /\ approvalRun \in [Approvals -> Runs]
  /\ approvalPlan \in [Approvals -> Plans]
  /\ approvalStage \in [Approvals -> Stages]
  /\ approvalBindingKind \in [Approvals -> ApprovalBindingKinds]
  /\ approvalBoundAction \in [Approvals -> Actions]
  /\ approvalBoundStageHash \in [Approvals -> HashTokens]
  /\ approvalManifestHash \in [Approvals -> PolicyHashes]
  /\ approvalPolicyInputHash \in [Approvals -> PolicyInputHashes]
  /\ approvalState \in [Approvals -> ApprovalLifecycle]
  /\ approvalConsumeCount \in [Approvals -> 0..1]
  /\ consumedByAction \in [Approvals -> SUBSET Actions]

  /\ gateAttemptRun \in [GateAttempts -> Runs]
  /\ gateAttemptPlan \in [GateAttempts -> Plans]
  /\ gateAttemptGate \in [GateAttempts -> Gates]
  /\ gateState \in [GateAttempts -> GateLifecycle]
  /\ gateOutcome \in [GateAttempts -> GateAttemptOutcomes]
  /\ gateFailedResultRef \in [GateAttempts -> (EvidenceRefs \cup {NoEvidence})]
  /\ gateWasFailedBeforeOverride \in [GateAttempts -> BOOLEAN]
  /\ brokerGateEvidenceRef \in [GateAttempts -> (EvidenceRefs \cup {NoEvidence})]

  /\ runnerApprovalMirror \in [Approvals -> ApprovalLifecycle]
  /\ runnerRunLifecycleMirror \in [Runs -> PublicRunLifecycle]
  /\ runnerPlanMirror \in [Runs -> Plans]

  /\ effectiveApprovalState \in [Approvals -> ApprovalLifecycle]
  /\ effectiveRunLifecycle \in [Runs -> PublicRunLifecycle]
  /\ effectiveRunPlan \in [Runs -> Plans]
  /\ effectiveGateEvidenceRef \in [GateAttempts -> (EvidenceRefs \cup {NoEvidence})]

  /\ approvalRequestAuditState \in [Approvals -> AuditObligationStates]
  /\ approvalDecisionAuditState \in [Approvals -> AuditObligationStates]
  /\ approvalConsumptionAuditState \in [Approvals -> AuditObligationStates]
  /\ stageSignoffConsumptionAuditState \in [Approvals -> AuditObligationStates]
  /\ gateResultAuditState \in [GateAttempts -> AuditObligationStates]
  /\ gateOverrideAuditState \in [Approvals -> AuditObligationStates]
  /\ runTerminalAuditState \in [Runs -> AuditObligationStates]
  /\ planSupersessionAuditState \in [Runs -> AuditObligationStates]
  /\ planSuperseded \in [Runs -> BOOLEAN]

  /\ executionRun \in [ExecutionScopes -> Runs]
  /\ executionState \in [ExecutionScopes -> ExecutionLifecycle]
  /\ executionWaitKind \in [ExecutionScopes -> WaitKinds]
  /\ executionWaitState \in [ExecutionScopes -> WaitStates]
  /\ executionDependsOn \in [ExecutionScopes -> SUBSET ExecutionScopes]
  /\ executionProjectSensitive \in [ExecutionScopes -> BOOLEAN]
  /\ executionBoundProjectDigest \in [ExecutionScopes -> ProjectDigests]
  /\ executionCurrentProjectDigest \in [ExecutionScopes -> ProjectDigests]
  /\ executionPendingApproval \in [ExecutionScopes -> (Approvals \cup {NoPendingApproval})]

  /\ stepCount \in 0..MaxSteps

ApprovalScopeIsolation ==
  \A a \in Approvals :
    \A act \in consumedByAction[a] :
      /\ actionReqRun[act] = approvalRun[a]
      /\ actionReqPlan[act] = approvalPlan[a]

SingleUseApprovalConsumption ==
  \A a \in Approvals :
    /\ approvalConsumeCount[a] \in 0..1
    /\ Cardinality(consumedByAction[a]) <= 1

StageSignoffSupersession ==
  \A a \in Approvals :
    approvalBindingKind[a] = "stage_sign_off"
      /\ approvalState[a] \in {"pending", "approved"}
      => approvalBoundStageHash[a] =
           stageSummaryHash[approvalRun[a]][approvalPlan[a]][approvalStage[a]]

BrokerWinsAuthority ==
  /\ effectiveApprovalState = approvalState
  /\ effectiveRunLifecycle = runLifecycle
  /\ effectiveRunPlan = runPlan
  /\ effectiveGateEvidenceRef = brokerGateEvidenceRef

CorrectFailedGateToOverrideLinkage ==
  \A a \in Approvals :
    \A act \in consumedByAction[a] :
      actionKind[act] = "gate_override"
        => LET ga == actionOverrideAttempt[act] IN
           /\ gateWasFailedBeforeOverride[ga]
           /\ actionOverriddenFailedRef[act] = gateFailedResultRef[ga]

PartialBlockingSeparateFromPublicLifecycle ==
  /\ "partially_blocked" \notin PublicRunLifecycle
  /\ \A r \in Runs : runLifecycle[r] = "blocked" => noEligibleWork[r]
  /\ \A r \in Runs : partialBlocked[r] => runLifecycle[r] \in
        {"starting", "active", "recovering", "blocked"}

ExecutionWaitVocabularySeparation ==
  /\ "waiting_approval" \notin PublicRunLifecycle
  /\ "waiting_operator_input" \notin PublicRunLifecycle
  /\ \A es \in ExecutionScopes :
       executionWaitState[es] = WaitStateFor(executionWaitKind[es])
  /\ \A es \in ExecutionScopes :
       executionState[es] = "waiting"
         => executionWaitKind[es] \in {"operator_input", "approval", "external_dependency"}
  /\ \A es \in ExecutionScopes :
       executionWaitKind[es] = "project_blocked"
         => executionState[es] = "blocked"

ExecutionApprovalWaitBinding ==
  \A es \in ExecutionScopes :
    executionWaitKind[es] = "approval"
      => /\ executionPendingApproval[es] # NoPendingApproval
         /\ approvalRun[executionPendingApproval[es]] = executionRun[es]
         /\ executionWaitState[es] = "waiting_approval"

DependencyAwarePartialBlocking ==
  \A es \in ExecutionScopes :
    executionState[es] = "blocked" /\ executionWaitKind[es] # "project_blocked"
      => /\ executionDependsOn[es] # {}
         /\ \E dep \in executionDependsOn[es] :
              /\ executionState[dep] = "waiting"
              /\ executionWaitKind[dep] = executionWaitKind[es]
              /\ executionWaitState[dep] = executionWaitState[es]

ProjectBindingDriftFailsClosed ==
  \A es \in ExecutionScopes :
    executionProjectSensitive[es] /\
    executionCurrentProjectDigest[es] # executionBoundProjectDigest[es]
      => /\ executionState[es] = "blocked"
         /\ executionWaitKind[es] = "project_blocked"
         /\ executionWaitState[es] = "waiting_project_blocked"

AuditObligationsSatisfied ==
  /\ \A a \in Approvals : approvalRequestAuditState[a] = "required_recorded"
  /\ \A a \in Approvals :
       approvalState[a] \in {"approved", "denied", "consumed"}
         => approvalDecisionAuditState[a] = "required_recorded"
  /\ \A a \in Approvals :
       approvalState[a] = "consumed"
         => approvalConsumptionAuditState[a] = "required_recorded"
  /\ \A a \in Approvals :
       approvalState[a] = "consumed" /\ approvalBindingKind[a] = "stage_sign_off"
         => stageSignoffConsumptionAuditState[a] = "required_recorded"
  /\ \A ga \in GateAttempts :
       gateState[ga] \in {"passed", "failed", "overridden"}
         => gateResultAuditState[ga] = "required_recorded"
  /\ \A a \in Approvals :
       \A act \in consumedByAction[a] :
         actionKind[act] = "gate_override"
           => gateOverrideAuditState[a] = "required_recorded"
  /\ \A r \in Runs :
       runLifecycle[r] \in TerminalRunLifecycle
         => runTerminalAuditState[r] = "required_recorded"
  /\ \A r \in Runs :
       planSuperseded[r]
         => planSupersessionAuditState[r] = "required_recorded"

=============================================================================
\* Traceability anchors (high-level):
\* - Approval lifecycle + binding separation: CHG-2026-007, CHG-2026-008, security/approval-binding-and-verifier-identity
\* - Runner advisory vs broker authority: CHG-2026-033, security/runner-durable-state-and-replay
\* - Gate evidence authority + override linkage: CHG-2026-035, security/trusted-runtime-evidence-and-broker-projection
\* - Manifest/policy input hash binding: CHG-2026-007, security/policy-evaluation-foundations
\* - Public lifecycle + partial blocking split: CHG-2026-012, CHG-2026-033
\* - Execution wait vocabulary + dependency-aware partial blocking: CHG-2026-048, CHG-2026-050, global/session-execution-contract-and-watch-families
\* - Project-substrate digest binding + fail-closed drift: CHG-2026-046, CHG-2026-048, global/project-substrate-contract-and-lifecycle
=============================================================================

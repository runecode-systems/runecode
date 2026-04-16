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
  MaxRevision,
  MaxSteps

ApprovalLifecycle ==
  {"pending", "approved", "denied", "expired", "cancelled", "superseded", "consumed"}

ApprovalTerminalStates == {"consumed", "denied", "expired", "cancelled", "superseded"}

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
  /\ MaxRevision \in Nat
  /\ MaxSteps \in Nat
  /\ NoEvidence \notin EvidenceRefs
  /\ Cardinality(Runs) >= 2
  /\ Cardinality(Plans) >= 2
  /\ Cardinality(Stages) >= 1
  /\ Cardinality(Actions) >= 3
  /\ Cardinality(Approvals) >= 3
  /\ Cardinality(GateAttempts) >= 1
  /\ Cardinality(HashTokens) >= 2
  /\ Cardinality(ArtifactDigests) >= 1
  /\ Cardinality(PolicyHashes) >= 1
  /\ Cardinality(PolicyInputHashes) >= 1
  /\ Cardinality(EvidenceRefs) >= 2

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

DeniedApproval == CHOOSE a \in (Approvals \ {ExactApproval, StageApproval}) : TRUE

PrimaryGateAttempt == CHOOSE ga \in GateAttempts : TRUE

PrimaryEvidence == CHOOSE e \in EvidenceRefs : TRUE

SecondaryEvidence == CHOOSE e \in (EvidenceRefs \ {PrimaryEvidence}) : TRUE

PrimaryHash == CHOOSE h \in HashTokens : TRUE

SecondaryHash == CHOOSE h \in (HashTokens \ {PrimaryHash}) : TRUE

PrimaryArtifactDigest == CHOOSE d \in ArtifactDigests : TRUE

PrimaryPolicyHash == CHOOSE h \in PolicyHashes : TRUE

PrimaryPolicyInputHash == CHOOSE h \in PolicyInputHashes : TRUE

MirrorDriftApprovalState == "denied"

MirrorDriftRunLifecycle == "blocked"

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

  \* Bounded trace length for deterministic TLC runs.
  stepCount

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
  runTerminalAuditState, planSupersessionAuditState, planSuperseded, stepCount
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
  \E a \in {ExactApproval, StageApproval, DeniedApproval}, decision \in {"approved", "denied"} :
    /\ approvalState[a] = "pending"
    /\ (a = DeniedApproval => decision = "denied")
    /\ (a # DeniedApproval => decision = "approved")
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

ConsumeApproval ==
  \E a \in {ExactApproval, StageApproval}, act \in {ExactAction, StageAction, OverrideAction} :
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

UpdateRunCoordination ==
  \E pblocked \in BOOLEAN, noWork \in BOOLEAN,
     nextState \in {"active", "blocked", "recovering"} :
    LET
      r == PrimaryRun
      nextLifecycle == [runLifecycle EXCEPT ![r] = nextState]
    IN
    /\ nextState = "blocked" => noWork
    /\ pblocked => nextState \in {"starting", "active", "recovering", "blocked"}
    /\ partialBlocked' = [partialBlocked EXCEPT ![r] = pblocked]
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
=============================================================================

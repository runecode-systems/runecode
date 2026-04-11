package artifacts

func copyRunnerAdvisoryState(in RunnerAdvisoryState) RunnerAdvisoryState {
	out := RunnerAdvisoryState{}
	out.LastCheckpoint = cloneCheckpoint(in.LastCheckpoint)
	out.LastResult = cloneResult(in.LastResult)
	out.Lifecycle = cloneRunnerLifecycleHint(in.Lifecycle)
	out.StepAttempts = copyRunnerStepHints(in.StepAttempts)
	out.GateAttempts = copyRunnerGateHints(in.GateAttempts)
	out.ApprovalWaits = copyRunnerApprovals(in.ApprovalWaits)
	return out
}

func cloneRunnerLifecycleHint(in *RunnerLifecycleHint) *RunnerLifecycleHint {
	if in == nil {
		return nil
	}
	copyLifecycle := *in
	return &copyLifecycle
}

func copyRunnerStepHints(in map[string]RunnerStepHint) map[string]RunnerStepHint {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]RunnerStepHint, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyRunnerGateHints(in map[string]RunnerGateHint) map[string]RunnerGateHint {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]RunnerGateHint, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyRunnerApprovals(in map[string]RunnerApproval) map[string]RunnerApproval {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]RunnerApproval, len(in))
	for k, v := range in {
		out[k] = copyRunnerApproval(v)
	}
	return out
}

func copyRunnerApproval(in RunnerApproval) RunnerApproval {
	out := in
	if in.ResolvedAt != nil {
		t := *in.ResolvedAt
		out.ResolvedAt = &t
	}
	return out
}

func cloneCheckpoint(in *RunnerCheckpointAdvisory) *RunnerCheckpointAdvisory {
	if in == nil {
		return nil
	}
	out := *in
	if in.Details != nil {
		out.Details = copyMap(in.Details)
	}
	return &out
}

func cloneResult(in *RunnerResultAdvisory) *RunnerResultAdvisory {
	if in == nil {
		return nil
	}
	out := *in
	if in.Details != nil {
		out.Details = copyMap(in.Details)
	}
	return &out
}

func cloneRunnerApproval(in *RunnerApproval) *RunnerApproval {
	if in == nil {
		return nil
	}
	out := *in
	if in.ResolvedAt != nil {
		t := *in.ResolvedAt
		out.ResolvedAt = &t
	}
	return &out
}

func copyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = copyAny(value)
	}
	return out
}

func copyAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return copyMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = copyAny(typed[i])
		}
		return out
	default:
		return typed
	}
}

func copyRunnerAdvisoryByRun(in map[string]RunnerAdvisoryState) map[string]RunnerAdvisoryState {
	out := make(map[string]RunnerAdvisoryState, len(in))
	for key, value := range in {
		out[key] = copyRunnerAdvisoryState(value)
	}
	return out
}

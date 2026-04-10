package launcherbackend

import (
	"fmt"
	"strings"
)

func (e BackendCacheEvidence) Normalized() BackendCacheEvidence {
	out := e
	out.ImageCacheResult = normalizeCacheResult(out.ImageCacheResult)
	out.BootArtifactCacheResult = normalizeCacheResult(out.BootArtifactCacheResult)
	out.ResolvedImageDescriptorDigest = strings.TrimSpace(out.ResolvedImageDescriptorDigest)
	out.ResolvedBootComponentDigests = uniqueSortedStrings(out.ResolvedBootComponentDigests)
	return out
}

func (e BackendCacheEvidence) Validate() error {
	normalized := e.Normalized()
	if normalized.ImageCacheResult == "" {
		return fmt.Errorf("image_cache_result must be one of %q, %q, or %q", CacheResultHit, CacheResultMiss, CacheResultBypass)
	}
	if normalized.BootArtifactCacheResult == "" {
		return fmt.Errorf("boot_artifact_cache_result must be one of %q, %q, or %q", CacheResultHit, CacheResultMiss, CacheResultBypass)
	}
	if !looksLikeDigest(normalized.ResolvedImageDescriptorDigest) {
		return fmt.Errorf("resolved_image_descriptor_digest must be sha256:<64 lowercase hex>")
	}
	for _, digest := range normalized.ResolvedBootComponentDigests {
		if !looksLikeDigest(digest) {
			return fmt.Errorf("resolved_boot_component_digests values must be sha256:<64 lowercase hex>")
		}
	}
	return nil
}

func (s BackendLifecycleSnapshot) Normalized() BackendLifecycleSnapshot {
	out := s
	out.CurrentState = normalizeBackendLifecycleState(out.CurrentState)
	out.PreviousState = normalizeBackendLifecycleState(out.PreviousState)
	if out.TransitionCount < 0 {
		out.TransitionCount = 0
	}
	return out
}

func (s BackendLifecycleSnapshot) Validate() error {
	normalized := s.Normalized()
	if normalized.CurrentState == "" {
		return fmt.Errorf("current_state must be set")
	}
	if !normalized.TerminateBetweenSteps {
		return fmt.Errorf("terminate_between_steps must be true")
	}
	if normalized.PreviousState != "" {
		if err := ValidateBackendLifecycleTransition(normalized.PreviousState, normalized.CurrentState); err != nil {
			return err
		}
	}
	return nil
}

func ValidateBackendLifecycleTransition(fromState string, toState string) error {
	from := normalizeBackendLifecycleState(fromState)
	to := normalizeBackendLifecycleState(toState)
	if from == "" {
		return fmt.Errorf("from_state must be a known backend lifecycle state")
	}
	if to == "" {
		return fmt.Errorf("to_state must be a known backend lifecycle state")
	}
	if from == to {
		return nil
	}
	allowed := map[string][]string{
		BackendLifecycleStatePlanned:     {BackendLifecycleStateLaunching, BackendLifecycleStateTerminating},
		BackendLifecycleStateLaunching:   {BackendLifecycleStateStarted, BackendLifecycleStateTerminating},
		BackendLifecycleStateStarted:     {BackendLifecycleStateBinding, BackendLifecycleStateTerminating},
		BackendLifecycleStateBinding:     {BackendLifecycleStateActive, BackendLifecycleStateTerminating},
		BackendLifecycleStateActive:      {BackendLifecycleStateTerminating},
		BackendLifecycleStateTerminating: {BackendLifecycleStateTerminated},
		BackendLifecycleStateTerminated:  {},
	}
	next := allowed[from]
	for _, candidate := range next {
		if candidate == to {
			return nil
		}
	}
	return fmt.Errorf("invalid backend lifecycle transition %q -> %q", from, to)
}

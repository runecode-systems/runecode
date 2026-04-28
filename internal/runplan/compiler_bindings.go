package runplan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/runecode-ai/runecode/internal/policyengine"
)

func validateBindingsAgainstTrustedRegistry(bindings []ExecutorBinding, registry policyengine.ExecutorRegistryProjection) error {
	byID := trustedExecutorsByID(registry)
	for _, binding := range bindings {
		if err := validateBindingAgainstTrustedRegistry(binding, byID); err != nil {
			return err
		}
	}
	return nil
}

func trustedExecutorsByID(registry policyengine.ExecutorRegistryProjection) map[string]policyengine.ExecutorProjectionRecord {
	byID := map[string]policyengine.ExecutorProjectionRecord{}
	for _, rec := range registry.Executors {
		byID[rec.ExecutorID] = rec
	}
	return byID
}

func validateBindingAgainstTrustedRegistry(binding ExecutorBinding, trustedByID map[string]policyengine.ExecutorProjectionRecord) error {
	rec, ok := trustedByID[binding.ExecutorID]
	if !ok {
		return fmt.Errorf("executor_binding %q references unknown trusted executor_id %q", binding.BindingID, binding.ExecutorID)
	}
	if rec.ExecutorClass != binding.ExecutorClass {
		return fmt.Errorf("executor_binding %q class %q does not match trusted executor class %q", binding.BindingID, binding.ExecutorClass, rec.ExecutorClass)
	}
	return validateBindingRolesAllowed(binding, rec.AllowedRoles)
}

func validateBindingRolesAllowed(binding ExecutorBinding, allowedRoles []string) error {
	allowed := toSet(allowedRoles)
	for _, roleKind := range binding.AllowedRoleKinds {
		if _, ok := allowed[roleKind]; !ok {
			return fmt.Errorf("executor_binding %q role_kind %q not allowed by trusted executor registry", binding.BindingID, roleKind)
		}
	}
	return nil
}

func compileExecutorBindings(bindings []ExecutorBinding) ([]ExecutorBinding, error) {
	merged := map[string]ExecutorBinding{}
	for _, binding := range bindings {
		if err := mergeExecutorBindingRecord(merged, binding); err != nil {
			return nil, err
		}
	}
	return sortedExecutorBindings(merged), nil
}

func mergeExecutorBindingRecord(merged map[string]ExecutorBinding, binding ExecutorBinding) error {
	if strings.TrimSpace(binding.BindingID) == "" {
		return fmt.Errorf("executor binding_id is required")
	}
	existing, seen := merged[binding.BindingID]
	if !seen {
		merged[binding.BindingID] = normalizedExecutorBinding(binding)
		return nil
	}
	if existing.ExecutorID != binding.ExecutorID || existing.ExecutorClass != binding.ExecutorClass {
		return fmt.Errorf("executor binding %q conflicts within process definition", binding.BindingID)
	}
	merged[binding.BindingID] = mergedExecutorBinding(existing, binding)
	return nil
}

func normalizedExecutorBinding(binding ExecutorBinding) ExecutorBinding {
	copyBinding := binding
	copyBinding.AllowedRoleKinds = sortedUniqueStrings(binding.AllowedRoleKinds)
	return copyBinding
}

func mergedExecutorBinding(existing ExecutorBinding, incoming ExecutorBinding) ExecutorBinding {
	combinedRoles := append(existing.AllowedRoleKinds, incoming.AllowedRoleKinds...)
	existing.AllowedRoleKinds = sortedUniqueStrings(combinedRoles)
	if existing.Description == "" {
		existing.Description = incoming.Description
	}
	return existing
}

func sortedExecutorBindings(merged map[string]ExecutorBinding) []ExecutorBinding {
	bindingIDs := make([]string, 0, len(merged))
	for bindingID := range merged {
		bindingIDs = append(bindingIDs, bindingID)
	}
	sort.Strings(bindingIDs)
	out := make([]ExecutorBinding, 0, len(bindingIDs))
	for _, bindingID := range bindingIDs {
		out = append(out, merged[bindingID])
	}
	return out
}

func toSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func sortedUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	set := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

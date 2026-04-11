package policyengine

import (
	"fmt"
	"path/filepath"
	"strings"
)

func validateExecutorArgvShape(argv []string, contract typedExecutorContract) error {
	if len(argv) == 0 {
		return fmt.Errorf("argv must not be empty")
	}
	if hasSudoToken(argv) {
		return fmt.Errorf("executor contract forbids privilege-escalation launcher")
	}
	base := argv
	if contract.AllowEnvWrapper {
		base = unwrapLauncherArgv(argv)
		if len(base) == 0 {
			return fmt.Errorf("argv wrapper chain does not resolve to concrete executable")
		}
	}
	invoked := strings.ToLower(filepath.Base(base[0]))
	expected := strings.ToLower(filepath.Base(contract.ID))
	if expected != "workspace-runner" && invoked != expected {
		return fmt.Errorf("argv executable %q does not match executor_id %q", invoked, contract.ID)
	}
	if err := validateExecutorArgvHead(base, contract); err != nil {
		return err
	}
	if isExecutorRawShell(contract.ID, argv) {
		return fmt.Errorf("executor contract forbids raw shell passthrough")
	}
	if hasCommandStringPassthrough(base) {
		return fmt.Errorf("executor contract forbids command-string passthrough")
	}
	return nil
}

func validateExecutorArgvHead(argv []string, contract typedExecutorContract) error {
	if len(contract.AllowedArgvHeads) == 0 {
		return nil
	}
	for _, head := range contract.AllowedArgvHeads {
		if argvHeadMatches(argv, head) {
			return nil
		}
	}
	return fmt.Errorf("argv does not match reviewed operation heads for executor_id %q", contract.ID)
}

func validateExecutorEnvironmentShape(environment map[string]string, contract typedExecutorContract) error {
	if err := validateExecutorEnvironmentPresence(environment, contract); err != nil {
		return err
	}
	for key, value := range environment {
		if err := validateEnvironmentEntry(key, value, contract); err != nil {
			return err
		}
	}
	return nil
}

func hasCommandStringPassthrough(argv []string) bool {
	if len(argv) < 2 {
		return false
	}
	exec := normalizedExecutableName(argv[0])
	tokenSet := map[string]struct{}{}
	for i := 1; i < len(argv); i++ {
		tokenSet[strings.ToLower(strings.TrimSpace(argv[i]))] = struct{}{}
	}
	return executorAllowsCommandString(exec, tokenSet)
}

func hasSudoToken(argv []string) bool {
	for _, token := range argv {
		if strings.EqualFold(strings.TrimSpace(token), "sudo") {
			return true
		}
	}
	return false
}

func isExecutorRawShell(executorID string, argv []string) bool {
	return isRawShellInvocation(executorRunPayload{ExecutorID: executorID, Argv: argv})
}

func argvHeadMatches(argv []string, head []string) bool {
	if len(argv) < len(head) {
		return false
	}
	for i := range head {
		if !strings.EqualFold(argv[i], head[i]) {
			return false
		}
	}
	return true
}

func validateExecutorEnvironmentPresence(environment map[string]string, contract typedExecutorContract) error {
	if len(environment) > 0 || contract.AllowEmptyEnv {
		return nil
	}
	return fmt.Errorf("environment must not be empty")
}

func validateEnvironmentEntry(key, value string, contract typedExecutorContract) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("environment key must not be empty")
	}
	if strings.Contains(key, "=") {
		return fmt.Errorf("environment key %q must not include '='", key)
	}
	if strings.Contains(value, "\x00") {
		return fmt.Errorf("environment value for %q contains NUL byte", key)
	}
	if _, ok := contract.AllowedEnvKeys[key]; !ok {
		return fmt.Errorf("environment key %q is not allowed", key)
	}
	return nil
}

func normalizedExecutableName(token string) string {
	return strings.ToLower(filepath.Base(token))
}

func executorAllowsCommandString(exec string, tokenSet map[string]struct{}) bool {
	switch exec {
	case "python", "python3":
		_, short := tokenSet["-c"]
		return short
	case "node":
		_, short := tokenSet["-e"]
		_, long := tokenSet["--eval"]
		return short || long
	case "powershell", "pwsh":
		_, alias := tokenSet["-c"]
		_, short := tokenSet["-command"]
		_, long := tokenSet["--command"]
		return alias || short || long
	case "cmd", "cmd.exe":
		_, short := tokenSet["/c"]
		return short
	default:
		return false
	}
}

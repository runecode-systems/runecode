package brokerapi

import (
	"os"
	"runtime"
	"strings"
)

func sessionExecutionRunnerEnv(protocolSchemasRoot, stateRoot string) []string {
	env := sessionExecutionRunnerBaseEnv(runtime.GOOS, stateRoot, os.LookupEnv)
	env = append(env,
		"LANG=C",
		"LC_ALL=C",
		"RUNECODE_PROTOCOL_SCHEMAS_ROOT="+protocolSchemasRoot,
	)
	return env
}

func sessionExecutionRunnerBaseEnv(goos, stateRoot string, lookupEnv func(string) (string, bool)) []string {
	env := []string{"PATH=" + firstNonEmpty(lookupRunnerEnvValue(lookupEnv, "PATH", "Path"), defaultSessionExecutionRunnerPath(goos))}
	for _, variable := range sessionExecutionRunnerOptionalEnvVars(goos, stateRoot) {
		value := strings.TrimSpace(variable.value)
		if value == "" {
			value = strings.TrimSpace(lookupRunnerEnvValue(lookupEnv, variable.aliases...))
		}
		if value != "" {
			env = append(env, variable.key+"="+value)
		}
	}
	return env
}

type sessionExecutionRunnerEnvVar struct {
	key     string
	value   string
	aliases []string
}

func sessionExecutionRunnerOptionalEnvVars(goos, stateRoot string) []sessionExecutionRunnerEnvVar {
	vars := []sessionExecutionRunnerEnvVar{{key: "HOME", value: stateRoot}, {key: "TMPDIR", value: stateRoot}, {key: "TEMP", value: stateRoot}, {key: "TMP", value: stateRoot}}
	if goos != "windows" {
		return vars
	}
	roamingRoot := strings.TrimRight(stateRoot, `\/`) + `\AppData\Roaming`
	localRoot := strings.TrimRight(stateRoot, `\/`) + `\AppData\Local`
	return append(vars,
		sessionExecutionRunnerEnvVar{key: "USERPROFILE", value: stateRoot},
		sessionExecutionRunnerEnvVar{key: "APPDATA", value: roamingRoot},
		sessionExecutionRunnerEnvVar{key: "LOCALAPPDATA", value: localRoot},
		sessionExecutionRunnerEnvVar{key: "SystemRoot", aliases: []string{"SystemRoot", "SYSTEMROOT", "windir", "WINDIR"}},
		sessionExecutionRunnerEnvVar{key: "ComSpec", aliases: []string{"ComSpec", "COMSPEC"}},
		sessionExecutionRunnerEnvVar{key: "PATHEXT", aliases: []string{"PATHEXT"}},
	)
}

func lookupRunnerEnvValue(lookupEnv func(string) (string, bool), keys ...string) string {
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if value, ok := lookupEnv(key); ok {
			return value
		}
	}
	return ""
}

func defaultSessionExecutionRunnerPath(goos string) string {
	if goos == "windows" {
		return `C:\Windows\System32;C:\Windows`
	}
	return "/usr/bin:/bin"
}

package main

import (
	"fmt"
	"strings"
)

func selectedModelConfigs(mode string) ([]string, error) {
	switch strings.TrimSpace(mode) {
	case "", "all":
		return modelConfigs, nil
	case "core":
		return []string{"SecurityKernelV0.core.cfg"}, nil
	case "replay":
		return []string{"SecurityKernelV0.replay.cfg"}, nil
	default:
		return nil, usageError{err: fmt.Errorf("unsupported mode %q", mode)}
	}
}

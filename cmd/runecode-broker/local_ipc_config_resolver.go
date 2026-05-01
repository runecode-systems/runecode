package main

import (
	"sync"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

var (
	defaultLocalIPCConfigMu sync.RWMutex
	defaultLocalIPCConfigFn = brokerapi.DefaultLocalIPCConfig
)

func loadDefaultLocalIPCConfig() (brokerapi.LocalIPCConfig, error) {
	defaultLocalIPCConfigMu.RLock()
	resolver := defaultLocalIPCConfigFn
	defaultLocalIPCConfigMu.RUnlock()
	return resolver()
}

func setDefaultLocalIPCConfigForTest(t interface{ Cleanup(func()) }, resolver func() (brokerapi.LocalIPCConfig, error)) {
	defaultLocalIPCConfigMu.Lock()
	prev := defaultLocalIPCConfigFn
	defaultLocalIPCConfigFn = resolver
	defaultLocalIPCConfigMu.Unlock()
	t.Cleanup(func() {
		defaultLocalIPCConfigMu.Lock()
		defaultLocalIPCConfigFn = prev
		defaultLocalIPCConfigMu.Unlock()
	})
}
